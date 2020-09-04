package orm

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/yubo/golib/status"
	"github.com/yubo/golib/util"
	"google.golang.org/grpc/codes"
	"k8s.io/klog/v2"
)

const (
	MAX_ROWS = 1000
)

type db interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
}

type Db struct {
	Greatest string
	Db       *sql.DB
	tx       *sql.Tx
	db       db
}

func printString(b []byte) string {
	s := make([]byte, len(b))

	for i := 0; i < len(b); i++ {
		if strconv.IsPrint(rune(b[i])) {
			s[i] = b[i]
		} else {
			s[i] = '.'
		}
	}
	return string(s)
}

func dlog(format string, args ...interface{}) {
	if klog.V(3).Enabled() {
		klog.InfoDepth(2, fmt.Sprintf(format, args...))
	}
}

func dlogSql(query string, args ...interface{}) {
	if klog.V(3).Enabled() {
		args2 := make([]interface{}, len(args))

		for i := 0; i < len(args2); i++ {
			rv := reflect.Indirect(reflect.ValueOf(args[i]))
			if rv.IsValid() && rv.CanInterface() {
				if b, ok := rv.Interface().([]byte); ok {
					args2[i] = printString(b)
				} else {
					args2[i] = rv.Interface()
				}
			}
		}
		klog.InfoDepth(2, "\n\t"+fmt.Sprintf(strings.Replace(query, "?", "`%v`", -1), args2...))
	}
}

func DbOpen(driverName, dataSourceName string) (*Db, error) {
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}

	ret := &Db{Db: db, db: db, Greatest: "greatest"}

	if driverName == "sqlite3" {
		ret.Greatest = "max"
	}

	return ret, nil
}

func DbOpenWithCtx(driverName, dsn string, ctx context.Context) (*Db, error) {

	db, err := DbOpen(driverName, dsn)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "sql.Open() err: "+err.Error())
	}

	if err := db.Db.Ping(); err != nil {
		db.Db.Close()
		return nil, status.Errorf(codes.Internal, "db.Ping() err: "+err.Error())
	}

	go func() {
		<-ctx.Done()
		db.Db.Close()
	}()

	return db, nil
}

func (p *Db) Tx() bool {
	return p.tx != nil
}

func (p *Db) BeginWithCtx(ctx context.Context) (*Db, error) {
	if tx, err := p.Db.BeginTx(ctx, nil); err != nil {
		return nil, err
	} else {
		return &Db{tx: tx, db: tx, Greatest: p.Greatest}, nil
	}
}

func (p *Db) Rollback() error {
	if p.tx != nil {
		return p.tx.Rollback()
	}
	return status.Errorf(codes.Internal, "tx is nil")
}

func (p *Db) Commit() error {
	if p.tx != nil {
		return p.tx.Commit()
	}
	return status.Errorf(codes.Internal, "tx is nil")
}

func (p *Db) Begin() (*Db, error) {
	return p.BeginWithCtx(context.Background())
}

func (p *Db) SetConns(maxIdleConns, maxOpenConns int) {
	p.Db.SetMaxIdleConns(maxIdleConns)
	p.Db.SetMaxOpenConns(maxOpenConns)
}

func (p *Db) Close() {
	p.Db.Close()
}

func (p *Db) Query(query string, args ...interface{}) *Rows {
	dlogSql(query, args...)
	ret := &Rows{}
	ret.rows, ret.err = p.db.Query(query, args...)
	return ret
}

type Rows struct {
	rows *sql.Rows
	err  error
}

// Row(*int, *int, ...)
// Row(*struct{})
// Row(**struct{})
func (p *Rows) Row(dst ...interface{}) error {
	if p.err != nil {
		return p.err
	}
	defer p.rows.Close()

	if p.rows.Next() {
		if len(dst) == 1 && isStructMode(dst[0]) {
			// klog.V(5).Infof("enter row scan struct")
			return p.scanRow(dst[0])
		}

		// klog.V(5).Infof("enter row scan")
		return p.rows.Scan(dst...)
	}
	return status.Errorf(codes.NotFound, "sql: no rows in result set")
}

// scanRow scan row result into dst struct
// dst must be struct, should be prechecked by isStructMode()
func (p *Rows) scanRow(dst interface{}) error {
	rv := reflect.Indirect(reflect.ValueOf(dst))

	if !rv.CanSet() {
		return status.Errorf(codes.InvalidArgument, "scan target can not be set")
	}

	b, err := p.genBinder()
	if err != nil {
		return err
	}

	tran, err := b.bind(rv)
	if err != nil {
		return err
	}

	if err := p.rows.Scan(b.dest...); err != nil {
		return err
	}

	for _, v := range tran {
		if err := v.unmarshal(); err != nil {
			return err
		}
	}
	return nil
}

// Rows([]struct{})
// Rows([]*struct{})
// Rows(*[]struct{})
// Rows(*[]*struct{})
// Rows([]string)
// Rows([]*string)
func (p *Rows) Rows(dst interface{}, opts ...int) error {
	if p.err != nil {
		return p.err
	}
	defer p.rows.Close()

	limit := MAX_ROWS
	if len(opts) > 0 && opts[0] > 0 {
		limit = opts[0]
	}

	rv := reflect.Indirect(reflect.ValueOf(dst))

	if !rv.CanSet() {
		return status.Errorf(codes.InvalidArgument, "scan target can not be set")
	}

	// for *[]struct{}
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return status.Errorf(codes.Internal, "needs a pointer to a slice")
		}
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Slice {
		return status.Errorf(codes.Internal, "needs a pointer to a slice")
	}

	et := rv.Type().Elem()
	n := 0

	// et is slice elem type
	if isStructMode(reflect.New(et).Interface()) {

		b, err := p.genBinder()
		if err != nil {
			return err
		}

		for p.rows.Next() {
			row := reflect.New(et).Elem()

			extra, err := b.bind(row)
			if err != nil {
				return err
			}

			if err := p.rows.Scan(b.dest...); err != nil {
				return status.Errorf(codes.Internal, "Scan() err: "+err.Error())
			}

			for _, v := range extra {
				if err := v.unmarshal(); err != nil {
					return err
				}
			}

			rv.Set(reflect.Append(rv, row))

			if n += 1; n >= limit {
				break
			}
		}
	} else {
		// e.g. []string or []*string
		for p.rows.Next() {
			row := reflect.New(et).Elem()

			if err := p.rows.Scan(row.Addr().Interface()); err != nil {
				return status.Errorf(codes.Internal, "Scan() err: "+err.Error())
			}

			rv.Set(reflect.Append(rv, row))

			if n += 1; n >= limit {
				break
			}
		}
	}

	if n == 0 {
		return status.Errorf(codes.NotFound, "sql: no rows in result set")
	}

	return nil
}

func (p *Db) Exec(sql string, args ...interface{}) (sql.Result, error) {
	dlogSql(sql, args...)

	ret, err := p.db.Exec(sql, args...)
	if err != nil {
		klog.V(3).Info(1, err)
		return nil, status.Errorf(codes.Internal, "Exec() err: "+err.Error())
	}

	return ret, nil
}

func (p *Db) ExecErr(sql string, args ...interface{}) error {
	dlogSql(sql, args...)

	_, err := p.db.Exec(sql, args...)
	if err != nil {
		klog.InfoDepth(1, err)
	}
	return err
}

func (p *Db) ExecLastId(sql string, args ...interface{}) (int64, error) {
	// if p.Tx() {
	// 	return 0, status.Errorf(codes.Internal, "In TX mode, reading data is not supported")
	// }

	dlogSql(sql, args...)

	res, err := p.db.Exec(sql, args...)
	if err != nil {
		klog.InfoDepth(1, err)
		return 0, status.Errorf(codes.Internal, "Exec() err: "+err.Error())
	}

	if ret, err := res.LastInsertId(); err != nil {
		dlogSql("%v", err)
		return 0, status.Errorf(codes.Internal, "LastInsertId() err: "+err.Error())
	} else {
		return ret, nil
	}

}

func (p *Db) execNum(sql string, args ...interface{}) (int64, error) {
	res, err := p.db.Exec(sql, args...)
	if err != nil {
		dlogSql("%v", err)
		return 0, status.Errorf(codes.Internal, "Exec() err: "+err.Error())
	}

	if ret, err := res.RowsAffected(); err != nil {
		dlogSql("%v", err)
		return 0, status.Errorf(codes.Internal, "RowsAffected() err: "+err.Error())
	} else {
		return ret, nil
	}
}

func (p *Db) ExecNum(sql string, args ...interface{}) (int64, error) {
	dlogSql(sql, args...)
	return p.execNum(sql, args...)
}

func (p *Db) ExecNumErr(s string, args ...interface{}) error {
	dlogSql(s, args...)
	if n, err := p.execNum(s, args...); err != nil {
		return err
	} else if n == 0 {
		return status.Errorf(codes.NotFound, "no rows affected")
	} else {
		return nil
	}
}

func (p *Db) ExecRows(bytes []byte) (err error) {
	var (
		cmds []string
		tx   *sql.Tx
	)

	if tx, err = p.Db.Begin(); err != nil {
		return status.Errorf(codes.Internal, "Begin() err: "+err.Error())
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	lines := strings.Split(string(bytes), "\n")
	for cmd, in, i := "", false, 0; i < len(lines); i++ {
		line := lines[i]
		if len(line) == 0 || strings.HasPrefix(line, "-- ") {
			continue
		}

		if in {
			cmd += " " + strings.TrimSpace(line)
			if cmd[len(cmd)-1] == ';' {
				cmds = append(cmds, cmd)
				in = false
			}
		} else {
			n := strings.Index(line, " ")
			if n <= 0 {
				continue
			}

			switch line[:n] {
			case "SET", "CREATE", "INSERT", "DROP":
				cmd = line
				if line[len(line)-1] == ';' {
					cmds = append(cmds, cmd)
				} else {
					in = true
				}
			}
		}
	}

	for i := 0; i < len(cmds); i++ {
		_, err := tx.Exec(cmds[i])
		if err != nil {
			klog.V(3).Infof("%v", err)
			return status.Errorf(codes.Internal, "sql %s\nerr %s", cmds[i], err.Error())
		}
	}
	return nil
}

func (p *Db) Update(table string, sample interface{}) error {
	sql, args, err := GenUpdateSql(table, sample)
	if err != nil {
		dlog("%v", err)
		return err
	}

	dlogSql(sql, args...)
	_, err = p.db.Exec(sql, args...)
	if err != nil {
		dlog("%v", err)
	}
	return err
}

// TODO: rename Insert
func (p *Db) Insert(table string, sample interface{}) error {
	sql, args, err := GenInsertSql(table, sample)
	if err != nil {
		return err
	}

	dlogSql(sql, args...)
	if _, err := p.db.Exec(sql, args...); err != nil {
		dlog("%v", err)
		return status.Errorf(codes.Internal,
			"Insert() err: "+err.Error())
	}
	return nil
}

func (p *Db) InsertLastId(table string, sample interface{}) (int64, error) {
	//if p.Tx() {
	//	return 0, status.Errorf(codes.Internal, "In TX mode, reading data is not supported")
	//}

	sql, args, err := GenInsertSql(table, sample)
	if err != nil {
		return 0, err
	}

	dlogSql(sql, args...)
	res, err := p.db.Exec(sql, args...)
	if err != nil {
		dlog("%v", err)
		return 0, status.Errorf(codes.Internal, "Exec() err: "+err.Error())
	}

	if ret, err := res.LastInsertId(); err != nil {
		dlog("%v", err)
		return 0, status.Errorf(codes.Internal, "LastInsertId() err: "+err.Error())
	} else {
		return ret, nil
	}
}

// utils

func getTypeName(obj interface{}) (typestr string) {
	typ := reflect.TypeOf(obj)
	typestr = typ.String()

	if lastDotIndex := strings.LastIndex(typestr, "."); lastDotIndex != -1 {
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

// {1,2,3} => "(1,2,3)"
func Ints2sql(array []int64) string {
	out := bytes.NewBuffer([]byte("("))

	for i := 0; i < len(array); i++ {
		if i > 0 {
			out.WriteByte(',')
		}
		fmt.Fprintf(out, "%d", array[i])
	}
	out.WriteByte(')')
	return out.String()
}

// {"1","2","3"} => "('1', '2', '3')"
func Strings2sql(array []string) string {
	out := bytes.NewBuffer([]byte("("))

	for i := 0; i < len(array); i++ {
		if i > 0 {
			out.WriteByte(',')
		}
		out.WriteByte('\'')
		out.Write([]byte(array[i]))
		out.WriteByte('\'')
	}
	out.WriteByte(')')
	return out.String()
}

func DsnSummary(dsn string) (string, error) {
	return dsn, nil
}

func stringArrayContains(needle string, haystack []string) bool {
	for _, v := range haystack {
		if needle == v {
			return true
		}
	}
	return false
}

func isStructMode(in interface{}) bool {
	rt := reflect.TypeOf(in)

	// depth 2
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	return rt.Kind() == reflect.Struct && rt.String() != "time.Time"
}

// sql:"-"
// sql:"foo_bar"
// sql:",inline"
func getTags(ff reflect.StructField) (name, extra string, skip, inline bool) {
	tag, _ := ff.Tag.Lookup("sql")
	if tag == "-" {
		skip = true
		return
	}

	if strings.HasSuffix(tag, ",inline") {
		inline = true
		return
	}

	tags := strings.Split(tag, ",")

	if len(tags) > 1 {
		extra = tags[1]
	}

	if len(tags) > 0 {
		name = tags[0]
	}

	if name == "" {
		name = util.SnakeCasedName(ff.Name)
	}

	return
}

type kv struct {
	k string
	v interface{}
}

func GenUpdateSql(table string, sample interface{}) (string, []interface{}, error) {
	set := []kv{}
	where := []kv{}

	rv := reflect.Indirect(reflect.ValueOf(sample))
	rt := rv.Type()

	if err := genUpdateSql(rv, rt, &set, &where); err != nil {
		return "", nil, err
	}

	if len(set) == 0 {
		return "", nil, status.Errorf(codes.InvalidArgument, "update %s `set` is empty", table)
	}
	if len(where) == 0 {
		return "", nil, status.Errorf(codes.InvalidArgument, "update %s `where` is empty", table)
	}

	buf := &bytes.Buffer{}
	buf.WriteString("update " + table + " set ")

	args := []interface{}{}
	for i, v := range set {
		if i != 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(v.k + "=?")
		args = append(args, v.v)
	}

	buf.WriteString(" where ")
	for i, v := range where {
		if i != 0 {
			buf.WriteString(" and ")
		}
		buf.WriteString(v.k + "=?")
		args = append(args, v.v)
	}

	return buf.String(), args, nil
}

func genUpdateSql(rv reflect.Value, rt reflect.Type, set, where *[]kv) error {

	for i := 0; i < rt.NumField(); i++ {
		fv := rv.Field(i)
		ff := rt.Field(i)
		ft := fv.Type()

		name, extra, skip, inline := getTags(ff)
		if skip || !fv.CanInterface() {
			continue
		}

		if isNil(fv) {
			continue
		}

		if fv.Kind() == reflect.Ptr {
			fv = fv.Elem()
			ft = fv.Type()
		}

		if inline {
			if err := genUpdateSql(fv, ft, set, where); err != nil {
				return err
			}
			continue
		}

		if extra == "where" {
			*where = append(*where, kv{name, fv.Interface()})
		} else {

			v, err := sqlInterface(fv)
			if err != nil {
				return err
			}
			*set = append(*set, kv{name, v})
		}
	}

	return nil
}

func GenInsertSql(table string, sample interface{}) (string, []interface{}, error) {
	values := []kv{}

	rv := reflect.Indirect(reflect.ValueOf(sample))
	rt := rv.Type()

	if err := genInsertSql(rv, rt, &values); err != nil {
		return "", nil, err
	}

	if len(values) == 0 {
		return "", nil, status.Errorf(codes.InvalidArgument, "insert into %s `values` is empty", table)
	}

	buf := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}
	args := []interface{}{}

	buf.WriteString("insert into " + table + " (")

	for i, v := range values {
		if i != 0 {
			buf.WriteString(", ")
			buf2.WriteString(", ")
		}
		buf.WriteString("`" + v.k + "`")
		buf2.WriteString("?")
		args = append(args, v.v)
	}

	return buf.String() + ") values (" + buf2.String() + ")", args, nil
}

func genInsertSql(rv reflect.Value, rt reflect.Type, values *[]kv) error {
	for i := 0; i < rt.NumField(); i++ {
		fv := rv.Field(i)
		ff := rt.Field(i)
		ft := fv.Type()

		name, _, skip, inline := getTags(ff)
		if skip || !fv.CanInterface() {
			continue
		}

		if isNil(fv) {
			continue
		}

		if fv.Kind() == reflect.Ptr {
			fv = fv.Elem()
			ft = fv.Type()
		}

		if inline {
			if err := genInsertSql(fv, ft, values); err != nil {
				return err
			}
			continue
		}

		v, err := sqlInterface(fv)
		if err != nil {
			return err
		}

		*values = append(*values, kv{name, v})
	}

	return nil
}

// bindDest bind struct{} or *struct{} fields to dest
func bindDest(rv reflect.Value, dest []interface{},
	fieldMap map[string]int, tran *[]*transfer) (err error) {
	rt := rv.Type()

	// for **struct{}
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			rv.Set(reflect.New(rt.Elem()))
		}

		rv = rv.Elem()
		rt = rv.Type()
	}

	if rv.Kind() != reflect.Struct || rt.String() == "time.Time" {
		return status.Errorf(codes.InvalidArgument, "orm: interface must be a pointer to struct, got %s", rv.Kind())
	}

	for i := 0; i < rt.NumField(); i++ {
		fv := rv.Field(i)
		ff := rt.Field(i)

		if !fv.CanSet() {
			dlog("can't addr name %s, continue", ff.Name)
			continue
		}

		name, _, skip, inline := getTags(ff)
		//klog.V(5).Infof("%s name %s skip %v inline %v", ff.Name, name, skip, inline)
		if skip {
			continue
		}

		if inline {
			if err = bindDest(fv, dest, fieldMap, tran); err != nil {
				return err
			}
			continue
		}

		if i, ok := fieldMap[name]; ok {
			// rows.Scan() can malloc when ptr == nil
			// do not need to malloc ptr mem at here
			if dest[i], err = scanInterface(fv, tran); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *Rows) genBinder() (*binder, error) {
	if p.rows == nil {
		return nil, status.Errorf(codes.Internal, "rows is nil")
	}

	fields, err := p.rows.Columns()
	if err != nil {
		return nil, err
	}

	fieldMap := map[string]int{}
	for i, name := range fields {
		fieldMap[name] = i
	}

	var empty interface{}
	dest := make([]interface{}, len(fields))
	for i := 0; i < len(dest); i++ {
		dest[i] = &empty
	}

	// klog.V(5).Infof("dest len %d", len(dest))

	return &binder{
		dest:     dest,
		fieldMap: fieldMap,
	}, nil

}

type binder struct {
	dest     []interface{}
	fieldMap map[string]int
}

type transfer struct {
	dstProxy interface{} // byte
	dst      interface{} // raw
	ptr      bool
}

// json -> dst
func (p *transfer) unmarshal() error {
	if p.dstProxy == nil {
		return nil
	}

	jsonStr, ok := p.dstProxy.([]byte)
	if !ok {
		return nil
	}

	rv := reflect.Indirect(reflect.ValueOf(p.dst))
	if p.ptr {
		if rv.IsNil() {
			rv.Set(reflect.New(rv.Type().Elem()))
		}
		rv = rv.Elem()
	}

	if err := json.Unmarshal(jsonStr, rv.Addr().Interface()); err != nil {
		dlog("json.Unmarshal() error %s", err)
	}

	return nil
}

func (p *binder) bind(rv reflect.Value) ([]*transfer, error) {
	tran := []*transfer{}
	if err := bindDest(rv, p.dest, p.fieldMap, &tran); err != nil {
		return nil, err
	}
	return tran, nil
}

// sqlInterface: rv should not be ptr, return interface for use in sql's args
func sqlInterface(rv reflect.Value) (interface{}, error) {
	if rv.Kind() == reflect.Struct || rv.Kind() == reflect.Map ||
		(rv.Kind() == reflect.Slice && rv.Type().Elem().Kind() != reflect.Uint8) {
		if b, err := json.Marshal(rv.Interface()); err != nil {
			return nil, err
		} else {
			return b, nil
		}
	}

	// if rv.Kind() == reflect.Ptr {
	// 	panic(fmt.Sprintf("rv %v rt %v", rv.Kind(), rv.Type().Name()))
	// }

	return rv.Interface(), nil
}

// scanInterface input is struct's field
func scanInterface(rv reflect.Value, tran *[]*transfer) (interface{}, error) {
	rt := rv.Type()
	ptr := false

	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
		ptr = true
	}

	if rt.Kind() == reflect.Struct || rt.Kind() == reflect.Map ||
		(rt.Kind() == reflect.Slice && rt.Elem().Kind() != reflect.Uint8) {
		//if rt.Kind() == reflect.Slice || rt.Kind() == reflect.Map || rt.Kind() == reflect.Struct {
		dst := rv.Addr().Interface()
		// json decode support *struct{}, but not **struct{}, so should adapt it
		node := &transfer{dst: dst, ptr: ptr}
		*tran = append(*tran, node)
		return &node.dstProxy, nil
	}

	return rv.Addr().Interface(), nil
}

func isNil(rv reflect.Value) bool {
	switch rv.Kind() {
	case reflect.Map, reflect.Ptr, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}
