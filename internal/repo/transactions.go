package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"moneykeeper/internal/models"
	"moneykeeper/internal/pgclient"
	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

type Transactions struct{ db *pgclient.Client }

func NewTransactions(db *pgclient.Client) *Transactions { return &Transactions{db: db} }

const txColumns = `id, budget_id, account_id, category_id, kind, amount_minor, currency, occurred_on, note, transfer_id, created_by, created_at`

// snapshotExpr собирает состояние записи для истории изменений.
const snapshotExpr = `jsonb_build_object('account_id', account_id, 'category_id', category_id,
	'amount_minor', amount_minor, 'occurred_on', occurred_on, 'note', note)`

func (r *Transactions) NextTransferID(ctx context.Context) (int64, error) {
	var id int64
	if err := r.db.Q(ctx).QueryRow(ctx, `SELECT nextval('transfer_ids')`).Scan(&id); err != nil {
		return 0, fmt.Errorf("next transfer id: %w", err)
	}
	return id, nil
}

// Insert пишет запись и событие create. Поля ID и CreatedAt заполняет база.
func (r *Transactions) Insert(ctx context.Context, t models.Transaction) (models.Transaction, error) {
	err := r.db.InTx(ctx, func(ctx context.Context) error {
		q := r.db.Q(ctx)
		if err := q.QueryRow(ctx,
			`INSERT INTO transactions (budget_id, account_id, category_id, kind, amount_minor, currency, occurred_on, note, transfer_id, created_by)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id, created_at`,
			t.BudgetID, t.AccountID, t.CategoryID, string(t.Kind), t.AmountMinor, string(t.Currency),
			t.OccurredOn.Date(), t.Note, t.TransferID, t.CreatedBy,
		).Scan(&t.ID, &t.CreatedAt); err != nil {
			return err
		}
		_, err := q.Exec(ctx,
			`INSERT INTO transaction_events (transaction_id, user_id, action, after)
			 SELECT id, $2, 'create', `+snapshotExpr+` FROM transactions WHERE id = $1`, t.ID, t.CreatedBy)
		return err
	})
	if err != nil {
		return models.Transaction{}, fmt.Errorf("insert transaction: %w", err)
	}
	return t, nil
}

func (r *Transactions) Get(ctx context.Context, budgetID, id int64) (models.Transaction, error) {
	t, err := scanTransaction(r.db.Q(ctx).QueryRow(ctx,
		`SELECT `+txColumns+` FROM transactions WHERE budget_id = $1 AND id = $2 AND deleted_at IS NULL`, budgetID, id))
	if err != nil {
		return models.Transaction{}, fmt.Errorf("get transaction: %w", mapErr(err))
	}
	return t, nil
}

// Legs возвращает обе записи перевода.
func (r *Transactions) Legs(ctx context.Context, budgetID, transferID int64) ([]models.Transaction, error) {
	return r.query(ctx,
		`SELECT `+txColumns+` FROM transactions WHERE budget_id = $1 AND transfer_id = $2 AND deleted_at IS NULL ORDER BY id`,
		budgetID, transferID)
}

// CoinActivity возвращает все записи в монетах и все записи групп (обмены, переводы), в которых есть монета, в
// порядке даты. По ним считается портфель: покупки и продажи, потери на комиссиях сети.
func (r *Transactions) CoinActivity(ctx context.Context, budgetID int64, coins []string) ([]models.Transaction, error) {
	return r.query(ctx,
		`SELECT `+txColumns+` FROM transactions
		 WHERE budget_id = $1 AND deleted_at IS NULL AND (
		       currency = ANY($2)
		    OR transfer_id IN (SELECT transfer_id FROM transactions
		                       WHERE budget_id = $1 AND deleted_at IS NULL AND transfer_id IS NOT NULL AND currency = ANY($2)))
		 ORDER BY occurred_on, id`,
		budgetID, coins)
}

// Expenses возвращает расходы бюджета за период, по ним платежи расписания находят свои траты.
func (r *Transactions) Expenses(ctx context.Context, budgetID int64, from, to apptime.Day) ([]models.Transaction, error) {
	return r.query(ctx,
		`SELECT `+txColumns+` FROM transactions
		 WHERE budget_id = $1 AND deleted_at IS NULL AND kind = 'expense' AND occurred_on BETWEEN $2 AND $3
		 ORDER BY occurred_on, id`, budgetID, from.Date(), to.Date())
}

func (r *Transactions) List(ctx context.Context, budgetID int64, limit, offset int) ([]models.Transaction, error) {
	return r.query(ctx,
		`SELECT `+txColumns+` FROM transactions WHERE budget_id = $1 AND deleted_at IS NULL
		 ORDER BY occurred_on DESC, id DESC LIMIT $2 OFFSET $3`,
		budgetID, limit, offset)
}

// Update записывает итоговые значения полей и событие update с состоянием до и после.
func (r *Transactions) Update(ctx context.Context, budgetID, id, userID int64, v models.Transaction) error {
	err := r.db.InTx(ctx, func(ctx context.Context) error {
		q := r.db.Q(ctx)
		before, err := r.lockSnapshot(ctx, budgetID, id)
		if err != nil {
			return err
		}
		var after string
		if err := q.QueryRow(ctx,
			`UPDATE transactions SET account_id = $3, category_id = $4, amount_minor = $5, currency = $6,
				occurred_on = $7, note = $8, updated_at = now()
			 WHERE id = $1 AND budget_id = $2 RETURNING `+snapshotExpr+`::text`,
			id, budgetID, v.AccountID, v.CategoryID, v.AmountMinor, string(v.Currency), v.OccurredOn.Date(), v.Note,
		).Scan(&after); err != nil {
			return err
		}
		_, err = q.Exec(ctx,
			`INSERT INTO transaction_events (transaction_id, user_id, action, before, after) VALUES ($1, $2, 'update', $3::jsonb, $4::jsonb)`,
			id, userID, before, after)
		return err
	})
	if err != nil {
		return fmt.Errorf("update transaction: %w", mapErr(err))
	}
	return nil
}

// Delete скрывает запись и пишет событие delete.
func (r *Transactions) Delete(ctx context.Context, budgetID, id, userID int64) error {
	err := r.db.InTx(ctx, func(ctx context.Context) error {
		q := r.db.Q(ctx)
		before, err := r.lockSnapshot(ctx, budgetID, id)
		if err != nil {
			return err
		}
		if _, err := q.Exec(ctx, `UPDATE transactions SET deleted_at = now(), updated_at = now() WHERE id = $1`, id); err != nil {
			return err
		}
		_, err = q.Exec(ctx,
			`INSERT INTO transaction_events (transaction_id, user_id, action, before) VALUES ($1, $2, 'delete', $3::jsonb)`,
			id, userID, before)
		return err
	})
	if err != nil {
		return fmt.Errorf("delete transaction: %w", mapErr(err))
	}
	return nil
}

// lockSnapshot блокирует живую запись бюджета и возвращает её состояние как JSON.
// Нет записи: pgx.ErrNoRows, его превращает в ErrNotFound вызывающий через mapErr.
func (r *Transactions) lockSnapshot(ctx context.Context, budgetID, id int64) (string, error) {
	var snap string
	err := r.db.Q(ctx).QueryRow(ctx,
		`SELECT `+snapshotExpr+`::text FROM transactions WHERE id = $1 AND budget_id = $2 AND deleted_at IS NULL FOR UPDATE`,
		id, budgetID).Scan(&snap)
	return snap, err
}

// Events возвращает историю записи, в том числе уже удалённой.
func (r *Transactions) Events(ctx context.Context, budgetID, id int64) ([]models.TransactionEvent, error) {
	var exists bool
	if err := r.db.Q(ctx).QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM transactions WHERE id = $1 AND budget_id = $2)`, id, budgetID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check transaction: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("events: %w", models.ErrNotFound)
	}

	rows, err := r.db.Q(ctx).Query(ctx,
		`SELECT id, transaction_id, action, before, after, created_at FROM transaction_events WHERE transaction_id = $1 ORDER BY id`, id)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	var out []models.TransactionEvent
	for rows.Next() {
		var (
			e             models.TransactionEvent
			action        string
			before, after []byte
		)
		if err := rows.Scan(&e.ID, &e.TransactionID, &action, &before, &after, &e.At); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		e.Action = models.EventAction(action)
		if e.Before, err = decodeSnapshot(before); err != nil {
			return nil, err
		}
		if e.After, err = decodeSnapshot(after); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	return out, nil
}

func (r *Transactions) query(ctx context.Context, sql string, args ...any) ([]models.Transaction, error) {
	rows, err := r.db.Q(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query transactions: %w", err)
	}
	defer rows.Close()

	var out []models.Transaction
	for rows.Next() {
		t, err := scanTransaction(rows)
		if err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("query transactions: %w", err)
	}
	return out, nil
}

func scanTransaction(row pgx.Row) (models.Transaction, error) {
	var (
		t        models.Transaction
		kind, cu string
		day      time.Time
	)
	err := row.Scan(&t.ID, &t.BudgetID, &t.AccountID, &t.CategoryID, &kind, &t.AmountMinor, &cu, &day, &t.Note, &t.TransferID, &t.CreatedBy, &t.CreatedAt)
	if err != nil {
		return models.Transaction{}, err
	}
	t.Kind, t.Currency, t.OccurredOn = models.TransactionKind(kind), money.Currency(cu), apptime.FromDate(day)
	return t, nil
}

func decodeSnapshot(raw []byte) (*models.TransactionSnapshot, error) {
	if raw == nil {
		return nil, nil
	}
	var s models.TransactionSnapshot
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("decode snapshot: %w", err)
	}
	return &s, nil
}
