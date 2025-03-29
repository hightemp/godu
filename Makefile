BINARY_NAME=godu
GO=go
GOFLAGS=-trimpath
LDFLAGS=-ldflags "-s -w"
STATIC_LDFLAGS=-ldflags "-s -w -extldflags '-static'"

.PHONY: all build build-static clean test install

all: build

build:
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_NAME) .

build-static:
	CGO_ENABLED=0 $(GO) build $(GOFLAGS) $(STATIC_LDFLAGS) -o $(BINARY_NAME) .

test:
	$(GO) test -v ./...

clean:
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)_*

install: build
	install -d $(DESTDIR)/usr/local/bin/
	install -m 755 $(BINARY_NAME) $(DESTDIR)/usr/local/bin/