package repo

import (
	"errors"

	"github.com/jackc/pgx/v5"

	"moneykeeper/internal/models"
)

func mapErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return models.ErrNotFound
	}
	return err
}
