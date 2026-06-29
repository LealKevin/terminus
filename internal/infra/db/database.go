package db

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Database struct {
	Conn *pgxpool.Pool
}

func NewDatabase() (*Database, error) {
	databaseUrl := os.Getenv("DATABASE_URL")
	if databaseUrl == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is not set")
	}

	config, err := pgxpool.ParseConfig(databaseUrl)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %v", err)
	}
	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnIdleTime = time.Minute * 30
	config.MaxConnLifetime = time.Hour
	config.ConnConfig.ConnectTimeout = time.Second * 30

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("unable to ping database: %v", err)
	}

	return &Database{Conn: pool}, nil
}

func (db *Database) Close() {
	if db.Conn != nil {
		db.Conn.Close()
	}
}
