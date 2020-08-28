TARGET=$(GOPATH)/bin/mysqldiff

all: $(TARGET)

$(TARGET): *.go
	go build -o $@

.PHONY: clean
clean:
	rm -f $(TARGET)


.PHONY: test
test:
	go test -v --run TestDiff

