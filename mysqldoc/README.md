# mysqldoc

mysql 数据库文档导出工具

## install

```
go get github.com/yubo/gotool/mysqldoc
```

e.g.

```shell
$cat >test.sql <<'EOF'
CREATE TABLE `user` (
  `id`           bigint unsigned                   NOT NULL AUTO_INCREMENT,
  `uid`          varchar(128)        DEFAULT ''    NOT NULL,
  `name`         varchar(128)        DEFAULT ''    NOT NULL,
  `title`        varchar(128)        DEFAULT ''    NOT NULL,
  `display_name` varchar(128)        DEFAULT ''    NOT NULL,
  `email`        varchar(256)        DEFAULT ''    NOT NULL,
  `phone`        varchar(16)         DEFAULT ''    NOT NULL,
  `department`   varchar(128)        DEFAULT ''    NOT NULL,
  `company`      varchar(128)        DEFAULT ''    NOT NULL,
  `extra`        blob                NULL,
  `is_root`      integer unsigned    DEFAULT '0'    NOT NULL,
  `created_at`   integer unsigned    DEFAULT '0'    NOT NULL,
  `updated_at`   integer unsigned    DEFAULT '0'    NOT NULL,
  `last_at`      integer unsigned    DEFAULT '0'    NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `index_name` (`name`)
) ENGINE = InnoDB AUTO_INCREMENT=1000 DEFAULT CHARACTER SET = utf8 COLLATE = utf8_unicode_ci
COMMENT = 'user';

EOF

$mysql test_db < test.sql
```

```shell
$mysqldoc --dsn="${DB_USER}:${DB_PWD}@tcp(localhost:3306)/test_db"

#### 表名 user
user

序号 | 字段名 | 类型 | 允许空 | 缺省值 | 备注
-- | -- | -- | -- | -- | --
0 | id | bigint(20) unsigned | false | - |
1 | uid | varchar(128) | false | '' |
2 | name | varchar(128) | false | '' |
3 | title | varchar(128) | false | '' |
4 | display_name | varchar(128) | false | '' |
5 | email | varchar(256) | false | '' |
6 | phone | varchar(16) | false | '' |
7 | department | varchar(128) | false | '' |
8 | company | varchar(128) | false | '' |
9 | extra | blob | true | - |
10 | is_root | int(10) unsigned | false | '0' |
11 | created_at | int(10) unsigned | false | '0' |
12 | updated_at | int(10) unsigned | false | '0' |
13 | last_at | int(10) unsigned | false | '0' |

索引

序号 | 索引名 | 类型 | 字段名
-- | -- | -- | --
0 | - | PRIMARY KEY | id
1 | index_name | UNIQUE KEY | name
```


#### 表名 user
user

序号 | 字段名 | 类型 | 允许空 | 缺省值 | 备注
-- | -- | -- | -- | -- | --
0 | id | bigint(20) unsigned | false | - |
1 | uid | varchar(128) | false | '' |
2 | name | varchar(128) | false | '' |
3 | title | varchar(128) | false | '' |
4 | display_name | varchar(128) | false | '' |
5 | email | varchar(256) | false | '' |
6 | phone | varchar(16) | false | '' |
7 | department | varchar(128) | false | '' |
8 | company | varchar(128) | false | '' |
9 | extra | blob | true | - |
10 | is_root | int(10) unsigned | false | '0' |
11 | created_at | int(10) unsigned | false | '0' |
12 | updated_at | int(10) unsigned | false | '0' |
13 | last_at | int(10) unsigned | false | '0' |

索引

序号 | 索引名 | 类型 | 字段名
-- | -- | -- | --
0 | - | PRIMARY KEY | id
1 | index_name | UNIQUE KEY | name
