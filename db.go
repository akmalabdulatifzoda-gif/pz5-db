package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // драйвер pgx в режиме совместимости с database/sql
)

// PoolConfig — параметры пула соединений *sql.DB.
type PoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// DefaultPool — настройки, выбранные по результатам замеров (см. bench.go и README).
var DefaultPool = PoolConfig{
	MaxOpenConns:    10,
	MaxIdleConns:    5,
	ConnMaxLifetime: 30 * time.Minute,
	ConnMaxIdleTime: 5 * time.Minute,
}

func openDB(dsn string) (*sql.DB, error) {
	return openDBWithPool(dsn, DefaultPool)
}

func openDBWithPool(dsn string, p PoolConfig) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	// настройки пула
	db.SetMaxOpenConns(p.MaxOpenConns)
	db.SetMaxIdleConns(p.MaxIdleConns)
	db.SetConnMaxLifetime(p.ConnMaxLifetime)
	db.SetConnMaxIdleTime(p.ConnMaxIdleTime)

	// проверка соединения с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	log.Println("Connected to PostgreSQL")
	return db, nil
}
