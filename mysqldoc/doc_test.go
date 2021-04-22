package main

import (
	"encoding/json"
	"testing"
)

var jsonContext = []byte(`
{
 "Name": "user", ` +
	" \"SqlStr\": \"CREATE TABLE `user` (  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,  `uid` varchar(128) COLLATE utf8_unicode_ci NOT NULL DEFAULT '',  `name` varchar(128) COLLATE utf8_unicode_ci NOT NULL DEFAULT '',  `title` varchar(128) COLLATE utf8_unicode_ci NOT NULL DEFAULT '',  `display_name` varchar(128) COLLATE utf8_unicode_ci NOT NULL DEFAULT '',  `email` varchar(256) COLLATE utf8_unicode_ci NOT NULL DEFAULT '',  `phone` varchar(16) COLLATE utf8_unicode_ci NOT NULL DEFAULT '',  `department` varchar(128) COLLATE utf8_unicode_ci NOT NULL DEFAULT '',  `company` varchar(128) COLLATE utf8_unicode_ci NOT NULL DEFAULT '',  `extra` blob,  `is_root` int(10) unsigned NOT NULL DEFAULT '0',  `created_at` int(10) unsigned NOT NULL DEFAULT '0',  `updated_at` int(10) unsigned NOT NULL DEFAULT '0',  `last_at` int(10) unsigned NOT NULL DEFAULT '0',  PRIMARY KEY (`id`),  UNIQUE KEY `index_name` (`name`)) ENGINE=InnoDB AUTO_INCREMENT=1005 DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci COMMENT='user';\"," +
	` "Fields": [
  {
   "Name": "id",
   "Desc": "bigint(20) unsigned NOT NULL AUTO_INCREMENT"
  },
  {
   "Name": "uid",
   "Desc": "varchar(128) COLLATE utf8_unicode_ci NOT NULL DEFAULT ''"
  },
  {
   "Name": "name",
   "Desc": "varchar(128) COLLATE utf8_unicode_ci NOT NULL DEFAULT ''"
  },
  {
   "Name": "title",
   "Desc": "varchar(128) COLLATE utf8_unicode_ci NOT NULL DEFAULT ''"
  },
  {
   "Name": "display_name",
   "Desc": "varchar(128) COLLATE utf8_unicode_ci NOT NULL DEFAULT ''"
  },
  {
   "Name": "email",
   "Desc": "varchar(256) COLLATE utf8_unicode_ci NOT NULL DEFAULT ''"
  },
  {
   "Name": "phone",
   "Desc": "varchar(16) COLLATE utf8_unicode_ci NOT NULL DEFAULT ''"
  },
  {
   "Name": "department",
   "Desc": "varchar(128) COLLATE utf8_unicode_ci NOT NULL DEFAULT ''"
  },
  {
   "Name": "company",
   "Desc": "varchar(128) COLLATE utf8_unicode_ci NOT NULL DEFAULT ''"
  },
  {
   "Name": "extra",
   "Desc": "blob"
  },
  {
   "Name": "is_root",
   "Desc": "int(10) unsigned NOT NULL DEFAULT '0'"
  },
  {
   "Name": "created_at",
   "Desc": "int(10) unsigned NOT NULL DEFAULT '0'"
  },
  {
   "Name": "updated_at",
   "Desc": "int(10) unsigned NOT NULL DEFAULT '0'"
  },
  {
   "Name": "last_at",
   "Desc": "int(10) unsigned NOT NULL DEFAULT '0'"
  }
 ],
 "Keys": [
  {
   "Name": "",
   "Type": "PRIMARY KEY",
   "Fields": "id"
  },
  {
   "Name": "index_name",
   "Type": "UNIQUE KEY",
   "Fields": "name"
  }
 ],
 "Engine": {
  "Name": "InnoDB",
  "Desc": "AUTO_INCREMENT=1005 DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci COMMENT='user'"
 },
 "IsChild": false,
 "ChildNames": null,
 "LikeTbl": ""
}
`)

func TestPrintDoc(t *testing.T) {
	tab := &MysqlTable{}
	if err := json.Unmarshal(jsonContext, tab); err != nil {
		t.Error(err)
	}

	printTableDoc(tab)
}
