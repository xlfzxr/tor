BINARY := torstack
PKG := ./cmd/torstack
INSTALL_DIR := $(HOME)/.local/bin

.PHONY: all build vet test install clean cross

all: vet build

build:
	go build -o $(BINARY) $(PKG)

vet:
	go vet ./...

test:
	go test ./...

install: build
	mkdir -p $(INSTALL_DIR)
	install -m 755 $(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "Installed to $(INSTALL_DIR)/$(BINARY)"

clean:
	rm -f $(BINARY)

# Cross-compile for common targets
cross:
	GOOS=linux GOARCH=amd64 go build -o dist/$(BINARY)-linux-amd64 $(PKG)
	GOOS=linux GOARCH=arm64 go build -o dist/$(BINARY)-linux-arm64 $(PKG)
	GOOS=android GOARCH=arm64 go build -o dist/$(BINARY)-android-arm64 $(PKG)
	@echo "Cross binaries in dist/"
