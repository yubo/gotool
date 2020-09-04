TARGET=$(GOPATH)/bin/rpmdumper

all: $(TARGET)

$(TARGET): *.go
	go build -o $@

.PHONY: clean
clean:
	rm -f $(TARGET)
