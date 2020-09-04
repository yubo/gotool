# watcher

#### install
```sh
go get github.com/yubo/gotool/watcher
```


#### how to use

```
watcher -logtostderr -v 6
```

when src change below cmd will be called
```
make && make devrun
```

##### e.g. makefile

```
.PHONY: dev devrun devbuild

TMP_EXEC=example.tmp
EXEC=example

devbuild:
	go build -o $(TMP_EXEC) 01_user.go

devrun:
	@mv -f $(TMP_EXEC) $(EXEC) && echo ./$(EXEC)

dev:
	watcher  -logtostderr -v 6

```
