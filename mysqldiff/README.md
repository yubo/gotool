# Mysqldiff

mysqldiff is used to compare the differences between the two databases and generate synchronous SQL 

## install

```
go install github.com/yubo/gotool/mysqldiff@latest
```

e.g.

```shell
$cat >test.sql <<'EOF'
-- drop database if exists test_old;
create database test_old;
use test_old;

CREATE TABLE `a` (
  `id`			bigint unsigned				NOT NULL AUTO_INCREMENT,
  `title`		varchar(128)		DEFAULT ''	NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE = InnoDB AUTO_INCREMENT=1000 DEFAULT CHARACTER SET = utf8 COLLATE = utf8_unicode_ci
COMMENT = 'a';


-- drop database if exists test_new;
create database test_new;
use test_new;
CREATE TABLE `a` (
  `id`			bigint unsigned				NOT NULL AUTO_INCREMENT,
  `name`		varchar(128)		DEFAULT ''	NOT NULL,
  `title`		varchar(128)		DEFAULT ''	NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `index_name` (`name`)
) ENGINE = InnoDB AUTO_INCREMENT=1000 DEFAULT CHARACTER SET = utf8 COLLATE = utf8_unicode_ci
COMMENT = 'a';
EOF

$mysql < test.sql
```

```shell
$mysqldiff --dsn1="${DB_USER}:${DB_PWD}@tcp(localhost:3306)/test_old" --dsn2="${DB_USER}:${DB_PWD}@tcp(localhost:3306)/test_new"
alter table a add `name` varchar(128) COLLATE utf8_unicode_ci NOT NULL DEFAULT '' after id;
alter table a add UNIQUE KEY index_name (`name`);
```
