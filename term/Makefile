.PHONY: dev devbuild devrun arti_push coverage parse doc test vendor reload install rpm devtools

GOFILES=$(shell find . -name "*.go" -type f -not -path "./vendor*")
TARGETS=.build/server
APP=server
all: $(TARGETS)

.build/server: $(GOFILES) .build
	go build -o $@ ./server

.build:
	@mkdir .build

devrun:
	@echo "./.build/$(APP)"

dev:
	watcher -logtostderr -v 3 -e dist -e pkg/data -e docs -e vendor -f .go -f .sql -d 100
