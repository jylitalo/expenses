package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/jylitalo/expenses/config"
)

type OrderConfig struct {
	GroupBy []string
	OrderBy []string
	Limit   int
}

type QueryConfig struct {
	Tables        []string
	Name          string
	Day           int
	Month         int
	Years         []int
	Order         *OrderConfig
	Amount        string // condition like "< 0"
	ExcludeLabels []string
}

type QueryOption func(c *QueryConfig)

type Sqlite3 struct {
	db *sql.DB
}

// dbName is the filename of sqlite file
const dbName = "expenses.sql"

// EventTable is where bank account events
const EventTable = "Event"

func WithAmount(condition string) QueryOption {
	return func(c *QueryConfig) {
		c.Amount = condition
	}
}

func WithDayOfYear(day, month int) QueryOption {
	return func(c *QueryConfig) {
		c.Day = day
		c.Month = month
	}
}

func WithYears(year ...int) QueryOption {
	return func(c *QueryConfig) {
		c.Years = append(c.Years, year...)
	}
}

func WithOrder(order OrderConfig) QueryOption {
	return func(c *QueryConfig) {
		c.Order = &order
	}
}

func WithTable(table string) QueryOption {
	return func(c *QueryConfig) {
		if slices.Contains(c.Tables, table) {
			slog.Error("WithTable already contains table", "c.Tables", c.Tables, "table", table)
		}
		c.Tables = append(c.Tables, table)
	}
}

func WithoutLabels(labels []string) QueryOption {
	return func(c *QueryConfig) {
		c.ExcludeLabels = append(c.ExcludeLabels, labels...)
	}
}

func (sq *Sqlite3) Remove() error {
	if _, err := os.Stat(dbName); err != nil && errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return os.Remove(dbName)
}

func (sq *Sqlite3) Open() error {
	var err error

	if sq.db == nil {
		sq.db, err = sql.Open("sqlite3", dbName)
	}
	return err
}

func (sq *Sqlite3) Create() error {
	if sq.db == nil {
		return errors.New("database is nil")
	}
	ymd := "Year integer, Month integer, Day integer,"
	_, err := sq.db.Exec(`create table ` + EventTable + ` ( ` + ymd + `
		Explanation string,
		Name string,
		Account string,
		Amount number,
		Labels string
	)`)
	return err
}

func (sq *Sqlite3) Insert(ctx context.Context, records []config.EventRecord) error {
	if sq.db == nil {
		return errors.New("database is nil")
	}
	tx, err := sq.db.Begin()
	if err != nil {
		return err
	}
	fields := []string{"Year", "Month", "Day", "Explanation", "Name", "Account", "Amount", "Labels"}
	q := strings.Repeat("?,", len(fields)-1) + "?"
	// #nosec G202
	stmt, err := tx.Prepare("insert into " + EventTable + "(" + strings.Join(fields, ",") + ") values (" + q + ")")
	if err != nil {
		return fmt.Errorf("InsertEvent caused %w", err)
	}
	defer func() { _ = stmt.Close() }()
	for _, r := range records {
		_, err := stmt.Exec(r.Year, r.Month, r.Day, r.Explanation, r.Name, r.Account, r.Amount, r.Labels)
		if err != nil {
			return fmt.Errorf("InsertSummary statement execution caused: %w", err)
		}
	}
	return tx.Commit()
}

func sqlQuery(fields []string, opts ...QueryOption) (string, []interface{}) { //nolint:cyclop
	cfg := &QueryConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	where := []string{}
	args := []string{}
	if len(cfg.Tables) == 0 {
		cfg.Tables = []string{EventTable}
	}
	if cfg.Month > 0 && cfg.Day > 0 {
		where = append(where, "(Month < ? or (Month=? and Day<=?))")
		month := strconv.Itoa(cfg.Month)
		args = append(args, month, month, strconv.Itoa(cfg.Day))
	}
	if len(cfg.Years) > 0 {
		where = append(where, "(Year="+strings.Repeat("? or Year=", len(cfg.Years)-1)+"?)")
		for _, y := range cfg.Years {
			args = append(args, strconv.Itoa(y))
		}
	}
	if cfg.Amount != "" {
		where = append(where, "Amount "+cfg.Amount)
	}
	if cfg.ExcludeLabels != nil {
		for _, label := range cfg.ExcludeLabels {
			where = append(where, fmt.Sprintf("Labels NOT LIKE '%%%s%%'", label))
		}
	}
	condition := ""
	if len(where) > 0 {
		condition = " where " + strings.Join(where, " and ")
	}
	ifArgs := make([]interface{}, len(args))
	for i, v := range args {
		ifArgs[i] = v
	}
	return fmt.Sprintf(
		"select %s from %s%s%s", strings.Join(fields, ","), strings.Join(cfg.Tables, ","),
		condition, sortingOrder(cfg.Order),
	), ifArgs
}

func sortingOrder(order *OrderConfig) string {
	sorting := ""
	if order != nil {
		if order.GroupBy != nil {
			sorting += " group by " + strings.Join(order.GroupBy, ",")
		}
		if order.OrderBy != nil {
			sorting += " order by " + strings.Join(order.OrderBy, ",")
		}
		if order.Limit > 0 {
			sorting += " limit " + strconv.FormatInt(int64(order.Limit), 10)
		}
	}
	return sorting
}

func (sq *Sqlite3) Query(ctx context.Context, fields []string, opts ...QueryOption) (*sql.Rows, error) {
	if sq.db == nil {
		return nil, errors.New("database is nil")
	}
	query, values := sqlQuery(fields, opts...)
	// fmt.Println(query)
	return sq.db.QueryContext(ctx, query, values...)
}

func (sq *Sqlite3) Close() error {
	if sq.db != nil {
		return sq.db.Close()
	}
	return nil
}
