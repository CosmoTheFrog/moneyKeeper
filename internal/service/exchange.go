package service

import (
	"context"
	"fmt"

	"moneykeeper/internal/models"
	"moneykeeper/pkg/money"
)

// boundaryCategory проверяет категорию записи между двумя счетами. Деньги уходят со счёта внутри бюджета на счёт вне
// бюджета: это расход конверта, категория обязательна. В остальных случаях бюджет не меняется и категории быть не должно.
func (s *Service) boundaryCategory(ctx context.Context, u models.User, from, to models.Account, id *int64) (*int64, error) {
	if from.OffBudget || !to.OffBudget {
		if id != nil {
			return nil, fmt.Errorf("category without leaving the budget: %w", models.ErrInvalidTransaction)
		}
		return nil, nil
	}
	cat, err := s.usableCategory(ctx, u, id)
	if err != nil {
		return nil, err
	}
	return &cat.ID, nil
}

// CreateExchange записывает обмен валют, покупку или продажу монеты: две записи с суммами как провёл банк или биржа и
// при комиссии третья. Все записи одной группы, поэтому правятся и удаляются вместе.
func (s *Service) CreateExchange(ctx context.Context, u models.User, in models.NewExchange) ([]models.Transaction, error) {
	if in.OutMinor <= 0 || in.OutMinor > maxAmountMinor || in.InMinor <= 0 || in.InMinor > maxAmountMinor ||
		in.FeeMinor < 0 || in.FeeMinor > maxAmountMinor {
		return nil, fmt.Errorf("amount: %w", models.ErrInvalidTransaction)
	}
	outCur, err := money.Parse(in.OutCurrency)
	if err != nil {
		return nil, err
	}
	inCur, err := money.Parse(in.InCurrency)
	if err != nil {
		return nil, err
	}
	// Одна валюта по обе стороны это перевод, а не обмен.
	if outCur == inCur {
		return nil, fmt.Errorf("exchange within %s: %w", outCur, models.ErrInvalidTransaction)
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
		from, err := s.accounts.Get(ctx, u.ActiveBudgetID, in.FromAccountID)
		if err != nil {
			return err
		}
		to, err := s.accounts.Get(ctx, u.ActiveBudgetID, in.ToAccountID)
		if err != nil {
			return err
		}
		if !from.Allows(outCur) {
			return fmt.Errorf("%s on account %d: %w", outCur, from.ID, models.ErrCurrencyMismatch)
		}
		if !to.Allows(inCur) {
			return fmt.Errorf("%s on account %d: %w", inCur, to.ID, models.ErrCurrencyMismatch)
		}
		category, err := s.boundaryCategory(ctx, u, from, to, in.CategoryID)
		if err != nil {
			return err
		}

		var feeCategory *int64
		if in.FeeMinor > 0 {
			// Комиссия со счёта внутри бюджета это расход с категорией, со счёта вне бюджета только уменьшает портфель.
			if from.OffBudget {
				if in.FeeCategoryID != nil {
					return fmt.Errorf("fee category on an off-budget account: %w", models.ErrInvalidTransaction)
				}
			} else {
				cat, err := s.usableCategory(ctx, u, in.FeeCategoryID)
				if err != nil {
					return err
				}
				feeCategory = &cat.ID
			}
		}

		groupID, err := s.transactions.NextTransferID(ctx)
		if err != nil {
			return err
		}
		base := models.Transaction{
			BudgetID: u.ActiveBudgetID, Kind: models.KindExchange, OccurredOn: day, Note: note, CreatedBy: u.ID, TransferID: &groupID,
		}
		rows := []models.Transaction{
			withLeg(base, from.ID, -in.OutMinor, outCur, category),
			withLeg(base, to.ID, in.InMinor, inCur, nil),
		}
		if in.FeeMinor > 0 {
			fee := withLeg(base, from.ID, -in.FeeMinor, outCur, feeCategory)
			fee.Kind = models.KindFee
			if feeCategory != nil {
				fee.Kind = models.KindExpense
			}
			rows = append(rows, fee)
		}
		for _, r := range rows {
			t, err := s.transactions.Insert(ctx, r)
			if err != nil {
				return err
			}
			out = append(out, t)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := s.priceTransactions(ctx, u, out, ""); err != nil {
		return nil, err
	}
	return out, nil
}

func withLeg(base models.Transaction, accountID, amount int64, cur money.Currency, category *int64) models.Transaction {
	base.AccountID, base.AmountMinor, base.Currency, base.CategoryID = accountID, amount, cur, category
	return base
}
