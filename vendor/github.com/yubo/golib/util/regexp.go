package util

import (
	"fmt"
	"regexp"
)

var (
	networkExp = regexp.MustCompile(`^(tcp)|(unix)+:`)
	IpExp      = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+`)
	AddrExp    = regexp.MustCompile(`^([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)?:[0-9]+`)
	NumExp     = regexp.MustCompile(`^0x[0-9a-fA-F]+|^[0-9]+`)
	KeywordExp = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-_]+`)
	TextExp    = regexp.MustCompile(`(^"[^"]+")|(^[^"\n \t;]+)`)
	EnvExp     = regexp.MustCompile(`\$\{[a-zA-Z][0-9a-zA-Z_]+\}`)
	NameExp    = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-_\.]*$`)
	NameExp2   = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	NameExp3   = regexp.MustCompile(`^[a-z][a-z0-9:]*$`)
	PathExp    = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9-_\.]*(\/[a-zA-Z0-9_][a-zA-Z0-9-_\.]*)*$`)
)

func ParseAddr(url string) (net, addr string) {
	if f := networkExp.Find([]byte(url)); f != nil {
		return url[:len(f)-1], url[len(f):]
	}
	return "tcp", url
}

func exprCheck(str string, exp *regexp.Regexp) error {
	if f := exp.Find([]byte(str)); f != nil {
		return nil
	}
	return fmt.Errorf("'%s' is invalid string, expr(%s)", str, exp)
}

func CheckName(name string) error      { return exprCheck(name, NameExp) }
func CheckGroupName(name string) error { return exprCheck(name, NameExp2) }
func CheckDirName(name string) error   { return exprCheck(name, NameExp2) }
func CheckRoleName(name string) error  { return exprCheck(name, NameExp3) }
func CheckScopeName(name string) error { return exprCheck(name, NameExp3) }
func CheckPath(name string) error      { return exprCheck(name, PathExp) }
