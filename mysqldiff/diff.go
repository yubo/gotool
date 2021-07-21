package main

import (
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/yubo/golib/orm"
)

type Differ struct {
	*Config
	oDb  *orm.DB
	nDb  *orm.DB
	sqls []string
}

func (p *Differ) addSql(sql string, args ...interface{}) {
	p.sqls = append(p.sqls, fmt.Sprintf(sql, args...))
}

func (p *Differ) Do() error {
	for _, v := range p.sqls {
		fmt.Println(v + ";")
		if !p.exec {
			continue
		}

		if err := p.oDb.ExecErr(v); err != nil {
			return err
		}
	}
	return nil
}

func (p *Differ) Conn() error {
	var err error
	if p.oDb, err = orm.DbOpen("mysql", p.oDsn); err != nil {
		return err
	}
	if p.nDb, err = orm.DbOpen("mysql", p.nDsn); err != nil {
		return err
	}
	return nil
}

func (p *Differ) Close() error {
	p.oDb.Close()
	p.nDb.Close()
	return nil
}

func (p *Differ) CompareDb() error {
	var oTab []string
	if err := p.oDb.Query("show tables").Rows(&oTab); err != nil {
		return err
	}

	var nTab []string
	if err := p.nDb.Query("show tables").Rows(&nTab); err != nil {
		return err
	}

	add, drop, update := strDiff(oTab, nTab)

	for _, v := range add {
		if sql, err := getTableCreateSql(p.nDb, v); err != nil {
			return err
		} else {
			p.addSql(sql)
		}
	}

	for _, v := range drop {
		p.addSql("drop table %s", v)
	}

	for _, v := range update {
		if err := p.compareTable(v); err != nil {
			return err
		}
	}

	return nil
}

func (p *Differ) compareTable(tableName string) error {
	d, err := parseTable(p.nDb, tableName)
	if err != nil {
		return err
	}
	if d.IsChild {
		return nil
	}

	s, err := parseTable(p.oDb, tableName)
	if err != nil {
		return err
	}

	add, drop, err := p.mysqlDiffKey(s, d)
	if err != nil {
		return err
	}

	// 1. drop index
	for _, v := range drop {
		p.addSql(v)
	}
	// 2. drop & add field
	if err := p.mysqlDiffField(s, d); err != nil {
		return err
	}
	// 3. add index
	for _, v := range add {
		p.addSql(v)
	}

	return nil
}

func getTableCreateSql(db *orm.DB, table string) (sql string, err error) {
	var name string
	err = db.Query("show create table "+table).Row(&name, &sql)
	return
}

func (p *Differ) mysqlDiffField(oTab, nTab *MysqlTable) error {
	oFlds := oTab.Fields
	nFlds := nTab.Fields
	oMap := make(map[string]string, len(oFlds))
	nMap := make(map[string]string, len(nFlds))
	for _, f := range nFlds {
		nMap[f.Name] = f.Desc
	}

	// drop
	ignoreMap := make(map[string]bool)
	for _, f := range oFlds {
		if _, ok := nMap[f.Name]; !ok {
			ignoreMap[f.Name] = true

			// mother
			p.addSql("alter table %s %s `%s`", oTab.Name, "drop", f.Name)

			// child
			for _, cnm := range oTab.ChildNames {
				p.addSql("alter table %s %s `%s`", cnm, "drop", f.Name)
			}
		} else {
			oMap[f.Name] = f.Desc
		}
	}

	// update | add
	oIdx := 0
	lastFld := ""
	for _, nf := range nFlds {
		// 找一个基准
		var fp *FieldInfo
		for i := oIdx; i < len(oFlds); i++ {
			f := oFlds[i]
			if ignoreMap[f.Name] {
				oIdx += 1
			} else {
				fp = &f
				break
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
			p.addSql("alter table %s %s `%s` %s %s", nTab.Name, op, nf.Name, nf.Desc, pos)

			//child
			for _, cnm := range nTab.ChildNames {
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

func (p *Differ) mysqlDiffKey(oTab, nTab *MysqlTable) (add, del []string, err error) {
	oKeys := oTab.Keys
	nKeys := nTab.Keys
	oMap := make(map[string]bool, len(oKeys))
	nMap := make(map[string]bool, len(nKeys))
	for _, k := range nKeys {
		nMap[k.Name] = true
	}

	// drop
	ignoreMap := make(map[string]bool)
	for _, k := range oKeys {
		if _, ok := nMap[k.Name]; !ok {
			ignoreMap[k.Name] = true

			// mother
			// eg.: alter table xxx drop keytype keyname
			del = append(del, fmt.Sprintf("alter table %s drop %s %s", oTab.Name, typeTrimmer(k.Type), k.Name))

			// child
			for _, cnm := range oTab.ChildNames {
				del = append(del, fmt.Sprintf("alter table %s drop %s %s", cnm, typeTrimmer(k.Type), k.Name))
			}
		} else {
			oMap[k.Name] = true
		}
	}

	oIdx := 0
	for _, nk := range nKeys {
		var kp *KeyInfo
		for i := oIdx; i < len(oKeys); i++ {
			k := oKeys[i]
			if ignoreMap[k.Name] {
				oIdx += 1
			} else {
				kp = &k
				break
			}
		}

		var op string
		if kp != nil {
			if kp.Name != nk.Name {
				if _, ok := oMap[nk.Name]; ok {
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
			// key modify, drop -> add
			if op == "modify" {
				del = append(del, fmt.Sprintf("alter table %s drop %s %s", nTab.Name, typeTrimmer(nk.Type), nk.Name))

				// child
				for _, cnm := range oTab.ChildNames {
					del = append(del, fmt.Sprintf("alter table %s drop %s %s", cnm, typeTrimmer(nk.Type), nk.Name))
				}
			}

			// add
			// eg.: alter table xxx add keytype keyname (keyfield)
			add = append(add, fmt.Sprintf("alter table %s add %s %s (%s)", nTab.Name, nk.Type, nk.Name, nk.Fields))

			// child
			for _, cnm := range oTab.ChildNames {
				add = append(add, fmt.Sprintf("alter table %s add %s %s (%s)", cnm, nk.Type, nk.Name, nk.Fields))
			}
		}
	}
	return
}

func strDiff(o, n []string) (add, del, eq []string) {
	s := map[string]bool{}
	d := map[string]bool{}

	for _, v := range o {
		s[v] = true
	}

	for _, v := range n {
		d[v] = true
		if !s[v] {
			add = append(add, v)
		} else {
			eq = append(eq, v)
		}
	}

	for _, v := range o {
		if !d[v] {
			del = append(del, v)
		}
	}

	return
}
