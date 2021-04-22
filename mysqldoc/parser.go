package main

import (
	"errors"
	"io/ioutil"
	"regexp"
	"sort"
	"strings"

	"github.com/yubo/golib/orm"
)

type FieldInfo struct {
	Name string // 字段名
	Desc string // 字段描述
}

type KeyInfo struct {
	Name   string // 键名
	Type   string // 键类型
	Fields string // 键的字段列表
}

type EngineInfo struct {
	Name string // 引擎名
	Desc string // 引擎描述
}

type MysqlTable struct {
	Name       string      // 表名
	SqlStr     string      // sql语句
	Fields     []FieldInfo // 字段列表
	Keys       []KeyInfo   // 键列表
	Engine     EngineInfo  // 引擎
	IsChild    bool        // 是否是子表
	ChildNames []string    // 子表名列表
	LikeTbl    string      // like的表名
}

var (
	// regexps
	lineRe  = regexp.MustCompile(`.*?\n`)
	likeRe  = regexp.MustCompile(`like\s+?` + "`" + `(\S+)` + "`")
	tnmRe   = regexp.MustCompile(`CREATE\s+TABLE\s+` + "`" + `(\S+?)` + "`" + "(.+)")
	fldRe   = regexp.MustCompile(`^\s*` + "`" + `(\S+)` + "`" + `\s*(.+),`)
	keyRe   = regexp.MustCompile(`^\s*(.*?KEY)\s*(\S*)\s*\((.+?)\)`)
	knmRe   = regexp.MustCompile("`" + `(\S+)` + "`")
	ngnRe   = regexp.MustCompile(`^\s*\)\s*?ENGINE=(\S+)\s*(.*);`)
	childRe = regexp.MustCompile(`UNION=\((\S+)\)`)
	tnameRe = regexp.MustCompile("`" + `(\S+)` + "`")
)

func parseTablesFromFile(file string) ([]*MysqlTable, error) {
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`(?s)CREATE\s+?TABLE.+?;`)
	tableNames := re.FindAllString(string(bytes), -1)
	if len(tableNames) == 0 {
		return nil, err
	}

	tables := make([]*MysqlTable, 0, len(tableNames))
	for _, tbstr := range tableNames {
		t, err := parseTableSql(tbstr)
		if err != nil {
			return nil, err
		}
		tables = append(tables, t)
	}
	parseTableEx(tables)
	return tables, nil
}

func getTableCreateSql(db *orm.Db, table string) (sql string, err error) {
	var name string
	err = db.Query("show create table "+table).Row(&name, &sql)
	return
}

func parseTables(db *orm.Db) ([]*MysqlTable, error) {
	var tabNames []string

	if err := db.Query("show tables").Rows(&tabNames); err != nil {
		return nil, err
	}

	tables := make([]*MysqlTable, 0, len(tabNames))
	// show create table xxx;
	for _, v := range tabNames {
		sql, err := getTableCreateSql(db, v)
		if err != nil {
			return nil, err
		}
		t, err := parseTableSql(sql + ";")
		if err != nil {
			return nil, err
		}
		tables = append(tables, t)
	}
	parseTableEx(tables)
	return tables, nil
}

func parseTable(db *orm.Db, tabName string) (*MysqlTable, error) {
	sql, err := getTableCreateSql(db, tabName)
	if err != nil {
		return nil, err
	}
	return parseTableSql(sql + ";")
}

func parseTableSql(tabSql string) (*MysqlTable, error) {
	tabSql += "\n"

	lines := lineRe.FindAllString(tabSql, -1)
	t := MysqlTable{
		SqlStr:  tabSql,
		Fields:  make([]FieldInfo, 0, len(lines)),
		Keys:    make([]KeyInfo, 0, len(lines)),
		Engine:  EngineInfo{},
		IsChild: false,
	}

	step := ""
	for idx, line := range lines {
		line = strings.ReplaceAll(line, "\r", "") //兼容windows("\r\n")
		if idx == 0 {
			tblEx := ""
			ret := tnmRe.FindStringSubmatch(line)
			if len(ret) < 2 {
				panic("解析表名错误, line:" + line)
			} else if len(ret) > 2 {
				tblEx = ret[2]
			}
			t.Name = ret[1]

			// 支持：create table `xxx` like `yyy`;
			if len(tblEx) > 0 {
				// 复制表结构
				ret := likeRe.FindStringSubmatch(tblEx)
				if len(ret) == 2 {
					t.LikeTbl = ret[1]
					step = "t_end"
					continue
				}
			}
			step = "tname_end" // 表名解析完成
		} else {
			// 解析字段
			if step == "tname_end" {
				ret := fldRe.FindStringSubmatch(line)
				if len(ret) == 3 {
					fieldName, fieldDesc := ret[1], ret[2]
					t.Fields = append(t.Fields, FieldInfo{fieldName, fieldDesc})
				} else {
					step = "tflds_end" // 字段解析完成
				}
			}

			// 解析键（包括主键和其他键）
			if step == "tflds_end" {
				ret := keyRe.FindStringSubmatch(line) // RRIMARY KEY (`id`) 或 KEY `key_idx` (`xx`, `yy`)
				if len(ret) == 4 {
					var keyType, keyName, keyFlds string
					keyType = ret[1]
					keyFlds = ret[3]
					if keyType == "PRIMARY KEY" {
						// primary key
						keyName = ""
					} else {
						// other key
						knmRet := knmRe.FindStringSubmatch(ret[2])
						if len(knmRet) == 2 {
							keyName = knmRet[1]
						}
					}

					// bugfix:修复多个键名之间有空格时，每次都要重新更新数据库的问题
					keyFlds = strings.ReplaceAll(keyFlds, " ", "")
					t.Keys = append(t.Keys, KeyInfo{keyName, keyType, keyFlds})
				} else {
					// sort key(按键名升序)
					sort.Slice(t.Keys, func(i, j int) bool {
						return t.Keys[i].Name < t.Keys[j].Name
					})

					step = "tkeys_end"
				}
			}

			// 解析engine
			if step == "tkeys_end" {
				ret := ngnRe.FindStringSubmatch(line)
				if len(ret) == 3 {
					t.Engine.Name = ret[1]
					t.Engine.Desc = ret[2]

					if t.Engine.Name == "MRG_MyISAM" {
						// myisam 分表
						ret := childRe.FindStringSubmatch(t.Engine.Desc)
						if len(ret) == 2 {
							child := strings.Split(ret[1], ",")
							if len(child) > 0 {
								t.ChildNames = make([]string, 0, len(child))
								for _, v := range child {
									nmRet := tnameRe.FindStringSubmatch(v)
									if len(nmRet) == 2 {
										t.ChildNames = append(t.ChildNames, nmRet[1])
									}
								}
							}
						}
					}
					step = "t_end"
					break
				}
			}
		}
	}

	// append to table list
	if step != "t_end" {
		return nil, errors.New("解析table错误, sql:\n" + tabSql)
	}
	return &t, nil
}

func parseTableEx(tbls []*MysqlTable) {
	childList := make([]string, 0)
	likeMap := make(map[string]string)

	tblMap := make(map[string]*MysqlTable, len(tbls))
	for _, tbl := range tbls {
		tblMap[tbl.Name] = tbl

		// 分表处理
		if tbl.Engine.Name == "MRG_MyISAM" {
			for _, cnm := range tbl.ChildNames {
				childList = append(childList, cnm)
			}
		}

		// like处理
		// eg.：create table `xxx` like `yyy`;
		if len(tbl.LikeTbl) > 0 {
			likeMap[tbl.Name] = tbl.LikeTbl
		}
	}

	for _, tnm := range childList {
		if t, ok := tblMap[tnm]; ok {
			t.IsChild = true
		}
	}

	for tnm, lktnm := range likeMap {
		if lkt, ok := tblMap[lktnm]; ok {
			t := tblMap[tnm]
			t.Fields = append(t.Fields, lkt.Fields...)
			t.Keys = append(t.Keys, lkt.Keys...)
			t.Engine = lkt.Engine
		}
	}
}
