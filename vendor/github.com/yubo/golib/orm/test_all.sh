#!/bin/bash
#TEST_DB_DRIVER=mysql TEST_DB_DSN='root:12341234@tcp(localhost:3306)/test?parseTime=true&timeout=30s' go test -v
#TEST_DB_DRIVER='sqlite3' TEST_DB_DSN='file:/tmp/test1.db' go test -v
#TEST_DB_DRIVER='sqlite3' TEST_DB_DSN='file:test2.db?cache=shared&mode=memory' go test -v -test.run TestQueryRowsStructPtr -args -v 10 -logtostderr true
#TEST_DB_DRIVER='sqlite3' TEST_DB_DSN='file:test2.db?cache=shared&mode=memory' go test -v -args -v 10 -logtostderr true
#TEST_DB_DRIVER='sqlite3' TEST_DB_DSN='file:test2.db?cache=shared&mode=memory' go test -v
go test -v -test.run TestUpdateSql
