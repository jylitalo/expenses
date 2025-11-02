package cmd

import (
	"context"
	"errors"
	"log/slog"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"

	"github.com/jylitalo/expenses/config"
	"github.com/jylitalo/expenses/storage"
)

// fetchCmd fetches activity data from Strava
func makeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "make",
		Short: "Transform fetched CSV files into Sqlite database",
		RunE: func(cmd *cobra.Command, args []string) error {
			return makeDB(cmd.Context())
		},
	}
	return cmd
}

func makeDB(ctx context.Context) error {
	db := &storage.Sqlite3{}
	slog.Info("Making database")
	opEvents, errOP := config.ReadOPEvents(ctx)
	sEvents, errS := config.ReadSPankkiEvents(ctx)
	return errors.Join(
		errOP, errS,
		db.Remove(), db.Open(), db.Create(),
		db.Insert(ctx, append(opEvents, sEvents...)), db.Close(),
	)
}
