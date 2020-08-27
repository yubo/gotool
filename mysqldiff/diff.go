package main

import (
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/yubo/golib/orm"
	"github.com/yubo/golib/status"
	"github.com/yubo/golib/util"
)

type differ struct {
	*Config
	srcDb        *orm.Db
	dstDb        *orm.Db
	addTables    []string
	delTables    []string
	eqTables     []string
	sqls         []string
	repairTables map[string][]string
	err          error
}

func (p *differ) addSql(sql string, args ...interface{}) {
	p.sqls = append(p.sqls, fmt.Sprintf(sql, args...))
}

func mysqldiff(cf *Config) error {
	p := &differ{Config: cf}
	if err := p.conn(); err != nil {
		return err
	}
	defer p.close()

	if err := p.compareDb(); err != nil {
		p.err = err
		return err
	}

	// exec
	if err := p.do(); err != nil {
		p.err = err
		return err
	}

	return nil
}

func (p *differ) do() error {
	for _, v := range p.sqls {
		fmt.Println(v + ";")
		if !p.exec {
			continue
		}

		if err := p.srcDb.ExecErr(v); err != nil {
			p.err = err
			return err
		}
	}
	return nil
}

func (p *differ) conn() error {
	var err error
	if p.srcDb, err = orm.DbOpen("mysql", p.srcDsn); err != nil {
		return err
	}
	if p.dstDb, err = orm.DbOpen("mysql", p.dstDsn); err != nil {
		return err
	}
	return nil
}

func (p *differ) close() error {
	if p.exec {
		if p.err != nil {
			p.srcDb.Rollback()
		} else {
			p.srcDb.Commit()
		}
	}
	p.srcDb.Close()
	p.dstDb.Close()
	return nil
}

func (p *differ) compareDb() error {
	var srcTables []string
	if err := p.srcDb.Query("show tables").Rows(&srcTables); err != nil && !status.NotFound(err) {
		return err
	}

	var dstTables []string
	if err := p.dstDb.Query("show tables").Rows(&dstTables); err != nil && !status.NotFound(err) {
		return err
	}

	p.addTables, p.delTables, p.eqTables = util.Diff3(srcTables, dstTables)

	for _, v := range p.addTables {
		if sql, err := getTableCreateSql(p.dstDb, v); err != nil {
			return err
		} else {
			p.addSql(sql)
		}
	}

	for _, v := range p.delTables {
		p.addSql("drop table %s", v)
	}

	for _, v := range p.eqTables {
		if err := p.compareTable(v); err != nil {
			return err
		}
	}

	return nil
}

func (p *differ) compareTable(tableName string) error {
	d, err := parseTable(p.dstDb, tableName)
	if err != nil {
		return err
	}
	if d.IsChild {
		return nil
	}

	s, err := parseTable(p.srcDb, tableName)
	if err != nil {
		return err
	}

	// update

	// drop , del
	add, drop, err := p.mysqlDiffKey(s, d)
	if err != nil {
		return err
	}

	for _, v := range drop {
		p.addSql(v)
	}
	if err := p.mysqlDiffField(s, d); err != nil {
		return err
	}
	for _, v := range add {
		p.addSql(v)
	}

	return nil
}

func getTableCreateSql(db *orm.Db, table string) (sql string, err error) {
	var name string
	err = db.Query("show create table "+table).Row(&name, &sql)
	return
}

func (p *differ) mysqlDiffField(srcTable, dstTable *MysqlTable) error {

	oFlds := srcTable.Fields
	nFlds := dstTable.Fields
	oMap := make(map[string]string, len(oFlds))
	nMap := make(map[string]string, len(nFlds))
	for _, f := range nFlds {
		nMap[f.Name] = f.Desc
	}

	// 先drop
	ignoreMap := make(map[string]bool)
	for _, f := range oFlds {
		if _, ok := nMap[f.Name]; !ok {
			ignoreMap[f.Name] = true

			// mother
			p.addSql("alter table %s %s `%s`", srcTable.Name, "drop", f.Name)

			// child
			for _, cnm := range srcTable.ChildNames {
				p.addSql("alter table %s %s `%s`", cnm, "drop", f.Name)
			}
		} else {
			oMap[f.Name] = f.Desc
		}
	}

	// 新增的和变化的
	oIdx := 0
	lastFld := ""
	for _, nf := range nFlds {
		// 找一个基准
		var fp *FieldInfo
		for oi, f := range oFlds {
			if oi >= oIdx {
				if ignoreMap[f.Name] {
					oIdx += 1
				} else {
					fp = &f
					break
				}
			}
		}

		var op string
		var last = lastFld
		lastFld = nf.Name
		if fp != nil {
			if fp.Name != nf.Name {
				if _, ok := oMap[nf.Name]; !ok {
					op = "add"
				} else {
					op = "modify"
					ignoreMap[nf.Name] = true
				}
			} else if fp.Desc != nf.Desc {
				// eg.: alter table xxx modify `yyy` desc pos;
				op = "modify"
				oIdx += 1
			} else {
				// no change
				oIdx += 1
			}
		} else {
			// 新加
			// eg.: alter table xxx add `yyy` desc pot;
			op = "add"
		}

		if len(op) > 0 {
			var pos string
			if len(last) == 0 {
				pos = "first"
			} else {
				pos = "after " + last
			}

			// mother
			p.addSql("alter table %s %s `%s` %s %s", dstTable.Name, op, nf.Name, nf.Desc, pos)

			//child
			for _, cnm := range dstTable.ChildNames {
				p.addSql("alter table %s %s `%s` %s %s", cnm, op, nf.Name, nf.Desc, pos)
			}
		}
	}
	return nil
}

func typeTrimmer(typ string) string {
	switch typ {
	case "UNIQUE KEY":
		return "KEY"
	default:
		return typ
	}
}

func (p *differ) mysqlDiffKey(srcTable, dstTable *MysqlTable) (add, del []string, err error) {
	srcKeys := srcTable.Keys
	dstKeys := dstTable.Keys
	srcMap := make(map[string]bool, len(srcKeys))
	dstMap := make(map[string]bool, len(dstKeys))
	for _, k := range dstKeys {
		dstMap[k.Name] = true
	}

	// 先drop
	ignoreMap := make(map[string]bool)
	for _, k := range srcKeys {
		if _, ok := dstMap[k.Name]; !ok {
			ignoreMap[k.Name] = true

			// mother
			// eg.: alter table xxx drop keytype keyname
			del = append(del, fmt.Sprintf("alter table %s drop %s %s", srcTable.Name, typeTrimmer(k.Type), k.Name))

			// child
			for _, cnm := range srcTable.ChildNames {
				del = append(del, fmt.Sprintf("alter table %s drop %s %s", cnm, typeTrimmer(k.Type), k.Name))
			}
		} else {
			srcMap[k.Name] = true
		}
	}

	// 新增的和变化的
	oIdx := 0
	for _, nk := range dstKeys {
		// 找一个基准
		var kp *KeyInfo
		for oi, k := range srcKeys {
			if oi >= oIdx {
				if ignoreMap[k.Name] {
					oIdx += 1
				} else {
					kp = &k
					break
				}
			}
		}

		var op string
		if kp != nil {
			if kp.Name != nk.Name {
				if _, ok := srcMap[nk.Name]; ok {
					op = "modify"
					ignoreMap[nk.Name] = true
				} else {
					op = "add"
				}
			} else if kp.Fields != nk.Fields || kp.Type != nk.Type {
				op = "modify"
				oIdx += 1
			} else {
				// no change
				oIdx += 1
			}
		} else {
			op = "add"
		}

		if len(op) > 0 {
			// key的modify,要先drop,再add回去
			if op == "modify" {
				del = append(del, fmt.Sprintf("alter table %s drop %s %s", dstTable.Name, typeTrimmer(nk.Type), nk.Name))

				// child
				for _, cnm := range srcTable.ChildNames {
					del = append(del, fmt.Sprintf("alter table %s drop %s %s", cnm, typeTrimmer(nk.Type), nk.Name))
				}
			}

			// add
			// eg.: alter table xxx add keytype keyname (keyfield)
			add = append(add, fmt.Sprintf("alter table %s add %s %s (%s)", dstTable.Name, nk.Type, nk.Name, nk.Fields))

			// child
			for _, cnm := range srcTable.ChildNames {
				add = append(add, fmt.Sprintf("alter table %s add %s %s (%s)", cnm, nk.Type, nk.Name, nk.Fields))
			}
		}
	}
	return
}
