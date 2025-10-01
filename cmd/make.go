package cmd

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"

	"github.com/jylitalo/expenses/config"
	"github.com/jylitalo/expenses/storage"
)

type Storage interface {
	Query(ctx context.Context, fields []string, opts ...storage.QueryOption) (*sql.Rows, error)
	Close() error
}

// fetchCmd fetches activity data from Strava
func makeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "make",
		Short: "Transform fetched CSV files into Sqlite database",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := makeDB(cmd.Context())
			if err != nil {
				return err
			}
			defer func() { _ = db.Close() }()
			return err
		},
	}
	return cmd
}

func makeDB(ctx context.Context) (Storage, error) {
	db := &storage.Sqlite3{}
	slog.Info("Making database")
	events, err := config.ReadOPEvents(ctx)
	if err != nil {
		return nil, err
	}
	return db, errors.Join(
		db.Remove(), db.Open(), db.Create(),
		db.Insert(ctx, events),
	)
}
