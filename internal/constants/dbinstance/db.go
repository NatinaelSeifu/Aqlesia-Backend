package dbinstance

import (
	"aqlesia/internal/constants/model/db"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

type DBInstance struct {
	*db.Queries
	Pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) DBInstance {
	return DBInstance{
		Pool:    pool,
		Queries: db.New(stdlib.OpenDBFromPool(pool)),
	}
}
