.PHONY: build build-sidecar dev-tauri test clean

# Build Go CLI binary
build:
	go build -o prompt-agent ./cmd/prompt-agent

# Build Go binary and copy to Tauri sidecar path with target triple suffix
build-sidecar: build
	$(eval TARGET := $(shell rustc -vV | grep '^host:' | cut -d' ' -f2))
	mkdir -p src-tauri/binaries
	cp prompt-agent src-tauri/binaries/prompt-agent-$(TARGET)
	@echo "Sidecar built for $(TARGET)"

# Run Tauri in dev mode (builds sidecar first)
dev-tauri: build-sidecar
	cargo tauri dev --manifest-path src-tauri/Cargo.toml

# Run Go tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -f prompt-agent
	rm -f src-tauri/binaries/prompt-agent-*
