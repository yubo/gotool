.PHONY: install clean

all: watcher

watcher: *.go
	go build -o $@

install: watcher
	cp -a watcher ${GOPATH}/bin/watcher

clean:
	rm -f watcher
