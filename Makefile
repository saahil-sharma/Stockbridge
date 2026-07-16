GO_CACHE := $(CURDIR)/.gocache
GO_MOD_CACHE := $(CURDIR)/.gomodcache
GO_TMP := $(CURDIR)/.gotmp
GO_ENV := GOMODCACHE=$(GO_MOD_CACHE) GOCACHE=$(GO_CACHE) GOTMPDIR=$(GO_TMP)
GO_FLAGS := -ldflags=-linkmode=external
BIN_DIR := $(CURDIR)/bin
CLI_BINARY := $(BIN_DIR)/stockbridge
WEB_BINARY := $(BIN_DIR)/stockbridge-web

.PHONY: help analyze web build build-web test clean

help:
	@mkdir -p "$(GO_MOD_CACHE)" "$(GO_CACHE)" "$(GO_TMP)"
	$(GO_ENV) go run $(GO_FLAGS) ./cmd/stockbridge help

analyze:
	@test -n "$(TICKER)" || (echo "usage: make analyze TICKER=AMZN" && exit 1)
	@mkdir -p "$(GO_MOD_CACHE)" "$(GO_CACHE)" "$(GO_TMP)"
	$(GO_ENV) go run $(GO_FLAGS) ./cmd/stockbridge analyze "$(TICKER)"

web:
	@mkdir -p "$(GO_MOD_CACHE)" "$(GO_CACHE)" "$(GO_TMP)"
	$(GO_ENV) go run $(GO_FLAGS) ./cmd/stockbridge-web

build:
	@mkdir -p "$(GO_MOD_CACHE)" "$(GO_CACHE)" "$(GO_TMP)" "$(BIN_DIR)"
	$(GO_ENV) go build $(GO_FLAGS) -o "$(CLI_BINARY)" ./cmd/stockbridge

build-web:
	@mkdir -p "$(GO_MOD_CACHE)" "$(GO_CACHE)" "$(GO_TMP)" "$(BIN_DIR)"
	$(GO_ENV) go build -tags netgo -ldflags='-s -w' -o "$(WEB_BINARY)" ./cmd/stockbridge-web

test:
	@mkdir -p "$(GO_MOD_CACHE)" "$(GO_CACHE)" "$(GO_TMP)"
	$(GO_ENV) go test $(GO_FLAGS) ./...

clean:
	rm -rf "$(GO_MOD_CACHE)" "$(GO_CACHE)" "$(GO_TMP)" "$(BIN_DIR)"
