.PHONY: all build fmt lint test test-integration generate clean install

all: fmt lint build

build:
	go build -o packer-plugin-numspot

fmt:
	go fmt ./...
	golangci-lint fmt 2>/dev/null || true

lint:
	go vet ./...
	golangci-lint run --fix --path-mode abs 2>/dev/null || true

test:
	go test ./... -short

test-integration:
	@if [ -z "$$NUMSPOT_CLIENT_ID" ]; then \
		echo "Error: NUMSPOT_CLIENT_ID not set. Run: source .env"; \
		exit 1; \
	fi
	go test -tags=integration ./... -v -timeout 1800s

generate:
	go generate ./...

clean:
	rm -f packer-plugin-numspot

install: build
	@mkdir -p $(HOME)/.packer.d/plugins/github.com/numspot/numspot
	@cp packer-plugin-numspot $(HOME)/.packer.d/plugins/github.com/numspot/numspot/packer-plugin-numspot_v1.0.0-dev_x5.0_darwin_arm64
	@cd $(HOME)/.packer.d/plugins/github.com/numspot/numspot && shasum -a 256 packer-plugin-numspot_v1.0.0-dev_x5.0_darwin_arm64 > packer-plugin-numspot_v1.0.0-dev_x5.0_darwin_arm64_SHA256SUM
