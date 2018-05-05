/*
 * Copyright 2018 yubo. All rights reserved.
 * Use of this source code is governed by a BSD-style
 * license that can be found in the LICENSE file.
 */

package core

import (
	"database/sql"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang/glog"
	"golang.org/x/net/context"
)

const (
	MAX_ROWS = 1000
)

var (
	ErrNoStruct = errors.New("expected a pointer to a struct")
	OnDebug     = false
)

type Db struct {
	Db *sql.DB
}

func DbOpen(driverName, dataSourceName string) (*Db, error) {
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}

	return &Db{Db: db}, nil
}

func DbOpenWithCtx(driverName, dataSourceName string, ctx context.Context) (*Db, error) {
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		db.Close()
	}()

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

func (p *Rows) Row(dest ...interface{}) (err error) {
	if p.err != nil {
		return p.err
	}
	defer p.rows.Close()

	if !isStructMode(dest...) {
		if p.rows.Next() {
			return p.rows.Scan(dest...)
		}
		return sql.ErrNoRows
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
		err = sql.ErrNoRows
		for p.rows.Next() {
			newValue := reflect.New(sliceElementType)
			if err = p.rows.Scan(newValue.Interface()); err != nil {
				return
			}
			sliceValue.Set(reflect.Append(sliceValue, reflect.Indirect(reflect.ValueOf(newValue.Interface()))))
		}
		return
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
		glog.V(3).Infof("%#v", err)
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

// utils

func getTypeName(obj interface{}) (typestr string) {
	typ := reflect.TypeOf(obj)
	typestr = typ.String()

	lastDotIndex := strings.LastIndex(typestr, ".")
	if lastDotIndex != -1 {
		typestr = typestr[lastDotIndex+1:]
	}

	return
}

func snakeCasedName(name string) string {
	newstr := make([]rune, 0)
	firstTime := true

	for _, chr := range name {
		if isUpper := 'A' <= chr && chr <= 'Z'; isUpper {
			if firstTime == true {
				firstTime = false
			} else {
				newstr = append(newstr, '_')
			}
			chr -= ('A' - 'a')
		}
		newstr = append(newstr, chr)
	}

	return string(newstr)
}

func titleCasedName(name string) string {
	newstr := make([]rune, 0)
	upNextChar := true

	for _, chr := range name {
		switch {
		case upNextChar:
			upNextChar = false
			chr -= ('a' - 'A')
		case chr == '_':
			upNextChar = true
			continue
		}

		newstr = append(newstr, chr)
	}

	return string(newstr)
}

func pluralizeString(str string) string {
	if strings.HasSuffix(str, "data") {
		return str
	}
	if strings.HasSuffix(str, "y") {
		str = str[:len(str)-1] + "ie"
	}
	return str + "s"
}

func scanMapIntoStruct(obj interface{}, objMap map[string][]byte) error {
	dataStruct := reflect.Indirect(reflect.ValueOf(obj))
	if dataStruct.Kind() != reflect.Struct {
		return errors.New("expected a pointer to a struct")
	}

	dataStructType := dataStruct.Type()

	for i := 0; i < dataStructType.NumField(); i++ {
		field := dataStructType.Field(i)
		fieldv := dataStruct.Field(i)

		err := scanMapElement(fieldv, field, objMap)
		if err != nil {
			return err
		}
	}

	return nil
}

// snake string, XxYy to xx_yy , XxYY to xx_yy
func snakeString(s string) string {
	data := make([]byte, 0, len(s)*2)
	j := false
	num := len(s)
	for i := 0; i < num; i++ {
		d := s[i]
		if i > 0 && d >= 'A' && d <= 'Z' && j {
			data = append(data, '_')
		}
		if d != '_' {
			j = true
		}
		data = append(data, d)
	}
	return strings.ToLower(string(data[:]))
}

func scanMapElement(fieldv reflect.Value, field reflect.StructField, objMap map[string][]byte) error {

	objFieldName := snakeString(field.Name)
	bb := field.Tag
	sqlTag := bb.Get("sql")

	if bb.Get("beedb") == "-" || sqlTag == "-" || reflect.ValueOf(bb).String() == "-" {
		return nil
	}
	sqlTags := strings.Split(sqlTag, ",")
	sqlFieldName := objFieldName
	if len(sqlTags[0]) > 0 {
		sqlFieldName = sqlTags[0]
	}
	inline := false
	//omitempty := false //TODO!
	// CHECK INLINE
	if len(sqlTags) > 1 {
		if stringArrayContains("inline", sqlTags[1:]) {
			inline = true
		}
	}
	if inline {
		if field.Type.Kind() == reflect.Struct && field.Type.String() != "time.Time" {
			for i := 0; i < field.Type.NumField(); i++ {
				err := scanMapElement(fieldv.Field(i), field.Type.Field(i), objMap)
				if err != nil {
					return err
				}
			}
		} else {
			return errors.New("A non struct type can't be inline.")
		}
	}

	// not inline

	data, ok := objMap[sqlFieldName]

	if !ok {
		return nil
	}

	var v interface{}

	switch field.Type.Kind() {

	case reflect.Slice:
		v = data
	case reflect.String:
		v = string(data)
	case reflect.Bool:
		v = string(data) == "1"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		x, err := strconv.Atoi(string(data))
		if err != nil {
			return errors.New("arg " + sqlFieldName + " as int: " + err.Error())
		}
		v = x
	case reflect.Int64:
		x, err := strconv.ParseInt(string(data), 10, 64)
		if err != nil {
			return errors.New("arg " + sqlFieldName + " as int: " + err.Error())
		}
		v = x
	case reflect.Float32, reflect.Float64:
		x, err := strconv.ParseFloat(string(data), 64)
		if err != nil {
			return errors.New("arg " + sqlFieldName + " as float64: " + err.Error())
		}
		v = x
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		x, err := strconv.ParseUint(string(data), 10, 64)
		if err != nil {
			return errors.New("arg " + sqlFieldName + " as int: " + err.Error())
		}
		v = x
	//Supports Time type only (for now)
	case reflect.Struct:
		if fieldv.Type().String() != "time.Time" {
			return errors.New("unsupported struct type in Scan: " + fieldv.Type().String())
		}

		x, err := time.Parse("2006-01-02 15:04:05", string(data))
		if err != nil {
			x, err = time.Parse("2006-01-02 15:04:05.000 -0700", string(data))

			if err != nil {
				return errors.New("unsupported time format: " + string(data))
			}
		}

		v = x
	default:
		return errors.New("unsupported type in Scan: " + reflect.TypeOf(v).String())
	}

	fieldv.Set(reflect.ValueOf(v))

	return nil
}

func isStructMode(objs ...interface{}) bool {
	if len(objs) != 1 {
		return false
	}

	dataStruct := reflect.Indirect(reflect.ValueOf(objs[0]))
	if dataStruct.Kind() == reflect.Struct && dataStruct.String() != "time.Time" {
		return true
	} else {
		return false
	}

}

func scanStructIntoMap(objs ...interface{}) (map[string]interface{}, error) {
	if len(objs) != 1 {
		return nil, ErrNoStruct
	}

	dataStruct := reflect.Indirect(reflect.ValueOf(objs[0]))
	if dataStruct.Kind() != reflect.Struct {
		return nil, ErrNoStruct
	}

	dataStructType := dataStruct.Type()

	mapped := make(map[string]interface{})

	for i := 0; i < dataStructType.NumField(); i++ {
		field := dataStructType.Field(i)
		fieldv := dataStruct.Field(i)
		fieldName := field.Name
		bb := field.Tag
		sqlTag := bb.Get("sql")
		sqlTags := strings.Split(sqlTag, ",")
		var mapKey string

		inline := false

		if bb.Get("beedb") == "-" || sqlTag == "-" || reflect.ValueOf(bb).String() == "-" {
			continue
		} else if len(sqlTag) > 0 {
			//TODO: support tags that are common in json like omitempty
			if sqlTags[0] == "-" {
				continue
			}
			mapKey = sqlTags[0]
		} else {
			mapKey = fieldName
		}

		if len(sqlTags) > 1 {
			if stringArrayContains("inline", sqlTags[1:]) {
				inline = true
			}
		}

		if inline {
			// get an inner map and then put it inside the outer map
			map2, err2 := scanStructIntoMap(fieldv.Interface())
			if err2 != nil {
				return mapped, err2
			}
			for k, v := range map2 {
				mapped[k] = v
			}
		} else {
			value := dataStruct.FieldByName(fieldName).Interface()
			mapped[mapKey] = value
		}
	}

	return mapped, nil
}

func StructName(s interface{}) string {
	v := reflect.TypeOf(s)
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v.Name()
}

func getTableName(s interface{}) string {
	v := reflect.TypeOf(s)
	if v.Kind() == reflect.String {
		s2, _ := s.(string)
		return snakeCasedName(s2)
	}
	tn := scanTableName(s)
	if len(tn) > 0 {
		return tn
	}
	return getTableName(StructName(s))
}

func scanTableName(s interface{}) string {
	if reflect.TypeOf(reflect.Indirect(reflect.ValueOf(s)).Interface()).Kind() == reflect.Slice {
		sliceValue := reflect.Indirect(reflect.ValueOf(s))
		sliceElementType := sliceValue.Type().Elem()
		for i := 0; i < sliceElementType.NumField(); i++ {
			bb := sliceElementType.Field(i).Tag
			if len(bb.Get("tname")) > 0 {
				return bb.Get("tname")
			}
		}
	} else {
		tt := reflect.TypeOf(reflect.Indirect(reflect.ValueOf(s)).Interface())
		for i := 0; i < tt.NumField(); i++ {
			bb := tt.Field(i).Tag
			if len(bb.Get("tname")) > 0 {
				return bb.Get("tname")
			}
		}
	}
	return ""

}

func stringArrayContains(needle string, haystack []string) bool {
	for _, v := range haystack {
		if needle == v {
			return true
		}
	}
	return false
}
