package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ConnectPostgres открывает соединение с PostgreSQL по заданной строке подключения.
// Возвращает *pgx.Conn, которое надо закрывать.
func ConnectPostgres(connStr string) (*pgx.Conn, error) {
    cfg, err := pgx.ParseConfig(connStr)
    if err != nil {
        return nil, fmt.Errorf("parse config error: %w", err)
    }

    conn, err := pgx.ConnectConfig(context.Background(), cfg)
    if err != nil {
        return nil, fmt.Errorf("pgx connect error: %w", err)
    }

    // Проверка связи
    if err := conn.Ping(context.Background()); err != nil {
        conn.Close(context.Background())
        return nil, fmt.Errorf("pgx ping error: %w", err)
    }

    return conn, nil
}