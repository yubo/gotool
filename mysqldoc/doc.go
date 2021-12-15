package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/yubo/golib/orm"
)

type Doc struct {
	*Config
	db   orm.DB
	sqls []string
}

func (p *Doc) conn() error {
	var err error
	if p.db, err = orm.Open("mysql", p.Config.dsn); err != nil {
		return err
	}
	return nil
}

func (p *Doc) close() error {
	p.db.Close()
	return nil
}

func (p *Doc) loadDict() map[string]string {
	dict := map[string]string{}
	if p.dict != "" {
		bytesRead, err := ioutil.ReadFile(p.dict)
		if err != nil {
			fmt.Printf("error opening file: %v\n", err)
			os.Exit(1)
		}
		for _, line := range strings.Split(string(bytesRead), "\n") {
			if fs := strings.Split(strings.TrimSpace(line), ":"); len(fs) == 2 {
				dict[fs[0]] = fs[1]
			}
		}
	}
	return dict
}

func (p *Doc) dbDoc() error {
	var tabs []string
	if err := p.db.Query("show tables").Rows(&tabs); err != nil {
		return err
	}

	dict := p.loadDict()

	for _, tab := range tabs {
		p.tableDoc(tab, dict)
	}

	fmt.Printf("\n\n### miss dict\n")
	for desc := range dict {
		if dict[desc] == "" {
			fmt.Printf("%s\n", desc)
		}
	}

	return nil
}

func (p *Doc) tableDoc(tab string, dict map[string]string) error {
	d, err := parseTable(p.db, tab)
	if err != nil {
		return err
	}
	if d.IsChild {
		return nil
	}

	printTableDoc(d, dict)
	return nil
}

var (
	commentRe  = regexp.MustCompile(`COMMENT='(\S+)'\s*`)
	notnullRe  = regexp.MustCompile(`(NOT NULL)`)
	comment2Re = regexp.MustCompile(`COMMENT '(\S+)'\s*`)
	defaultRe  = regexp.MustCompile(`DEFAULT\s+(\S+)\s*`)
	typeRe     = regexp.MustCompile(`^(\S+(\s+unsigned)?)\s*`)
)

type fieldDoc struct {
	name    string
	typ     string
	def     string
	comment string
}

type tableDoc struct {
	comment string
	fields  map[string]string
}

func printTableDoc(t *MysqlTable, dict map[string]string) {
	var comment string
	if m := commentRe.FindStringSubmatch(t.Engine.Desc); len(m) == 2 {
		comment = m[1]
	}

	fmt.Printf("\n\n#### 表名 %s\n", t.Name)
	if len(comment) > 0 {
		fmt.Printf("%s\n\n", comment)
	}

	//fmt.Printf("序号 | 字段名 | 类型 | 允许空 | 缺省值 | 备注\n")
	fmt.Printf("序号 | 名称 | 数据类型 | 允许空值 | 说明\n")
	fmt.Printf("-- | -- | -- | -- | --\n")
	for i, v := range t.Fields {
		var notnull bool
		if m := notnullRe.FindStringSubmatch(v.Desc); len(m) == 2 {
			notnull = true
		}

		//var comment string
		//if m := comment2Re.FindStringSubmatch(v.Desc); len(m) == 2 {
		//	comment = m[1]
		//}

		var typ string
		if m := typeRe.FindStringSubmatch(v.Desc); len(m) >= 2 {
			typ = m[1]
		}

		//def := "-"
		//if m := defaultRe.FindStringSubmatch(v.Desc); len(m) == 2 {
		//	def = m[1]
		//}
		desc := strings.ReplaceAll(v.Name, "_", " ")
		if s, ok := dict[desc]; ok {
			desc = s
		} else {
			dict[desc] = ""
		}

		fmt.Printf("%d | %s | %s | %v | %s\n",
			i+1, v.Name, typ, !notnull, desc)
	}

	//fmt.Printf("\n索引\n\n")

	//fmt.Printf("序号 | 索引名 | 类型 | 字段名\n")
	//fmt.Printf("-- | -- | -- | --\n")
	//for i, v := range t.Keys {
	//	name := "-"
	//	if len(v.Name) > 0 {
	//		name = v.Name
	//	}
	//	fmt.Printf("%d | %s | %s | %s\n",
	//		i, name, v.Type, v.Fields)
	//}
}
