.PHONY: build build-windows build-all build-sidecar dev-tauri release test clean


# Build Go CLI binary (current platform)
build:
	go build -o prompt-agent ./cmd/prompt-agent

# Build Windows binaries (amd64 + arm64) — cross-compiled from any host
build-windows:
	@mkdir -p dist
	@echo "Building Windows amd64..."
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
		go build -ldflags="-s -w" -o dist/prompt-agent-windows-amd64.exe ./cmd/prompt-agent
	@echo "Building Windows arm64..."
	GOOS=windows GOARCH=arm64 CGO_ENABLED=0 \
		go build -ldflags="-s -w" -o dist/prompt-agent-windows-arm64.exe ./cmd/prompt-agent
	@echo ""
	@echo "✅ dist/prompt-agent-windows-amd64.exe"
	@echo "✅ dist/prompt-agent-windows-arm64.exe"

# Build for all platforms (macOS + Windows + Linux)
# Note: macOS uses CGO=1 (required by golang.design/x/hotkey).
#       Windows/Linux use CGO=0 (pure-Go, no C toolchain needed).
build-all:
	@mkdir -p dist
	@echo "--- macOS (native CGO, current arch only) ---"
	go build -ldflags="-s -w" -o dist/prompt-agent-darwin-$(shell go env GOARCH) ./cmd/prompt-agent
	@echo "--- Windows ---"
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o dist/prompt-agent-windows-amd64.exe ./cmd/prompt-agent
	GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o dist/prompt-agent-windows-arm64.exe ./cmd/prompt-agent
	@echo "--- Linux ---"
	GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o dist/prompt-agent-linux-amd64    ./cmd/prompt-agent
	GOOS=linux   GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o dist/prompt-agent-linux-arm64    ./cmd/prompt-agent
	@echo ""
	@ls -lh dist/

# Build Go binary and copy to Tauri sidecar path with target triple suffix
build-sidecar: build
	$(eval TARGET := $(shell rustc -vV | grep '^host:' | cut -d' ' -f2))
	mkdir -p src-tauri/binaries
	cp prompt-agent src-tauri/binaries/prompt-agent-cli-$(TARGET)
	@echo "Sidecar built for $(TARGET)"

# Run Tauri in dev mode (builds sidecar first)
dev-tauri: build-sidecar
	cd src-tauri && cargo tauri dev

# Build production .app + .dmg (macOS) and Windows CLI binaries
release: build-sidecar build-windows
	@mkdir -p dist
	cd src-tauri && cargo tauri build
	@echo ""
	@echo "✅ App:  src-tauri/target/release/bundle/macos/Prompt Agent.app"
	@echo "✅ DMG:  src-tauri/target/release/bundle/dmg/Prompt Agent_"*".dmg"
	@echo "✅ EXE:  dist/prompt-agent-windows-amd64.exe"
	@echo "✅ EXE:  dist/prompt-agent-windows-arm64.exe"

# Run Go tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -f prompt-agent prompt-agent.exe
	rm -f src-tauri/binaries/prompt-agent-*
	rm -rf dist/
