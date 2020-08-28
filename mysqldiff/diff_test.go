package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type item struct {
	key   string
	value string
	info  string
}

func TestDiff(t *testing.T) {
	cases := []struct {
		src    []*item
		dst    []*item
		add    []*item
		del    []*item
		update []*item
	}{{
		src: []*item{
			&item{"a", "a", ""},
			&item{"b", "b", ""},
		}, dst: []*item{
			&item{"b", "b", ""},
			&item{"c", "c", ""},
		}, add: []*item{
			&item{"c", "c", "after b"},
		}, del: []*item{
			&item{"a", "a", ""},
		},
	}, {
		src: []*item{
			&item{"a", "a", ""},
			&item{"b", "b", ""},
		}, dst: []*item{
			&item{"b", "b", ""},
			&item{"a", "a", ""},
		}, update: []*item{
			&item{"b", "b", "first"},
		},
	}, {
		src: []*item{
			&item{"a", "a", ""},
			&item{"b", "b", ""},
			&item{"c", "c", ""},
		},
		dst: []*item{
			&item{"a", "a", ""},
			&item{"c", "c", ""},
			&item{"b", "b", ""},
		},
		update: []*item{
			&item{"c", "c", "after a"},
		},
	}, {
		src: []*item{
			&item{"a", "a", ""},
			&item{"b", "b", ""},
		},
		dst: []*item{
			&item{"a", "a", ""},
			&item{"a1", "a1", ""},
			&item{"a2", "a2", ""},
			&item{"b", "b", ""},
			&item{"b1", "b1", ""},
		},
		add: []*item{
			&item{"a1", "a1", "after a"},
			&item{"a2", "a2", "after a1"},
			&item{"b1", "b1", "after b"},
		},
	}}

	for i, c := range cases {
		add, del, update := diffItem(c.src, c.dst)
		require.Equal(t, c.add, add, "case-%d-add", i)
		require.Equal(t, c.del, del, "case-%d-del", i)
		require.Equal(t, c.update, update, "case-%d-update", i)
	}
}

func diffItem(oItems, nItems []*item) (add, del, update []*item) {
	oMap := make(map[string]string, len(oItems))
	nMap := make(map[string]string, len(nItems))
	for _, f := range nItems {
		nMap[f.key] = f.value
	}

	// drop
	ignoreMap := make(map[string]bool)
	for _, f := range oItems {
		if _, ok := nMap[f.key]; !ok {
			ignoreMap[f.key] = true
			del = append(del, f)
		} else {
			oMap[f.key] = f.value
		}
	}

	// update | add
	oIdx := 0
	lastFld := ""
	for _, nf := range nItems {
		var fp *item
		for i := oIdx; i < len(oItems); i++ {
			f := oItems[i]
			if ignoreMap[f.key] {
				oIdx += 1
			} else {
				fp = f
				break
			}
		}

		var op string
		var last = lastFld
		lastFld = nf.key
		if fp != nil {
			if fp.key != nf.key {
				if _, ok := oMap[nf.key]; !ok {
					op = "add"
				} else {
					op = "modify"
					ignoreMap[nf.key] = true
				}
			} else if fp.value != nf.value {
				// eg.: alter table xxx modify `yyy` desc pos;
				op = "modify"
				oIdx += 1
			} else {
				// no change
				oIdx += 1
			}
		} else {
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

			nf.info = pos
			if op == "add" {
				add = append(add, nf)
			} else {
				update = append(update, nf)
			}

		}
	}
	return
}
