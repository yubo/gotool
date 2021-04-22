TARGET=$(GOPATH)/bin/mysqldoc

all: $(TARGET)

$(TARGET): *.go
	go build -o $@

.PHONY: clean
clean:
	rm -f $(TARGET)


.PHONY: test
test:
	go test -v 

