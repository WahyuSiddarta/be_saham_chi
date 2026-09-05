package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
)

// This driver exercises database/sql and sqlx mapping without a live database.
type queryResult struct {
	contains string
	columns  []string
	values   []driver.Value
	err      error
}
type testConnection struct {
	results               []queryResult
	queries               []string
	arguments             [][]driver.NamedValue
	committed, rolledBack bool
}
type testConnector struct{ connection *testConnection }

func (c testConnector) Connect(context.Context) (driver.Conn, error) { return c.connection, nil }
func (c testConnector) Driver() driver.Driver                        { return testDriver{} }

type testDriver struct{}

func (testDriver) Open(string) (driver.Conn, error) { return nil, errors.New("use connector") }
func (c *testConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}
func (c *testConnection) Close() error                                                 { return nil }
func (c *testConnection) Begin() (driver.Tx, error)                                    { return c, nil }
func (c *testConnection) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) { return c, nil }
func (c *testConnection) Commit() error                                                { c.committed = true; return nil }
func (c *testConnection) Rollback() error                                              { c.rolledBack = true; return nil }
func (c *testConnection) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.queries = append(c.queries, query)
	c.arguments = append(c.arguments, args)
	if len(c.results) == 0 {
		return nil, errors.New("unexpected query: " + query)
	}
	result := c.results[0]
	c.results = c.results[1:]
	if !strings.Contains(query, result.contains) {
		return nil, errors.New("unexpected query: " + query)
	}
	if result.err != nil {
		return nil, result.err
	}
	return &testRows{columns: result.columns, values: result.values}, nil
}

type testRows struct {
	columns []string
	values  []driver.Value
	done    bool
}

func (r *testRows) Columns() []string { return r.columns }
func (r *testRows) Close() error      { return nil }
func (r *testRows) Next(dest []driver.Value) error {
	if r.done || r.values == nil {
		return io.EOF
	}
	r.done = true
	copy(dest, r.values)
	return nil
}
func testRepository(t *testing.T, results ...queryResult) (*Repository, *testConnection) {
	t.Helper()
	conn := &testConnection{results: results}
	db := sqlx.NewDb(sql.OpenDB(testConnector{conn}), "test")
	t.Cleanup(func() { _ = db.Close() })
	return New(db), conn
}

func TestRegisterCommitsUserAndMainPortfolioTogether(t *testing.T) {
	now := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	for _, failPortfolio := range []bool{false, true} {
		t.Run(map[bool]string{false: "commit", true: "rollback"}[failPortfolio], func(t *testing.T) {
			portfolio := queryResult{contains: "INSERT INTO portfolios", columns: []string{"portfolio_id", "user_id", "name", "base_currency_code", "is_main", "created_at", "updated_at"}, values: []driver.Value{"p-1", "u-1", "main", "IDR", true, now, now}}
			if failPortfolio {
				portfolio.err = errors.New("portfolio insert failed")
			}
			repo, conn := testRepository(t, queryResult{contains: "INSERT INTO t_user", columns: []string{"created_at"}, values: []driver.Value{now}}, portfolio)
			user, created, err := repo.Register(context.Background(), "person@example.com", "password")
			if failPortfolio {
				if err == nil || conn.committed || !conn.rolledBack {
					t.Fatalf("err=%v commit=%v rollback=%v", err, conn.committed, conn.rolledBack)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !conn.committed || conn.rolledBack || user.ID == "" || created.PortfolioID != "p-1" || !created.IsMain {
				t.Fatalf("user=%+v portfolio=%+v commit=%v rollback=%v", user, created, conn.committed, conn.rolledBack)
			}
			if conn.arguments[0][2].Value != hashPassword("password", user.ID) {
				t.Fatal("password format changed")
			}
			if conn.arguments[1][0].Value != user.ID {
				t.Fatal("main portfolio belongs to wrong user")
			}
		})
	}
}

func TestLoginUsesExistingRoleAndLoadsACL(t *testing.T) {
	now := time.Now().UTC()
	repo, _ := testRepository(t,
		queryResult{contains: "FROM t_user", columns: []string{"user_id", "email", "password", "created_at", "role", "role_id", "status"}, values: []driver.Value{"u-1", "person@example.com", hashPassword("password", "u-1"), now, int64(2), int64(1), true}},
		queryResult{contains: "FROM t_role_acl_rule", columns: []string{"code"}, values: []driver.Value{"stock.manage"}},
	)
	user, err := repo.Login(context.Background(), "person@example.com", "password")
	if err != nil {
		t.Fatal(err)
	}
	if user.RoleID != 2 || len(user.Rules) != 1 || user.Rules[0] != "stock.manage" {
		t.Fatalf("user=%+v", user)
	}
}

func TestCashTransactionMappingPreservesAmountsAndOwnership(t *testing.T) {
	now := time.Now().UTC()
	repo, conn := testRepository(t, queryResult{
		contains: "FROM portfolio_transactions",
		columns:  []string{"transaction_id", "portfolio_id", "account_id", "account_name", "asset_id", "transaction_type", "transaction_date", "amount", "cost_amount", "currency_code", "notes", "created_at", "updated_at"},
		values:   []driver.Value{"tx-1", "p-1", "account-1", "Bank", "asset-1", "deposit", now, float64(150), float64(100), "IDR", "note", now, now},
	})
	item, err := repo.GetCashTransaction(context.Background(), "owner-1", "p-1", "tx-1")
	if err != nil {
		t.Fatal(err)
	}
	if item.Amount != 150 || item.CostAmount != 100 || item.AccountName != "Bank" {
		t.Fatalf("transaction=%+v", item)
	}
	if !strings.Contains(conn.queries[0], "p.user_id = $3") || conn.arguments[0][2].Value != "owner-1" {
		t.Fatal("ownership filter lost")
	}
}

func TestTransactionNotFoundMapping(t *testing.T) {
	for _, kind := range []string{"cash", "bond", "gold"} {
		t.Run(kind, func(t *testing.T) {
			repo, _ := testRepository(t, queryResult{contains: "FROM portfolio_transactions", columns: []string{"transaction_id"}})
			var err, want error
			switch kind {
			case "cash":
				_, err = repo.GetCashTransaction(context.Background(), "u", "p", "tx")
				want = ErrCashTransactionNotFound
			case "bond":
				_, err = repo.GetBondTransaction(context.Background(), "u", "p", "tx")
				want = ErrBondTransactionNotFound
			case "gold":
				_, err = repo.GetGoldTransaction(context.Background(), "u", "p", "tx")
				want = ErrGoldTransactionNotFound
			}
			if !errors.Is(err, want) {
				t.Fatalf("error=%v want=%v", err, want)
			}
		})
	}
}
