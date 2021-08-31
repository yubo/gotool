.PHONY: dev devrun
EXEC=$(GOPATH)/bin/watcher

all: $(EXEC)

$(EXEC): *.go
	go build -o $@

clean:
	rm -f $(EXEC)
