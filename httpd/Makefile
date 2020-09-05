.PHONY: dev devrun clean tool
TARGET=$(GOPATH)/bin/httpd

all: $(TARGET)

$(TARGET): *.go
	go build -o $@

devrun:
	@echo "$(TARGET) -p 8080"

dev:
	watcher --logtostderr -v 3 -e build -e docs -e vendor -f .go -d 1000

clean:
	rm -f $(TARGET)

tool:
	go get -u github.com/yubo/gotool/watcher
