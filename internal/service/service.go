// Package service содержит бизнес-правила. SQL здесь нет, он в repo.
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"moneykeeper/internal/i18n"
	"moneykeeper/internal/models"
	"moneykeeper/internal/pgclient"
	"moneykeeper/internal/repo"
	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

const (
	maxNameRunes   = 40
	maxAmountMinor = 1_000_000_000_000
)

type Service struct {
	db           *pgclient.Client
	users        *repo.Users
	budgets      *repo.Budgets
	accounts     *repo.Accounts
	categories   *repo.Categories
	transactions *repo.Transactions
	plan         *repo.Plan
	schedules    *repo.Schedules
	rateStore    *repo.Rates
	fx           RateSource
	log          *slog.Logger
	now          func() time.Time
}

func New(db *pgclient.Client, users *repo.Users, budgets *repo.Budgets, accounts *repo.Accounts,
	categories *repo.Categories, transactions *repo.Transactions, plan *repo.Plan, schedules *repo.Schedules, rateStore *repo.Rates, fx RateSource, log *slog.Logger) *Service {
	return &Service{
		db: db, users: users, budgets: budgets, accounts: accounts,
		categories: categories, transactions: transactions,
		plan: plan, schedules: schedules, rateStore: rateStore, fx: fx, log: log, now: time.Now,
	}
}

// Resolve находит пользователя по Telegram-аккаунту. При первом входе создаёт пользователя,
// личный бюджет и стартовые категории одной транзакцией. Часовой пояс сохраняется только при создании.
func (s *Service) Resolve(ctx context.Context, tg models.TelegramUser, timezone string) (models.User, error) {
	u, err := s.users.GetByTelegramID(ctx, tg.ID)
	if err == nil {
		return u, nil
	}
	if !errors.Is(err, models.ErrNotFound) {
		return models.User{}, err
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return models.User{}, fmt.Errorf("%q: %w", timezone, models.ErrInvalidTimezone)
	}

	err = s.db.InTx(ctx, func(ctx context.Context) error {
		created, err := s.users.CreateIfAbsent(ctx, tg.ID, tg.FirstName, timezone)
		if errors.Is(err, models.ErrNotFound) {
			return nil // параллельный запрос успел создать пользователя
		}
		if err != nil {
			return err
		}
		// Имя личного бюджета пустое: подпись для него подставляет интерфейс через i18n.
		b, err := s.budgets.Create(ctx, "")
		if err != nil {
			return err
		}
		if err := s.budgets.AddMember(ctx, b.ID, created.ID); err != nil {
			return err
		}
		if err := s.categories.SeedDefaults(ctx, b.ID, i18n.DefaultGroups(), i18n.DefaultCategories()); err != nil {
			return err
		}
		return s.users.SetActiveBudget(ctx, created.ID, b.ID)
	})
	if err != nil {
		return models.User{}, err
	}
	return s.users.GetByTelegramID(ctx, tg.ID)
}

func (s *Service) Budget(ctx context.Context, u models.User) (models.Budget, error) {
	return s.budgets.Get(ctx, u.ActiveBudgetID)
}

func (s *Service) SetBaseCurrency(ctx context.Context, u models.User, code string) (models.Budget, error) {
	c, err := money.ParseBase(code)
	if err != nil {
		return models.Budget{}, err
	}
	if err := s.budgets.SetBaseCurrency(ctx, u.ActiveBudgetID, c); err != nil {
		return models.Budget{}, err
	}
	return s.budgets.Get(ctx, u.ActiveBudgetID)
}

// ListAccounts: in задаёт валюту, в которую пересчитываются суммы, пусто значит базовая валюта бюджета.
func (s *Service) ListAccounts(ctx context.Context, u models.User, in string) ([]models.Account, error) {
	accs, err := s.accounts.List(ctx, u.ActiveBudgetID)
	if err != nil {
		return nil, err
	}
	if err := s.priceAccounts(ctx, u, accs, in); err != nil {
		return nil, err
	}
	return accs, nil
}

// CreateAccount заводит счёт. Положительный начальный остаток пишется доходом на сегодняшний день пользователя.
func (s *Service) CreateAccount(ctx context.Context, u models.User, name, kind, currency string, multiCurrency bool, openingMinor int64) (models.Account, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > maxNameRunes {
		return models.Account{}, fmt.Errorf("name: %w", models.ErrInvalidAccount)
	}
	k := models.AccountKind(kind)
	switch k {
	case models.AccountCash, models.AccountDebit, models.AccountCredit, models.AccountWallet:
	default:
		return models.Account{}, fmt.Errorf("kind %q: %w", kind, models.ErrInvalidAccount)
	}
	c, err := money.Parse(currency)
	if err != nil {
		return models.Account{}, err
	}
	if openingMinor < 0 || openingMinor > maxAmountMinor {
		return models.Account{}, fmt.Errorf("opening balance: %w", models.ErrInvalidAccount)
	}

	var acc models.Account
	err = s.db.InTx(ctx, func(ctx context.Context) error {
		var err error
		acc, err = s.accounts.Create(ctx, u.ActiveBudgetID, models.NewAccount{Name: name, Kind: k, Currency: c, MultiCurrency: multiCurrency})
		if err != nil {
			return err
		}
		if openingMinor == 0 {
			return nil
		}
		_, err = s.transactions.Insert(ctx, models.Transaction{
			BudgetID: u.ActiveBudgetID, AccountID: acc.ID, Kind: models.KindOpening,
			AmountMinor: openingMinor, Currency: c, OccurredOn: s.today(u), CreatedBy: u.ID,
		})
		acc.Balances = []models.CurrencyBalance{{Currency: c, BalanceMinor: openingMinor}}
		return err
	})
	if err != nil {
		return models.Account{}, err
	}
	accs := []models.Account{acc}
	if err := s.priceAccounts(ctx, u, accs, ""); err != nil {
		return models.Account{}, err
	}
	return accs[0], nil
}

// ArchiveAccount закрывает счёт: он пропадает из списков, записи по нему остаются в истории. Закрыть можно только пустой
// счёт (остатки по всем валютам нулевые) и не основной.
func (s *Service) ArchiveAccount(ctx context.Context, u models.User, id int64) error {
	acc, err := s.accounts.Get(ctx, u.ActiveBudgetID, id)
	if err != nil {
		return err
	}
	if acc.IsDefault {
		return fmt.Errorf("default account: %w", models.ErrInvalidAccount)
	}
	for _, b := range acc.Balances {
		if b.BalanceMinor != 0 {
			return fmt.Errorf("account %d is not empty: %w", id, models.ErrInvalidAccount)
		}
	}
	return s.accounts.Archive(ctx, u.ActiveBudgetID, id)
}

// today считает сегодняшний день в поясе пользователя.
func (s *Service) today(u models.User) apptime.Day {
	loc, err := time.LoadLocation(u.Timezone)
	if err != nil {
		loc = time.UTC
	}
	return apptime.Today(loc, s.now())
}
