GO_CACHE := $(CURDIR)/.gocache
GO_MOD_CACHE := $(CURDIR)/.gomodcache
GO_TMP := $(CURDIR)/.gotmp
GO_ENV := GOMODCACHE=$(GO_MOD_CACHE) GOCACHE=$(GO_CACHE) GOTMPDIR=$(GO_TMP)
GO_FLAGS := -ldflags=-linkmode=external

.PHONY: help analyze build test clean

help:
	@mkdir -p "$(GO_MOD_CACHE)" "$(GO_CACHE)" "$(GO_TMP)"
	$(GO_ENV) go run $(GO_FLAGS) ./cmd/stockbridge help

analyze:
	@test -n "$(TICKER)" || (echo "usage: make analyze TICKER=AMZN" && exit 1)
	@mkdir -p "$(GO_MOD_CACHE)" "$(GO_CACHE)" "$(GO_TMP)"
	$(GO_ENV) go run $(GO_FLAGS) ./cmd/stockbridge analyze "$(TICKER)"

build:
	@mkdir -p "$(GO_MOD_CACHE)" "$(GO_CACHE)" "$(GO_TMP)"
	$(GO_ENV) go build $(GO_FLAGS) ./cmd/stockbridge

test:
	@mkdir -p "$(GO_MOD_CACHE)" "$(GO_CACHE)" "$(GO_TMP)"
	$(GO_ENV) go test $(GO_FLAGS) ./...

clean:
	rm -rf "$(GO_MOD_CACHE)" "$(GO_CACHE)" "$(GO_TMP)" stockbridge
