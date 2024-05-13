package db

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5"
)

var DB *Queries

func InitializeDB() error {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	DB = New(conn)
	return nil
}
