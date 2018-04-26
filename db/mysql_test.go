/*
 * Copyright 2018 yubo. All rights reserved.
 * Use of this source code is governed by a BSD-style
 * license that can be found in the LICENSE file.
 */

package db

import (
	"database/sql"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var (
	user      string
	pass      string
	prot      string
	addr      string
	dbname    string
	dsn       string
	netAddr   string
	available bool
)

var (
	tDate      = time.Date(2012, 6, 14, 0, 0, 0, 0, time.UTC)
	sDate      = "2012-06-14"
	tDateTime  = time.Date(2011, 11, 20, 21, 27, 37, 0, time.UTC)
	sDateTime  = "2011-11-20 21:27:37"
	tDate0     = time.Time{}
	sDate0     = "0000-00-00"
	sDateTime0 = "0000-00-00 00:00:00"
)

// See https://github.com/go-sql-driver/mysql/wiki/Testing
func init() {
	// get environment variables
	env := func(key, defaultValue string) string {
		if value := os.Getenv(key); value != "" {
			return value
		}
		return defaultValue
	}
	user = env("MYSQL_TEST_USER", "root")
	pass = env("MYSQL_TEST_PASS", "12341234")
	prot = env("MYSQL_TEST_PROT", "tcp")
	addr = env("MYSQL_TEST_ADDR", "localhost:3306")
	dbname = env("MYSQL_TEST_DBNAME", "test")
	netAddr = fmt.Sprintf("%s(%s)", prot, addr)
	dsn = fmt.Sprintf("%s:%s@%s/%s?timeout=30s", user, pass, netAddr, dbname)
	c, err := net.Dial(prot, addr)
	if err == nil {
		available = true
		c.Close()
	}
}

type DBTest struct {
	*testing.T
	db *Db
}

func runTests(t *testing.T, dsn string, tests ...func(dbt *DBTest)) {
	var (
		err error
		db  *Db
		dbt *DBTest
	)

	if !available {
		t.Skipf("MySQL server not running on %s", netAddr)
	}

	db, err = Open("mysql", dsn)
	if err != nil {
		t.Fatalf("error connecting: %s", err.Error())
	}
	defer db.Close()

	db.Exec("DROP TABLE IF EXISTS test")

	dbt = &DBTest{t, db}
	for _, test := range tests {
		test(dbt)
		dbt.db.Exec("DROP TABLE IF EXISTS test")
	}
}

func (dbt *DBTest) fail(method, query string, err error) {
	if len(query) > 300 {
		query = "[query too large to print]"
	}
	dbt.Fatalf("error on %s %s: %s", method, query, err.Error())
}

func (dbt *DBTest) mustQueryRow(output interface{}, query string, args ...interface{}) {
	err := dbt.db.Query(query, args...).Row(output)
	if err != nil {
		dbt.fail("query row", query, err)
	}
}
func (dbt *DBTest) mustQueryRows(output interface{}, query string, args ...interface{}) {
	err := dbt.db.Query(query, args...).Rows(output)
	if err != nil {
		dbt.fail("query rows", query, err)
	}
}
func (dbt *DBTest) mustExec(query string, args ...interface{}) (res sql.Result) {
	res, err := dbt.db.Exec(query, args...)
	if err != nil {
		dbt.fail("exec", query, err)
	}
	return res
}

func TestInsert(t *testing.T) {
	runTests(t, dsn, func(dbt *DBTest) {
		var v int
		dbt.mustExec("CREATE TABLE test (value int)")

		dbt.mustExec("INSERT INTO test VALUES (?)", 1)

		dbt.mustQueryRow(&v, "SELECT value FROM test")

		dbt.mustExec("DROP TABLE IF EXISTS test")
	})
}

func TestQueryRows(t *testing.T) {
	runTests(t, dsn, func(dbt *DBTest) {
		var v []int
		dbt.mustExec("CREATE TABLE test (value int)")

		dbt.mustExec("INSERT INTO test VALUES (?), (?), (?)", 1, 2, 3)

		dbt.mustQueryRows(&v, "SELECT value FROM test")

		if len(v) != 3 {
			t.Fatalf("query rows want 3 got %d", len(v))
		}

		dbt.mustExec("DROP TABLE IF EXISTS test")
	})
}

func TestQueryRowsStruct(t *testing.T) {
	runTests(t, dsn, func(dbt *DBTest) {
		var v []struct {
			PointX int64
			PointY int64 `sql:"point_y"`
		}

		dbt.mustExec("CREATE TABLE test (point_x int, point_y int)")

		dbt.mustExec("INSERT INTO test VALUES (?, ?), (?, ?), (?, ?)", 1, 2, 3, 4, 5, 6)

		dbt.mustQueryRows(&v, "SELECT * FROM test")

		if len(v) != 3 {
			t.Fatalf("query rows want 3 got %d", len(v))
		}
		if v[2].PointX != 5 {
			t.Fatalf("v[2].PointX want 5 got %d", v[2].PointX)
		}
		if v[2].PointY != 6 {
			t.Fatalf("v[2].PointY want 6 got %d", v[2].PointY)
		}

		dbt.mustExec("DROP TABLE IF EXISTS test")
	})
}

func TestPing(t *testing.T) {
	runTests(t, dsn, func(dbt *DBTest) {
		if err := dbt.db.Db.Ping(); err != nil {
			dbt.fail("Ping", "Ping", err)
		}
	})
}
