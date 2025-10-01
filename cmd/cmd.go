package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jylitalo/expenses/config"
	"github.com/jylitalo/expenses/storage"
)

func Execute(ctx context.Context) error {
	rootCmd := &cobra.Command{
		Use:   "expenses",
		Short: "Track monthly burn-rate",
		RunE: func(cmd *cobra.Command, args []string) error {
			var income, total, largeTotal float64

			ctx := cmd.Context()
			cfg, errCfg := config.Get(ctx)
			db := &storage.Sqlite3{}
			errDB := db.Open()
			if err := errors.Join(errCfg, errDB); err != nil {
				return err
			}
			rows, err := db.Query(ctx, []string{"Year", "Month", "Day", "Name", "Amount"})
			if err != nil {
				return err
			}
			defer func() { _ = rows.Close() }()
			income = 0
			total = 0
			largeTotal = 0
			for rows.Next() {
				var year, month, day int
				var name string
				var amount float64
				err := rows.Scan(&year, &month, &day, &name, &amount)
				if err != nil {
					return err
				}
				event := config.EventRecord{
					Year:   year,
					Month:  month,
					Day:    day,
					Name:   name,
					Amount: amount,
				}
				if event.Amount > 0 {
					income += event.Amount
					continue
				}
				// fmt.Printf("%.2f < %.2f\n", -event.Amount, cfg.Large)
				if -event.Amount > cfg.Large {
					largeTotal -= event.Amount
					fmt.Printf("%d-%02d-%02d %s %.2f\n", event.Year, event.Month, event.Day, event.Name, -event.Amount)
				} else {
					total -= event.Amount
				}
			}
			fmt.Printf("income: %.2f, total: %.2f, total(large): %.2f\n", income, total, largeTotal)
			fmt.Printf("income: %.2f/m, total: %.2f/m, total(large): %.2f/m\n", income/12, total/12, largeTotal/12)
			return nil
		},
	}
	rootCmd.AddCommand(makeCmd())
	return rootCmd.ExecuteContext(ctx)
}
