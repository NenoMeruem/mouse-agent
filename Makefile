.PHONY: build build-windows build-all build-sidecar dev-tauri release test clean


# Build Go CLI binary (current platform)
build:
	go build -o promptly ./cmd/promptly

# Build Windows binaries (amd64 + arm64) — cross-compiled from any host
build-windows:
	@mkdir -p dist
	@echo "Building Windows amd64..."
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
		go build -ldflags="-s -w" -o dist/promptly-windows-amd64.exe ./cmd/promptly
	@echo "Building Windows arm64..."
	GOOS=windows GOARCH=arm64 CGO_ENABLED=0 \
		go build -ldflags="-s -w" -o dist/promptly-windows-arm64.exe ./cmd/promptly
	@echo ""
	@echo "✅ dist/promptly-windows-amd64.exe"
	@echo "✅ dist/promptly-windows-arm64.exe"

# Build for all platforms (macOS + Windows + Linux)
# Note: macOS uses CGO=1 (required by golang.design/x/hotkey).
#       Windows/Linux use CGO=0 (pure-Go, no C toolchain needed).
build-all:
	@mkdir -p dist
	@echo "--- macOS (native CGO, current arch only) ---"
	go build -ldflags="-s -w" -o dist/promptly-darwin-$(shell go env GOARCH) ./cmd/promptly
	@echo "--- Windows ---"
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o dist/promptly-windows-amd64.exe ./cmd/promptly
	GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o dist/promptly-windows-arm64.exe ./cmd/promptly
	@echo "--- Linux ---"
	GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o dist/promptly-linux-amd64    ./cmd/promptly
	GOOS=linux   GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o dist/promptly-linux-arm64    ./cmd/promptly
	@echo ""
	@ls -lh dist/

# Build Go binary and copy to Tauri sidecar path with target triple suffix
build-sidecar: build
	$(eval TARGET := $(shell rustc -vV | grep '^host:' | cut -d' ' -f2))
	mkdir -p src-tauri/binaries
	cp promptly src-tauri/binaries/promptly-cli-$(TARGET)
	@echo "Sidecar built for $(TARGET)"

# Run Tauri in dev mode (builds sidecar first)
dev-tauri: build-sidecar
	cd src-tauri && cargo tauri dev

# Build production .app + .dmg (macOS) and Windows CLI binaries
release: build-sidecar build-windows
	@mkdir -p dist
	cd src-tauri && cargo tauri build
	@echo ""
	@echo "✅ App:  src-tauri/target/release/bundle/macos/Promptly.app"
	@echo "✅ DMG:  src-tauri/target/release/bundle/dmg/Promptly_"*".dmg"
	@echo "✅ EXE:  dist/promptly-windows-amd64.exe"
	@echo "✅ EXE:  dist/promptly-windows-arm64.exe"

# Run Go tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -f promptly promptly.exe
	rm -f src-tauri/binaries/promptly-*
	rm -rf dist/
