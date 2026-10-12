package repo

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"

	"moneykeeper/internal/models"
	"moneykeeper/internal/pgclient"
	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

type Accounts struct{ db *pgclient.Client }

func NewAccounts(db *pgclient.Client) *Accounts { return &Accounts{db: db} }

// Create делает первый счёт бюджета основным.
func (r *Accounts) Create(ctx context.Context, budgetID int64, in models.NewAccount) (models.Account, error) {
	a := models.Account{BudgetID: budgetID, Name: in.Name, Kind: in.Kind, Currency: in.Currency, MultiCurrency: in.MultiCurrency}
	err := r.db.Q(ctx).QueryRow(ctx,
		`INSERT INTO accounts (budget_id, name, kind, currency, multi_currency, is_default)
		 VALUES ($1, $2, $3, $4, $5, NOT EXISTS (SELECT 1 FROM accounts WHERE budget_id = $1))
		 RETURNING id, is_default`,
		budgetID, in.Name, string(in.Kind), string(in.Currency), in.MultiCurrency,
	).Scan(&a.ID, &a.IsDefault)
	if err != nil {
		return models.Account{}, fmt.Errorf("create account: %w", err)
	}
	a.Balances = []models.CurrencyBalance{{Currency: a.Currency}}
	return a, nil
}

const accountColumns = `id, budget_id, name, kind, currency, multi_currency, is_default`

func (r *Accounts) Get(ctx context.Context, budgetID, id int64) (models.Account, error) {
	a, err := scanAccount(r.db.Q(ctx).QueryRow(ctx,
		`SELECT `+accountColumns+` FROM accounts WHERE budget_id = $1 AND id = $2`, budgetID, id))
	if err != nil {
		return models.Account{}, fmt.Errorf("get account: %w", mapErr(err))
	}
	balances, err := r.balances(ctx, budgetID, &id)
	if err != nil {
		return models.Account{}, err
	}
	a.Balances = arrange(a.Currency, balances[a.ID])
	return a, nil
}

func (r *Accounts) List(ctx context.Context, budgetID int64) ([]models.Account, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `SELECT `+accountColumns+` FROM accounts WHERE budget_id = $1 ORDER BY id`, budgetID)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	defer rows.Close()

	var out []models.Account
	for rows.Next() {
		a, err := scanAccount(rows)
		if err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}

	balances, err := r.balances(ctx, budgetID, nil)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].Balances = arrange(out[i].Currency, balances[out[i].ID])
	}
	return out, nil
}

// balances считает остатки и расходы из транзакций, по счетам и валютам. accountID nil значит все счета бюджета.
func (r *Accounts) balances(ctx context.Context, budgetID int64, accountID *int64) (map[int64][]models.CurrencyBalance, error) {
	rows, err := r.db.Q(ctx).Query(ctx,
		`SELECT account_id, currency, SUM(amount_minor), COALESCE(SUM(-amount_minor) FILTER (WHERE kind = 'expense'), 0)
		 FROM transactions
		 WHERE budget_id = $1 AND deleted_at IS NULL AND ($2::bigint IS NULL OR account_id = $2)
		 GROUP BY account_id, currency`, budgetID, accountID)
	if err != nil {
		return nil, fmt.Errorf("account balances: %w", err)
	}
	defer rows.Close()

	out := map[int64][]models.CurrencyBalance{}
	for rows.Next() {
		var (
			id int64
			cu string
			b  models.CurrencyBalance
		)
		if err := rows.Scan(&id, &cu, &b.BalanceMinor, &b.SpentMinor); err != nil {
			return nil, fmt.Errorf("scan balance: %w", err)
		}
		b.Currency = money.Currency(cu)
		out[id] = append(out[id], b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("account balances: %w", err)
	}
	return out, nil
}

// DailySpends отдаёт расходы по счетам, валютам и дням: по ним считаются суммы в базовой валюте по курсу дня записи.
func (r *Accounts) DailySpends(ctx context.Context, budgetID int64) ([]models.DailySpend, error) {
	rows, err := r.db.Q(ctx).Query(ctx,
		`SELECT account_id, currency, occurred_on, SUM(-amount_minor)
		 FROM transactions
		 WHERE budget_id = $1 AND deleted_at IS NULL AND kind = 'expense'
		 GROUP BY account_id, currency, occurred_on`, budgetID)
	if err != nil {
		return nil, fmt.Errorf("daily spends: %w", err)
	}
	defer rows.Close()

	var out []models.DailySpend
	for rows.Next() {
		var (
			s  models.DailySpend
			cu string
			d  time.Time
		)
		if err := rows.Scan(&s.AccountID, &cu, &d, &s.SpentMinor); err != nil {
			return nil, fmt.Errorf("scan daily spend: %w", err)
		}
		s.Currency, s.Day = money.Currency(cu), apptime.FromDate(d)
		out = append(out, s)
	}
	return out, rows.Err()
}

// arrange ставит основную валюту первой (даже без записей, с нулём), остальные по коду.
func arrange(main money.Currency, got []models.CurrencyBalance) []models.CurrencyBalance {
	out := []models.CurrencyBalance{{Currency: main}}
	var rest []models.CurrencyBalance
	for _, b := range got {
		if b.Currency == main {
			out[0] = b
		} else {
			rest = append(rest, b)
		}
	}
	sort.Slice(rest, func(i, j int) bool { return rest[i].Currency < rest[j].Currency })
	return append(out, rest...)
}

func scanAccount(row pgx.Row) (models.Account, error) {
	var (
		a        models.Account
		kind, cu string
	)
	if err := row.Scan(&a.ID, &a.BudgetID, &a.Name, &kind, &cu, &a.MultiCurrency, &a.IsDefault); err != nil {
		return models.Account{}, err
	}
	a.Kind, a.Currency = models.AccountKind(kind), money.Currency(cu)
	return a, nil
}
