package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jylitalo/expenses/config"
	"github.com/spf13/cobra"
)

func Execute(ctx context.Context) error {
	rootCmd := &cobra.Command{
		Use:   "expenses",
		Short: "Track monthly burn-rate",
		RunE: func(cmd *cobra.Command, args []string) error {
			var income, total, largeTotal float64

			ctx := cmd.Context()
			cfg, errCfg := config.Get(ctx)
			events, errEvents := config.ReadOPEvents(ctx)
			if err := errors.Join(errCfg, errEvents); err != nil {
				return err
			}
			income = 0
			total = 0
			largeTotal = 0
			for _, event := range events {
				if event.Amount > 0 {
					income += event.Amount
					continue
				}
				// fmt.Printf("%.2f < %.2f\n", -event.Amount, cfg.Large)
				if -event.Amount > cfg.Large {
					largeTotal -= event.Amount
					fmt.Printf("%s %s %.2f\n", event.Date.Format(time.RFC3339), event.Target, -event.Amount)
				} else {
					total -= event.Amount
				}
			}
			fmt.Printf("income: %.2f, total: %.2f, total(large): %.2f\n", income, total, largeTotal)
			return nil
		},
	}
	return rootCmd.ExecuteContext(ctx)
}
