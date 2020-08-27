EXEC=$(GOPATH)/bin/mysqldiff

all: $(EXEC)

$(EXEC): *.go
	go build -o $@
