# mysqldoc

mysql 数据库文档导出工具

## install

```shell
go install github.com/yubo/gotool/mysqldoc@latest
```

```shell
mysqldoc --dsn="${DB_USER}:${DB_PWD}@tcp(localhost:3306)/test_db"

#### 表名 version
序号 | 名称 | 数据类型 | 允许空值 | 说明
-- | -- | -- | -- | --
1 | version | bigint unsigned | false |  version
2 | update_time | timestamp | false |  update time
3 | hostname | varchar(32) | false |  hostname

There are 3 keyword descriptions that were not found and have been added to the dictionary file ./dict.txt
```

如果需要翻译, 可使用 https://translate.google.com/ 翻译成需要的语言，然后再次运行
```shell
mysqldoc --dsn="${DB_USER}:${DB_PWD}@tcp(localhost:3306)/test_db"

#### 表名 version
序号 | 名称 | 数据类型 | 允许空值 | 说明
-- | -- | -- | -- | --
1 | version | bigint unsigned | false | 版本
2 | update_time | timestamp | false | 更新时间
3 | hostname | varchar(32) | false | 主机名
```

#### 表名 version
序号 | 名称 | 数据类型 | 允许空值 | 说明
-- | -- | -- | -- | --
1 | version | bigint unsigned | false | 版本
2 | update_time | timestamp | false | 更新时间
3 | hostname | varchar(32) | false | 主机名
