package cmd

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/spf13/cobra"

	"github.com/jylitalo/expenses/config"
	"github.com/jylitalo/expenses/storage"
)

type database interface {
	Query(ctx context.Context, fields []string, opts ...storage.QueryOption) (*sql.Rows, error)
}

func Execute(ctx context.Context) error {
	rootCmd := &cobra.Command{
		Use:   "expenses",
		Short: "Track monthly burn-rate",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, errCfg := config.Get(ctx)
			db := &storage.Sqlite3{}
			if err := errors.Join(errCfg, makeDB(ctx), db.Open()); err != nil {
				return err
			}
			defer db.Close()
			incoming, errIn := monthlyStats(ctx, db, storage.WithAmount("> 0"))
			outgoing, errOut := monthlyStats(ctx, db, storage.WithAmount("< 0"))
			if err := errors.Join(outsideBoundaries(ctx, *cfg, db), errIn, errOut); err != nil {
				return err
			}
			fmt.Println()
			uniq := maps.Clone(incoming)
			maps.Copy(uniq, outgoing)
			months := slices.Collect(maps.Keys(uniq))
			slices.Sort(months)
			in := []float64{}
			out := []float64{}
			for _, month := range months {
				fmt.Printf("%s in: %8.2f€ out: %8.2f€\n", month, incoming[month], -outgoing[month])
				in = append(in, incoming[month])
				out = append(out, -outgoing[month])
			}
			slices.Sort(in)
			slices.Sort(out)
			n := len(in)
			fmt.Println()
			fmt.Printf("median:\nin: %7.2f€/m out: %7.2f€/m\n", in[n/2], out[n/2])
			fmt.Printf("average:\nin: %7.2f€/m out: %7.2f€/m\n", sum(in)/float64(n), sum(out)/float64(n))
			return nil
		},
	}
	rootCmd.AddCommand(makeCmd())
	return rootCmd.ExecuteContext(ctx)
}

func sum(values []float64) float64 {
	var f float64

	for _, v := range values {
		f += v
	}
	return f
}

func monthlyStats(ctx context.Context, db database, opts ...storage.QueryOption) (map[string]float64, error) {
	o := []storage.QueryOption{
		storage.WithOrder(storage.OrderConfig{GroupBy: []string{"Year", "Month"}}),
		storage.WithoutLabels([]string{"exclude"}),
	}
	rows, err := db.Query(
		ctx, []string{"Year", "Month", "sum(Amount)"},
		append(o, opts...)...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	numbers := map[string]float64{}
	for rows.Next() {
		var year, month int
		var amount float64
		err := rows.Scan(&year, &month, &amount)
		if err != nil {
			return numbers, err
		}
		numbers[fmt.Sprintf("%d-%02d", year, month)] = amount
	}
	return numbers, nil
}

func outsideBoundaries(ctx context.Context, cfg config.Config, db database) error {
	rows, err := db.Query(
		ctx, []string{"Year", "Month", "Day", "Name", "Amount", "Explanation"},
		storage.WithoutLabels([]string{"silent"}),
	)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var year, month, day int
		var name, explanation string
		var amount float64
		if err := rows.Scan(&year, &month, &day, &name, &amount, &explanation); err != nil {
			return err
		}
		if amount < cfg.Silent.Min || cfg.Silent.Max < amount {
			fmt.Printf("%d-%02d-%02d %9.2f€ %s - %s\n", year, month, day, amount, name, explanation)
		}
	}
	return nil
}
