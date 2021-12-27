package qframe_test

import (
	"database/sql"
	"database/sql/driver"
	"io"
	"testing"

	"github.com/yistabraq/qframe"
	qsql "github.com/yistabraq/qframe/config/sql"
)

// MockDriver implements a fake SQL driver for testing.
type MockDriver struct {
	t *testing.T
	// expected SQL query
	query string
	// results holds values that are
	// returned from a database query
	results struct {
		// column names for each row of values
		columns []string
		// each value for each row
		values [][]driver.Value
	}
	// args holds expected values
	args struct {
		// values we expect to be given
		// to the database
		values [][]driver.Value
	}

	// optional statement Query implementation
	mockQuery MockQuery
}

func (m MockDriver) Open(name string) (driver.Conn, error) {
	stmt := &MockStmt{
		t:      m.t,
		values: m.args.values,
		rows: &MockRows{
			t:       m.t,
			columns: m.results.columns,
			values:  m.results.values,
		},
		mockQuery: m.mockQuery,
	}
	return &MockConn{
		t:     m.t,
		stmt:  stmt,
		query: m.query,
	}, nil
}

type MockRows struct {
	t       *testing.T
	idx     int
	columns []string
	values  [][]driver.Value
}

func (m *MockRows) Next(dest []driver.Value) error {
	if m.idx == len(m.values) {
		return io.EOF
	}
	for i := 0; i < len(dest); i++ {
		dest[i] = m.values[m.idx][i]
	}
	m.idx++
	return nil
}

func (m MockRows) Close() error { return nil }

func (m MockRows) Columns() []string { return m.columns }

type MockTx struct{}

func (m MockTx) Commit() error { return nil }

func (m MockTx) Rollback() error { return nil }

type MockStmt struct {
	t         *testing.T
	rows      *MockRows
	idx       int
	values    [][]driver.Value
	mockQuery MockQuery
}

func (s MockStmt) Close() error { return nil }

func (s MockStmt) NumInput() int {
	if len(s.values) > 0 {
		return len(s.values[0])
	}
	return 0
}

func (s *MockStmt) Exec(args []driver.Value) (driver.Result, error) {
	for i, arg := range args {
		if s.values[s.idx][i] != arg {
			s.t.Errorf("arg %t != %t", arg, s.values[s.idx][i])
		}
	}
	s.idx++
	return nil, nil
}

func (s MockStmt) Query(args []driver.Value) (driver.Rows, error) {
	// use the mock query implementation if supplied by the test
	if s.mockQuery != nil {
		return s.mockQuery(args)
	}
	return s.rows, nil
}

type MockQuery func(args []driver.Value) (driver.Rows, error)

type MockConn struct {
	t     *testing.T
	query string
	stmt  *MockStmt
}

func (m MockConn) Prepare(query string) (driver.Stmt, error) {
	if query != m.query {
		m.t.Errorf("invalid query: %s != %s", query, m.query)
	}
	return m.stmt, nil
}

func (c MockConn) Close() error { return nil }
func (c MockConn) Begin() (driver.Tx, error) {
	return &MockTx{}, nil
}

var (
	_ driver.Conn = (*MockConn)(nil)
	_ driver.Rows = (*MockRows)(nil)
	_ driver.Tx   = (*MockTx)(nil)
	_ driver.Stmt = (*MockStmt)(nil)
	_ driver.Conn = (*MockConn)(nil)
)

func TestQFrame_ToSQL(t *testing.T) {
	dvr := MockDriver{t: t}
	dvr.query = "INSERT INTO test (COL1,COL2,COL3,COL4) VALUES (?,?,?,?);"
	dvr.args.values = [][]driver.Value{
		{int64(1), 1.1, "one", true},
		{int64(2), 2.2, "two", true},
		{int64(3), 3.3, "three", false},
	}
	sql.Register("TestToSQL", dvr)
	db, _ := sql.Open("TestToSQL", "")
	tx, _ := db.Begin()
	qf := qframe.New(map[string]interface{}{
		"COL1": []int{1, 2, 3},
		"COL2": []float64{1.1, 2.2, 3.3},
		"COL3": []string{"one", "two", "three"},
		"COL4": []bool{true, true, false},
	})
	assertNotErr(t, qf.ToSQL(tx, qsql.Table("test")))
}

func TestQFrame_ReadSQL(t *testing.T) {
	dvr := MockDriver{t: t}
	dvr.results.columns = []string{"COL1", "COL2", "COL3", "COL4"}
	dvr.results.values = [][]driver.Value{
		{int64(1), 1.1, "one", true},
		{int64(2), 2.2, "two", true},
		{int64(3), 3.3, "three", false},
	}
	sql.Register("TestReadSQL", dvr)
	db, _ := sql.Open("TestReadSQL", "")
	tx, _ := db.Begin()
	qf := qframe.ReadSQL(tx)
	assertNotErr(t, qf.Err)
	expected := qframe.New(map[string]interface{}{
		"COL1": []int{1, 2, 3},
		"COL2": []float64{1.1, 2.2, 3.3},
		"COL3": []string{"one", "two", "three"},
		"COL4": []bool{true, true, false},
	})
	assertEquals(t, expected, qf)
}

func TestQFrame_ReadSQLCoercion(t *testing.T) {
	dvr := MockDriver{t: t}
	dvr.results.columns = []string{"COL1", "COL2"}
	dvr.results.values = [][]driver.Value{
		{int64(1), int64(0)},
		{int64(1), int64(0)},
		{int64(0), int64(1)},
	}
	sql.Register("TestReadSQLCoercion", dvr)
	db, _ := sql.Open("TestReadSQLCoercion", "")
	tx, _ := db.Begin()
	qf := qframe.ReadSQL(tx, qsql.Coerce(
		qsql.CoercePair{Column: "COL1", Type: qsql.Int64ToBool},
		qsql.CoercePair{Column: "COL2", Type: qsql.Int64ToBool},
	))
	assertNotErr(t, qf.Err)
	expected := qframe.New(map[string]interface{}{
		"COL1": []bool{true, true, false},
		"COL2": []bool{false, false, true},
	})
	assertEquals(t, expected, qf)
}

func TestQFrame_ReadWithArgs(t *testing.T) {
	dvr := MockDriver{t: t}
	dvr.args.values = [][]driver.Value{{""}}

	// mock rows indexed by string COL3
	indexRows := map[string][][]driver.Value{
		"one":   {{int64(1), 1.1, "one", true}},
		"two":   {{int64(2), 1.2, "two", false}},
		"three": {{int64(3), 1.3, "three", true}},
	}
	dvr.mockQuery = func(args []driver.Value) (driver.Rows, error) {
		// to confirm our argument made it through to the db driver as expected
		if len(args) != 1 {
			t.Error("expecting one argument in query invocation")
		}
		val, valid := args[0].(string)
		if !valid {
			t.Error("expecting argument in query invocation to be a string")
		}
		matching, has := indexRows[val]
		if !has {
			matching = [][]driver.Value{}
		}
		return &MockRows{
			t:       t,
			columns: []string{"COL1", "COL2", "COL3", "COL4"},
			values:  matching,
		}, nil
	}
	stmt := "SELECT * FROM mock_table WHERE COL3=$1"
	dvr.query = stmt
	sql.Register("TestReadPrepared", dvr)
	db, _ := sql.Open("TestReadPrepared", "")
	tx, _ := db.Begin()
	qf := qframe.ReadSQLWithArgs(tx, []interface{}{"one"}, qsql.Query(stmt))
	assertNotErr(t, qf.Err)
	expected := qframe.New(map[string]interface{}{
		"COL1": []int{1},
		"COL2": []float64{1.1},
		"COL3": []string{"one"},
		"COL4": []bool{true},
	})
	assertEquals(t, expected, qf)
}
