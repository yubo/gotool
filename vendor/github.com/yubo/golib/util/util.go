package util

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"os/user"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/yubo/golib/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
)

const (
	alphaDelta = 'a' - 'A'
)

func IndentLines(i int, lines string) (ret string) {
	ls := strings.Split(strings.Trim(lines, "\n"), "\n")
	indent := strings.Repeat(" ", i*IndentSize)
	for _, l := range ls {
		ret += fmt.Sprintf("%s%s\n", indent, l)
	}
	return string([]byte(ret)[:len(ret)-1])
}

func AddrIsDisable(addr string) bool {
	if addr == "" || addr == "disable" || addr == "<no value>" {
		return true
	}
	return false
}

func Dialer(addr string, timeout time.Duration) (net.Conn, error) {
	d := net.Dialer{Timeout: timeout}
	return d.Dial(ParseAddr(addr))
}

func sortTags(s []byte) []byte {
	str := strings.Replace(string(s), " ", "", -1)
	if str == "" {
		return []byte{}
	}

	tags := strings.Split(str, ",")
	sort.Strings(tags)
	return []byte(strings.Join(tags, ","))
}

func StructConv(in, out interface{}) interface{} {
	StructCopy(out, in)
	return out
}

func StructCopy(dst, src interface{}) error {
	srcV := reflect.Indirect(reflect.ValueOf(src))
	srcT := srcV.Type()

	dstV := reflect.Indirect(reflect.ValueOf(dst))

	if !dstV.CanSet() {
		return errors.New("target can't set")
	}

	for i := 0; i < srcV.NumField(); i++ {
		fname := srcT.Field(i).Name

		srcF := srcV.Field(i)
		dstF := dstV.FieldByName(fname)
		if !dstF.IsValid() || !dstF.CanSet() {
			//fmt.Printf("fname %s\n", fname)
			continue
		}

		if srcF.Type().Kind() != dstF.Type().Kind() {
			continue
		}

		switch dstF.Type().Kind() {
		case reflect.Struct, reflect.Map:
			// skip
		default:
			dstF.Set(srcF)
		}
	}
	return nil
}

func KeyAttr(key []byte) (string, string, string, string, error) {
	var err error
	s := strings.Split(string(key), "/")
	if len(s) != 4 {
		err = EINVAL
	}

	return s[0], s[1], s[2], s[3], err
}

func AttrKey(endpoint, metric, tags, typ string) []byte {
	return []byte(fmt.Sprintf("%s/%s/%s/%s", endpoint, metric, tags, typ))
}

func GetType(v interface{}) string {
	t := reflect.TypeOf(v)
	switch t.Kind() {
	case reflect.Ptr:
		//return "*" + t.Elem().Name()
		return t.Elem().Name()
	case reflect.Map:
		return fmt.Sprintf("%v", reflect.MapOf(t.Key(), t.Elem()))
	default:
		return t.Name()
	}
}

func GetName(v interface{}) string {
	return strings.ToLower(GetType(v))
}

// Environment Variables $HOME value instead of ${HOME}
func EnvVarFilter(data []byte) []byte {
	// flag : 0 - out,  1 - find '$', 2 - in value
	ret := make([]byte, 0, len(data))
	p := 0
	flag := 0
	for i := 0; i < len(data); i++ {
		c := data[i]
		switch flag {
		case 0:
			if c == '$' {
				flag = 1
				p = i
			} else {
				ret = append(ret, c)
			}
		case 1:
			if c == '{' {
				flag = 2
			} else {
				flag = 0
				ret = append(ret, data[p:i+1]...)
			}
		case 2:
			if c == '}' {
				// end
				ret = append(ret, []byte(os.Getenv(string(data[p+2:i])))...)
				flag = 0
			} else if !((c >= '0' && c <= '9') ||
				(c >= 'a' && c <= 'z') ||
				(c >= 'A' && c <= 'Z') ||
				c == '-' || c == '_') {
				// error
				flag = 0
				ret = append(ret, data[p:i+1]...)
			}
		}
	}
	if flag > 0 {
		ret = append(ret, data[p:]...)
	}
	return ret
}

// str, def, max
func Atoi(str string, def ...int) int {
	return int(Atoi64(str, def...))
}

func Atoi64(str string, def ...int) int64 {
	i64, err := strconv.ParseInt(str, 10, 0)
	if err != nil {
		if len(def) > 0 {
			return int64(def[0])
		}
		return 0
	}

	if len(def) > 1 && (i64 == 0 || i64 > int64(def[1])) {
		i64 = int64(def[1])
	}
	return i64
}

func Atob(str string, def ...bool) bool {
	b, err := strconv.ParseBool(str)
	if err != nil {
		if len(def) > 0 {
			return def[0]
		}
		return false
	}
	return b
}

func IntRange(a, b, c int) int {
	if a <= 0 {
		return b
	}

	if a > c {
		return c
	}

	return a
}

func JsonStr(in interface{}, pretty ...bool) string {
	var (
		b   []byte
		err error
	)
	if len(pretty) > 0 && pretty[0] {
		b, err = json.MarshalIndent(in, "", "  ")

	} else {
		b, err = json.Marshal(in)
	}
	if err != nil {
		return err.Error()
	} else {
		return string(b)
	}
}

func EnvDef(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func Caller(n int) string {
	pc, file, line, ok := runtime.Caller(n)
	if ok {
		f := runtime.FuncForPC(pc)
		return fmt.Sprintf("%s:%d %s", file, line, f.Name())
	}
	return ""
}

func Backtraces() {
	pcs := make([]uintptr, 20)
	pcs = pcs[:runtime.Callers(0, pcs)]
	frames := runtime.CallersFrames(pcs)
	i := 0
	for {
		frame, more := frames.Next()
		fmt.Printf("%d %s:%d\n", i, frame.Function, frame.Line)
		i++
		if !more {
			break
		}
	}
}

// ToScope transport strings fields into map[string]bool format
func ToScope(scope *string) (ret map[string]bool) {
	ret = map[string]bool{}

	if scope == nil {
		return
	}

	for _, v := range strings.Fields(*scope) {
		ret[v] = true
	}
	return
}

func ToScopeStr(scope map[string]bool) *string {
	if len(scope) == 0 {
		return nil
	}

	var ret string
	for k, v := range scope {
		if v {
			ret += k + " "
		}
	}
	ret = ret[:len(ret)-1]
	return &ret
}

func Strings2MapBool(ss []string) map[string]bool {
	ret := map[string]bool{}
	for _, v := range ss {
		ret[v] = true
	}
	return ret
}

// ss > in[n] > in[n-1] > ... > in[0]
func Strings2MapString(ss []string, in ...map[string]string) map[string]string {
	ret := map[string]string{}
	for _, v := range in {
		for k1, v1 := range v {
			ret[k1] = v1
		}
	}
	for _, s := range ss {
		if i := strings.IndexByte(s, '='); i > 0 {
			ret[s[:i]] = s[i+1:]
		}
	}
	return ret
}

func MergeMapString(in ...map[string]string) map[string]string {
	ret := map[string]string{}
	for _, v := range in {
		for k1, v1 := range v {
			ret[k1] = v1
		}
	}
	return ret
}

// srcsAddrs tries to UDP-connect to each address to see if it has a
// route. (This doesn't send any packets). The destination port
// number is irrelevant.
func SrcAddrs(addrs []net.IPAddr) []net.IP {
	srcs := make([]net.IP, len(addrs))
	dst := net.UDPAddr{Port: 9}
	for i := range addrs {
		dst.IP = addrs[i].IP
		dst.Zone = addrs[i].Zone
		c, err := net.DialUDP("udp", nil, &dst)
		if err == nil {
			if src, ok := c.LocalAddr().(*net.UDPAddr); ok {
				srcs[i] = src.IP
			}
			c.Close()
		}
	}
	return srcs
}

func SrcAddr(addr net.IPAddr) net.IP {
	dst := net.UDPAddr{Port: 9, IP: addr.IP, Zone: addr.Zone}
	conn, err := net.DialUDP("udp", nil, &dst)
	if err != nil {
		return net.IP{}
	}
	defer conn.Close()

	src, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return src.IP
	}
	conn.Close()

	return net.IP{}
}

func SrcAddrV4(a, b, c, d byte) net.IP {
	dst := net.UDPAddr{
		IP:   net.IPv4(a, b, c, d),
		Port: 9,
	}
	conn, err := net.DialUDP("udp", nil, &dst)
	if err != nil {
		return net.IP{}
	}

	defer conn.Close()

	if src, ok := conn.LocalAddr().(*net.UDPAddr); ok {
		return src.IP
	}

	return net.IP{}
}

func Base64Decode(in string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(in)
}

func Base64Encode(in []byte) string {
	return base64.StdEncoding.EncodeToString(in)
}

func DirNameMap(dirName string) (map[string]string, error) {
	var (
		keyZone bool
		i, j    int
		k, v    string
		ret     = make(map[string]string)
	)

	for keyZone, i, j = true, 0, 0; j < len(dirName); j++ {
		if dirName[j] == '=' && keyZone {
			k = strings.TrimSpace(dirName[i:j])
			i = j + 1
			keyZone = false
		} else if dirName[j] == ',' && !keyZone {
			v = strings.TrimSpace(dirName[i:j])
			if len(k) > 0 && len(v) > 0 {
				ret[k] = v
				k, v = "", ""
			} else {
				return ret, ErrParam
			}
			i = j + 1
			keyZone = true
		}
	}

	v = strings.TrimSpace(dirName[i:])
	if len(k) > 0 && len(v) > 0 {
		ret[k] = v
		return ret, nil
	} else {
		return ret, ErrParam
	}
}

func FirstLine(in string) string {
	in = strings.TrimSpace(in)
	if n := strings.IndexByte(in, '\n'); n > 0 {
		return in[:n]
	}
	return in
}

func idxSubStr(n, max, def int) int {
	if n == 0 {
		return def
	}
	if n < 0 {
		if n = max - n; n < 0 {
			n = 0
		}
	}
	if n > max {
		n = max
	}
	return n
}

// SubStr is Safe SubStr
func SubStr(in string, begin, end int) string {
	l := len(in)

	begin = idxSubStr(begin, l, 0)
	end = idxSubStr(end, l, l)

	return in[begin:end]
}

// SubStr2 return [begin, end)...
func SubStr2(in string, begin, end int) string {
	out := SubStr(in, begin, end)
	if len(in) > len(out) && len(out) > 3 {
		out = out[:len(out)-3] + "..."
	}
	return out
}

// SubStr3 return [0:pre]...[suf:len]
func SubStr3(in string, pre, suf int) string {
	L := len(in)

	if pre+suf >= L {
		return in
	}
	if pre < 0 {
		pre += L
	}
	if pre < 0 {
		pre = 0
	}
	if pre > L {
		pre = L
	}
	if suf < 0 {
		suf += L
	}
	if suf < 0 {
		suf = 0
	}
	if suf > L {
		suf = L
	}

	if pre >= suf {
		return in
	}

	return in[:pre] + "..." + in[suf:]
}

func LastLine(in string) string {
	in = strings.TrimSpace(in)
	if n := strings.LastIndexByte(in, '\n'); n > 0 {
		return in[n+1:]
	}
	return in
}
func IsEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}

func Errorf(format string, args ...interface{}) error {
	_, file, line, _ := runtime.Caller(1)
	return fmt.Errorf(fmt.Sprintf("%s:%d %s", file, line, format), args...)
}

func Error(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func ErrorString(err error) *string {
	if err == nil {
		return nil
	}
	return String(err.Error())
}

func Diff(src, dst []string) (add, del []string) {
	s := map[string]bool{}
	d := map[string]bool{}

	for _, v := range src {
		s[v] = true
	}

	for _, v := range dst {
		d[v] = true
	}

	for _, v := range dst {
		if !s[v] {
			add = append(add, v)
		}
	}

	for _, v := range src {
		if !d[v] {
			del = append(del, v)
		}
	}

	return
}

func Diff3(src, dst []string) (add, del, eq []string) {
	s := map[string]bool{}
	d := map[string]bool{}

	for _, v := range src {
		s[v] = true
	}

	for _, v := range dst {
		d[v] = true
		if !s[v] {
			add = append(add, v)
		} else {
			eq = append(eq, v)
		}
	}

	for _, v := range src {
		if !d[v] {
			del = append(del, v)
		}
	}

	return
}

// convert like this: "HelloWorld" to "hello_world"
func SnakeCasedName(name string) string {
	newstr := make([]rune, 0, len(name)+8)
	lastAppend := false

	for i, chr := range string(name) {
		if 'A' <= chr && chr <= 'Z' {
			if i > 0 && !lastAppend {
				newstr = append(newstr, '_')
				lastAppend = true
			}
			chr -= ('A' - 'a')
		} else {
			lastAppend = false
		}
		newstr = append(newstr, chr)
	}

	return string(newstr)
}

func LowerCamelCasedName(in string) string {
	return TitleCasedName(in, true)
}

func CamelCasedName(in string) string {
	return TitleCasedName(in, false)
}

// convert like this: "hello_world" to "HelloWorld"
func TitleCasedName(name string, lower bool) string {
	newstr := make([]rune, 0, len(name))
	upNextChar := true

	for _, chr := range name {
		if chr == '-' || chr == '_' || chr == '.' {
			upNextChar = true
			continue
		}

		if chr >= 'a' && chr <= 'z' && upNextChar {
			chr -= alphaDelta
		}

		newstr = append(newstr, chr)
		upNextChar = false
	}

	if lower && newstr[0] >= 'A' && newstr[0] <= 'Z' {
		newstr[0] += alphaDelta
	}

	return string(newstr)
}

// Maybe s is of the form t c u.
// If so, return t, c u (or t, u if cutc == true).
// If not, return s, "".
func Split(s string, c string, cutc bool) (string, string) {
	i := strings.Index(s, c)
	if i < 0 {
		return s, ""
	}
	if cutc {
		return s[:i], s[i+len(c):]
	}
	return s[:i], s[i:]
}

func Username() string {
	var username string
	if user, err := user.Current(); err == nil {
		username = user.Username
	} else {
		// user.Current() currently requires cgo. If an error is
		// returned attempt to get the username from the environment.
		username = os.Getenv("USER")
	}
	if username == "" {
		panic("Unable to get username")
	}
	return username
}

func escapeShell(in string) string {
	return `'` + strings.Replace(in, `'`, `'\''`, -1) + `'`
}

func StringArrayContains(needle string, haystack []string) bool {
	for _, v := range haystack {
		if needle == v {
			return true
		}
	}
	return false
}

// ab=cd => ab=***
func KvMask(s string) string {
	s2 := []byte(s)
	n := bytes.IndexByte(s2, '=')
	if n <= 0 {
		return ""
	}

	return s[:n] + "=***"
}

func NewStructPtr(in interface{}) interface{} {
	rt := reflect.TypeOf(in)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	if rt.Kind() != reflect.Struct {
		panic("needs a pointer to a struct")
	}

	return reflect.New(rt).Interface()
}

// MakeSlice return a slice of input
func MakeSlice(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	typ := reflect.TypeOf(v)
	sli := reflect.SliceOf(typ)
	slice := reflect.MakeSlice(sli, 0, 0)
	x := reflect.New(slice.Type())
	x.Elem().Set(slice)
	return x.Elem().Interface()
}

// GetArticleForNoun returns the article needed for the given noun.
func GetArticleForNoun(noun string, padding string) string {
	if noun[len(noun)-2:] != "ss" && noun[len(noun)-1:] == "s" {
		// Plurals don't have an article.
		// Don't catch words like class
		return fmt.Sprintf("%v", padding)
	}

	article := "a"
	if isVowel(rune(noun[0])) {
		article = "an"
	}

	return fmt.Sprintf("%s%s%s", padding, article, padding)
}

// isVowel returns true if the rune is a vowel (case insensitive).
func isVowel(c rune) bool {
	vowels := []rune{'a', 'e', 'i', 'o', 'u'}
	for _, value := range vowels {
		if value == unicode.ToLower(c) {
			return true
		}
	}
	return false
}

// toInt64 converts integer types to 64-bit integers
func toInt64(v interface{}) int64 {
	if str, ok := v.(string); ok {
		iv, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return 0
		}
		return iv
	}

	val := reflect.Indirect(reflect.ValueOf(v))
	switch val.Kind() {
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		return val.Int()
	case reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return int64(val.Uint())
	case reflect.Uint, reflect.Uint64:
		tv := val.Uint()
		if tv <= math.MaxInt64 {
			return int64(tv)
		}
		return math.MaxInt64
	case reflect.Float32, reflect.Float64:
		return int64(val.Float())
	case reflect.Bool:
		if val.Bool() == true {
			return 1
		}
		return 0
	default:
		return 0
	}
}

func SetValue(rv reflect.Value, rt reflect.Type, data []string) error {
	if len(data) == 0 {
		return nil
	}

	// Dereference ptr
	PrepareValue(rv, rt)
	if rv.Kind() == reflect.Ptr {
		// fmt.Printf("rt %s is ptr, derefreence\n", rt.String())
		rv = rv.Elem()
		rt = rv.Type()
	}

	switch rv.Kind() {
	case reflect.String:
		rv.SetString(data[0])

	case reflect.Bool:
		rv.SetBool(data[0] == "true")

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if v, err := strconv.ParseInt(data[0], 10, 64); err != nil {
			return status.Errorf(codes.Internal, "arg %s as int: %s", rt.Name(), err.Error())
		} else {
			rv.SetInt(v)
		}
	case reflect.Float32, reflect.Float64:
		if v, err := strconv.ParseFloat(data[0], 64); err != nil {
			return status.Errorf(codes.Internal, "arg %s as float: %s", rt.Name(), err.Error())
		} else {
			rv.SetFloat(v)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if v, err := strconv.ParseUint(data[0], 10, 64); err != nil {
			return status.Errorf(codes.Internal, "arg %s as uint: %s", rt.Name(), err.Error())
		} else {
			rv.SetUint(v)
		}
	case reflect.Slice:
		typeName := rt.Elem().String()
		if typeName == "string" {
			rv.Set(reflect.ValueOf(data))
		} else if typeName == "*string" {
			rv.Set(reflect.ValueOf(StringSlice(data)))
		} else {
			return status.Errorf(codes.Internal, "unsupported type scan %s slice", typeName)
		}
	default:
		return status.Errorf(codes.Internal, "unsupported type scan %s", rt.String())
	}
	return nil
}

func GetValue(rv reflect.Value, rt reflect.Type) (data []string, err error) {
	rv = reflect.Indirect(rv)
	switch rv.Kind() {
	case reflect.String, reflect.Bool, reflect.Int, reflect.Int8,
		reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:

		return []string{fmt.Sprintf("%v", rv.Interface())}, nil
	case reflect.Slice:
		typeName := rt.Elem().String()
		if typeName == "string" {
			return rv.Interface().([]string), nil
		} else if typeName == "*string" {
			return StringValueSlice(rv.Interface().([]*string)), nil
		}
		return nil, status.Errorf(codes.Internal, "unsupported type: %s %s", rt.String(), rv.Kind().String())
	default:
		return nil, status.Errorf(codes.Internal, "unsupported type: %s %s", rt.String(), rv.Kind().String())
	}
}

func PrepareValue(rv reflect.Value, rt reflect.Type) {
	if rv.Kind() == reflect.Ptr && rv.IsNil() {
		rv.Set(reflect.New(rt.Elem()))
	}
}

// GetPeerAddrFromCtx try to return peer address from grpc context
func GetPeerAddrFromCtx(ctx context.Context) (string, error) {
	pr, ok := peer.FromContext(ctx)
	if !ok {
		return "", status.Errorf(codes.Internal, "[getClinetIP] invoke FromContext() failed")
	}
	if pr.Addr == net.Addr(nil) {
		return "", status.Errorf(codes.Internal, "[getClientIP] peer.Addr is nil")
	}
	ip, _, err := net.SplitHostPort(pr.Addr.String())
	return ip, err
}
