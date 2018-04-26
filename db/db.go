/*
 * Copyright 2018 yubo. All rights reserved.
 * Use of this source code is governed by a BSD-style
 * license that can be found in the LICENSE file.
 */

package db

import (
	"database/sql"
	"errors"
	"reflect"
	"strconv"
	"time"
)

const (
	MAX_ROWS = 1000
)

var OnDebug = false

type Db struct {
	Db *sql.DB
}

/**
 * Add New sql.DB in the future i will add ConnectionPool.Get()
 */
func Open(driverName, dataSourceName string) (*Db, error) {
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	return &Db{Db: db}, nil
}

func (p *Db) Close() {
	p.Db.Close()
}

func (p *Db) Query(query string, args ...interface{}) *Rows {
	ret := &Rows{}
	ret.rows, ret.err = p.Db.Query(query, args...)
	return ret
}

type Rows struct {
	rows *sql.Rows
	err  error
}

func (p *Rows) Row(dest ...interface{}) error {
	var (
		err error
	)

	if p.err != nil {
		return err
	}
	defer p.rows.Close()

	if !isStructMode(dest...) {
		if p.rows.Next() {
			return p.rows.Scan(dest...)
		}

	}

	if err != nil {
		return err
	}

	resultsSlice, err := p._rows(1)
	if err != nil {
		return err
	}

	if len(resultsSlice) == 0 {
		return sql.ErrNoRows
	}

	if len(resultsSlice) == 1 {
		results := resultsSlice[0]
		err := scanMapIntoStruct(dest[0], results)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Rows) Rows(dest interface{}) (err error) {

	if p.err != nil {
		return err
	}
	defer p.rows.Close()

	sliceValue := reflect.Indirect(reflect.ValueOf(dest))
	if sliceValue.Kind() != reflect.Slice {
		return errors.New("needs a pointer to a slice")
	}

	sliceElementType := sliceValue.Type().Elem()
	st := reflect.New(sliceElementType)

	if !isStructMode(st.Interface()) {
		for p.rows.Next() {
			newValue := reflect.New(sliceElementType)
			if err = p.rows.Scan(newValue.Interface()); err != nil {
				return err
			}
			sliceValue.Set(reflect.Append(sliceValue, reflect.Indirect(reflect.ValueOf(newValue.Interface()))))
		}
		return nil
	}

	if err != nil {
		return err
	}

	// If we've already specific columns with Select(), use that
	resultsSlice, err := p._rows(MAX_ROWS)
	if err != nil {
		return err
	}

	for _, results := range resultsSlice {
		newValue := reflect.New(sliceElementType)
		err := scanMapIntoStruct(newValue.Interface(), results)
		if err != nil {
			return err
		}
		sliceValue.Set(reflect.Append(sliceValue, reflect.Indirect(reflect.ValueOf(newValue.Interface()))))
	}
	return nil
}

func (p *Rows) _rows(limit int) (resultsSlice []map[string][]byte, err error) {

	fields, err := p.rows.Columns()
	if err != nil {
		return nil, err
	}
	for p.rows.Next() && limit > 0 {
		result := make(map[string][]byte)
		var scanResultContainers []interface{}
		for i := 0; i < len(fields); i++ {
			var scanResultContainer interface{}
			scanResultContainers = append(scanResultContainers, &scanResultContainer)
		}
		if err := p.rows.Scan(scanResultContainers...); err != nil {
			return nil, err
		}
		for ii, key := range fields {
			rawValue := reflect.Indirect(reflect.ValueOf(scanResultContainers[ii]))
			//if row is null then ignore
			if rawValue.Interface() == nil {
				continue
			}
			aa := reflect.TypeOf(rawValue.Interface())
			vv := reflect.ValueOf(rawValue.Interface())
			var str string
			switch aa.Kind() {
			case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				str = strconv.FormatInt(vv.Int(), 10)
				result[key] = []byte(str)
			case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				str = strconv.FormatUint(vv.Uint(), 10)
				result[key] = []byte(str)
			case reflect.Float32, reflect.Float64:
				str = strconv.FormatFloat(vv.Float(), 'f', -1, 64)
				result[key] = []byte(str)
			case reflect.Slice:
				if aa.Elem().Kind() == reflect.Uint8 {
					result[key] = rawValue.Interface().([]byte)
					break
				}
			case reflect.String:
				str = vv.String()
				result[key] = []byte(str)
			case reflect.Struct:
				str = rawValue.Interface().(time.Time).Format("2006-01-02 15:04:05.000 -0700")
				result[key] = []byte(str)
			case reflect.Bool:
				if vv.Bool() {
					result[key] = []byte("1")
				} else {
					result[key] = []byte("0")
				}
			}
		}
		resultsSlice = append(resultsSlice, result)
		limit--
	}

	return resultsSlice, nil
}

//Execute sql
func (p *Db) Exec(sql string, args ...interface{}) (sql.Result, error) {
	return p.Db.Exec(sql, args...)

}

func (p *Db) ExecLastId(sql string, args ...interface{}) (int64, error) {
	res, err := p.Db.Exec(sql, args...)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (p *Db) ExecNum(sql string, args ...interface{}) (int64, error) {
	res, err := p.Db.Exec(sql, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
