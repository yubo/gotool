.PHONY: dev devrun
EXEC=$(GOPATH)/bin/vimdiff

all: $(EXEC)

$(EXEC): *.go
	go build -o $@
