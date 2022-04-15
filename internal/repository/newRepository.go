// Package repository contains code for handling different types of databases
package repository

import (
	"context"

	"github.com/EgMeln/position_service/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
)

// PostgresPrice struct for price pool
type PostgresPrice struct {
	PoolPrice *pgxpool.Pool
}

// PriceTransaction used for structuring, function for working with transaction
type PriceTransaction interface {
	OpenPosition(ctx context.Context, trans *model.Transaction, bay string) (*uuid.UUID, error)
	ClosePosition(ctx context.Context, closePrice *float64, id *uuid.UUID) (string, error)
}
