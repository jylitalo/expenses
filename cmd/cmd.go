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
			errDB := db.Open()
			if err := errors.Join(errCfg, errDB); err != nil {
				return err
			}
			if err := outsideBoundaries(ctx, *cfg, db); err != nil {
				return err
			}
			fmt.Println()
			incoming, errIn := monthlyStats(ctx, db, storage.WithAmount("> 0"))
			outgoing, errOut := monthlyStats(ctx, db, storage.WithAmount("< 0"))
			if err := errors.Join(errIn, errOut); err != nil {
				return err
			}
			uniq := maps.Clone(incoming)
			maps.Copy(uniq, outgoing)
			months := slices.Collect(maps.Keys(uniq))
			slices.Sort(months)
			in := []float64{}
			out := []float64{}
			for _, month := range months {
				fmt.Printf("%s in: %8.2f€ out: %8.2f€\n", month, incoming[month], -outgoing[month])
				in = append(in, incoming[month])
				out = append(out, outgoing[month])
			}
			slices.Sort(in)
			slices.Sort(out)
			n := len(in)
			fmt.Println()
			fmt.Printf("median:\nin: %7.2f€/m out: %7.2f€/m\n", in[n/2], -out[n/2])
			fmt.Printf("average:\nin: %7.2f€/m out: %7.2f€/m\n", sum(in)/float64(n), -sum(out)/float64(n))
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
	o := []storage.QueryOption{storage.WithOrder(storage.OrderConfig{
		GroupBy: []string{"Year", "Month"},
	})}
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
	rows, err := db.Query(ctx, []string{"Year", "Month", "Day", "Name", "Amount"})
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
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
		if event.Amount < cfg.Silent.Min || cfg.Silent.Max < event.Amount {
			fmt.Printf("%d-%02d-%02d %s %.2f€\n", event.Year, event.Month, event.Day, event.Name, event.Amount)
		}
	}
	return nil
}
