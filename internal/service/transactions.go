package service

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"moneykeeper/internal/models"
	"moneykeeper/pkg/money"
)

const (
	defaultPageSize = 50
	maxPageSize     = 200
	maxNoteRunes    = 200
)

// CreateTransaction создаёт расход, доход или перевод. Перевод даёт две записи, остальное одну.
func (s *Service) CreateTransaction(ctx context.Context, u models.User, in models.NewTransaction) ([]models.Transaction, error) {
	if in.AmountMinor <= 0 || in.AmountMinor > maxAmountMinor {
		return nil, fmt.Errorf("amount: %w", models.ErrInvalidTransaction)
	}
	note, err := validNote(in.Note)
	if err != nil {
		return nil, err
	}
	day := s.today(u)
	if in.OccurredOn != nil {
		day = *in.OccurredOn
	}

	var out []models.Transaction
	err = s.db.InTx(ctx, func(ctx context.Context) error {
		acc, err := s.accounts.Get(ctx, u.ActiveBudgetID, in.AccountID)
		if err != nil {
			return err
		}
		cur, err := currencyFor(acc, in.Currency)
		if err != nil {
			return err
		}
		base := models.Transaction{
			BudgetID: u.ActiveBudgetID, AccountID: acc.ID, Currency: cur,
			OccurredOn: day, Note: note, CreatedBy: u.ID,
		}

		switch in.Kind {
		case models.KindExpense:
			cat, err := s.usableCategory(ctx, u, in.CategoryID)
			if err != nil {
				return err
			}
			base.Kind, base.CategoryID, base.AmountMinor = models.KindExpense, &cat.ID, -in.AmountMinor
			t, err := s.transactions.Insert(ctx, base)
			out = []models.Transaction{t}
			return err

		case models.KindIncome:
			if in.CategoryID != nil || in.ToAccountID != nil {
				return fmt.Errorf("income with category or target account: %w", models.ErrInvalidTransaction)
			}
			base.Kind, base.AmountMinor = models.KindIncome, in.AmountMinor
			t, err := s.transactions.Insert(ctx, base)
			out = []models.Transaction{t}
			return err

		case models.KindTransfer:
			if in.CategoryID != nil || in.ToAccountID == nil || *in.ToAccountID == in.AccountID {
				return fmt.Errorf("transfer target: %w", models.ErrInvalidTransaction)
			}
			to, err := s.accounts.Get(ctx, u.ActiveBudgetID, *in.ToAccountID)
			if err != nil {
				return err
			}
			if !to.Allows(cur) {
				return fmt.Errorf("%s to account %d: %w", cur, to.ID, models.ErrCurrencyMismatch)
			}
			transferID, err := s.transactions.NextTransferID(ctx)
			if err != nil {
				return err
			}
			base.Kind, base.TransferID = models.KindTransfer, &transferID
			outLeg, inLeg := base, base
			outLeg.AmountMinor = -in.AmountMinor
			inLeg.AccountID, inLeg.AmountMinor = to.ID, in.AmountMinor
			for _, leg := range []models.Transaction{outLeg, inLeg} {
				t, err := s.transactions.Insert(ctx, leg)
				if err != nil {
					return err
				}
				out = append(out, t)
			}
			return nil

		default:
			return fmt.Errorf("kind %q: %w", in.Kind, models.ErrInvalidTransaction)
		}
	})
	if err != nil {
		return nil, err
	}
	if err := s.priceTransactions(ctx, u, out); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateTransaction меняет заданные поля. У перевода обе записи меняются вместе, счета не меняются.
func (s *Service) UpdateTransaction(ctx context.Context, u models.User, id int64, p models.TransactionPatch) ([]models.Transaction, error) {
	if p.AmountMinor != nil && (*p.AmountMinor <= 0 || *p.AmountMinor > maxAmountMinor) {
		return nil, fmt.Errorf("amount: %w", models.ErrInvalidTransaction)
	}
	if p.Note != nil {
		note, err := validNote(*p.Note)
		if err != nil {
			return nil, err
		}
		p.Note = &note
	}

	var out []models.Transaction
	err := s.db.InTx(ctx, func(ctx context.Context) error {
		t, err := s.transactions.Get(ctx, u.ActiveBudgetID, id)
		if err != nil {
			return err
		}
		if err := checkPatchAllowed(t.Kind, p); err != nil {
			return err
		}
		legs := []models.Transaction{t}
		if t.Kind == models.KindTransfer {
			if legs, err = s.transactions.Legs(ctx, u.ActiveBudgetID, *t.TransferID); err != nil {
				return err
			}
		}

		for _, leg := range legs {
			v := leg
			if p.AmountMinor != nil {
				v.AmountMinor = signOf(leg.AmountMinor) * *p.AmountMinor
			}
			if p.OccurredOn != nil {
				v.OccurredOn = *p.OccurredOn
			}
			if p.Note != nil {
				v.Note = *p.Note
			}
			if p.AccountID != nil || p.Currency != nil {
				accID := leg.AccountID
				if p.AccountID != nil {
					accID = *p.AccountID
				}
				acc, err := s.accounts.Get(ctx, u.ActiveBudgetID, accID)
				if err != nil {
					return err
				}
				cur := leg.Currency
				switch {
				case p.Currency != nil:
					if cur, err = currencyFor(acc, *p.Currency); err != nil {
						return err
					}
				case !acc.Allows(cur):
					// Переезд на обычный счёт в другой валюте меняет валюту записи: сумма та же цифра, валюта счёта.
					cur = acc.Currency
				}
				v.AccountID, v.Currency = acc.ID, cur
			}
			if p.CategoryID != nil {
				cat, err := s.usableCategory(ctx, u, p.CategoryID)
				if err != nil {
					return err
				}
				v.CategoryID = &cat.ID
			}
			if err := s.transactions.Update(ctx, u.ActiveBudgetID, leg.ID, u.ID, v); err != nil {
				return err
			}
			out = append(out, v)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := s.priceTransactions(ctx, u, out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteTransaction скрывает запись, у перевода обе записи сразу.
func (s *Service) DeleteTransaction(ctx context.Context, u models.User, id int64) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		t, err := s.transactions.Get(ctx, u.ActiveBudgetID, id)
		if err != nil {
			return err
		}
		legs := []models.Transaction{t}
		if t.Kind == models.KindTransfer {
			if legs, err = s.transactions.Legs(ctx, u.ActiveBudgetID, *t.TransferID); err != nil {
				return err
			}
		}
		for _, leg := range legs {
			if err := s.transactions.Delete(ctx, u.ActiveBudgetID, leg.ID, u.ID); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Service) ListTransactions(ctx context.Context, u models.User, limit, offset int) ([]models.Transaction, error) {
	if limit <= 0 {
		limit = defaultPageSize
	}
	limit = min(limit, maxPageSize)
	offset = max(offset, 0)
	ts, err := s.transactions.List(ctx, u.ActiveBudgetID, limit, offset)
	if err != nil {
		return nil, err
	}
	if err := s.priceTransactions(ctx, u, ts); err != nil {
		return nil, err
	}
	return ts, nil
}

func (s *Service) TransactionEvents(ctx context.Context, u models.User, id int64) ([]models.TransactionEvent, error) {
	return s.transactions.Events(ctx, u.ActiveBudgetID, id)
}

// usableCategory проверяет, что категория есть в бюджете и не скрыта.
func (s *Service) usableCategory(ctx context.Context, u models.User, id *int64) (models.Category, error) {
	if id == nil {
		return models.Category{}, fmt.Errorf("category is required: %w", models.ErrInvalidTransaction)
	}
	cat, err := s.categories.Get(ctx, u.ActiveBudgetID, *id)
	if err != nil {
		return models.Category{}, err
	}
	if cat.Hidden {
		return models.Category{}, fmt.Errorf("category %d is hidden: %w", cat.ID, models.ErrInvalidTransaction)
	}
	return cat, nil
}

// checkPatchAllowed: категорию можно менять только у расхода, счёт только у расхода и дохода.
func checkPatchAllowed(kind models.TransactionKind, p models.TransactionPatch) error {
	if p.CategoryID != nil && kind != models.KindExpense {
		return fmt.Errorf("category of %s: %w", kind, models.ErrInvalidTransaction)
	}
	if (p.AccountID != nil || p.Currency != nil) && kind != models.KindExpense && kind != models.KindIncome {
		return fmt.Errorf("account or currency of %s: %w", kind, models.ErrInvalidTransaction)
	}
	return nil
}

// currencyFor выбирает валюту записи: пустая значит основная валюта счёта, иначе она должна быть допустима на счёте.
func currencyFor(acc models.Account, requested string) (money.Currency, error) {
	if requested == "" {
		return acc.Currency, nil
	}
	c, err := money.Parse(requested)
	if err != nil {
		return "", err
	}
	if !acc.Allows(c) {
		return "", fmt.Errorf("%s on account %d: %w", c, acc.ID, models.ErrCurrencyMismatch)
	}
	return c, nil
}

func validNote(note string) (string, error) {
	note = strings.TrimSpace(note)
	if utf8.RuneCountInString(note) > maxNoteRunes {
		return "", fmt.Errorf("note: %w", models.ErrInvalidTransaction)
	}
	return note, nil
}

func signOf(v int64) int64 {
	if v < 0 {
		return -1
	}
	return 1
}
