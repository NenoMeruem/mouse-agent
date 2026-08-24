.PHONY: dev dev-all dev-tauri sidecar-dev sidecar-install build build-debug test lint fmt clean

# ─────────────────────────────────────────────────────────────────────────────
# Development
# ─────────────────────────────────────────────────────────────────────────────

# Run BOTH Python LangChain Sidecar and Tauri App concurrently
dev:
	@if [ ! -d "sidecar/venv" ]; then \
		echo "Creating virtual environment in sidecar/venv..."; \
		python3 -m venv sidecar/venv && ./sidecar/venv/bin/pip install -r sidecar/requirements.txt; \
	fi
	@echo "🚀 Starting Python FastAPI Sidecar (Port 8000) & Tauri Overlay..."
	@(trap 'kill 0' SIGINT SIGTERM EXIT; \
		./sidecar/venv/bin/uvicorn sidecar.main:app --host 127.0.0.1 --port 8000 --reload & \
		cd src-tauri && cargo tauri dev)

# Alias for dev
dev-all: dev

# Run only Tauri app (assumes sidecar is running or optional)
dev-tauri:
	cd src-tauri && cargo tauri dev

# Run only FastAPI LangChain sidecar server in foreground
sidecar-dev:
	@if [ ! -d "sidecar/venv" ]; then \
		echo "Creating virtual environment in sidecar/venv..."; \
		python3 -m venv sidecar/venv && ./sidecar/venv/bin/pip install -r sidecar/requirements.txt; \
	fi
	./sidecar/venv/bin/uvicorn sidecar.main:app --host 127.0.0.1 --port 8000 --reload

# Install Python sidecar dependencies
sidecar-install:
	@if [ ! -d "sidecar/venv" ]; then \
		python3 -m venv sidecar/venv; \
	fi
	./sidecar/venv/bin/pip install -r sidecar/requirements.txt




# ─────────────────────────────────────────────────────────────────────────────
# Build (native — runs on whichever OS you are currently on)
#
#   macOS   → src-tauri/target/release/bundle/macos/Promptly.app
#             src-tauri/target/release/bundle/dmg/Promptly_*.dmg
#
#   Linux   → src-tauri/target/release/bundle/deb/promptly_*.deb
#             src-tauri/target/release/bundle/appimage/promptly_*.AppImage
#
#   Windows → src-tauri/target/release/bundle/nsis/Promptly_*-setup.exe
#             src-tauri/target/release/bundle/msi/Promptly_*.msi
#
# NOTE: Tauri does NOT support cross-compilation across different OSes.
#       Use GitHub Actions (release.yml) to produce multi-platform releases.
# ─────────────────────────────────────────────────────────────────────────────

build:
	@mkdir -p dist
	cd src-tauri && cargo tauri build
	@echo ""
	@echo "✅ Build complete — check src-tauri/target/release/bundle/"

# Debug build (skips bundle, faster iteration)
build-debug:
	cd src-tauri && cargo build

# ─────────────────────────────────────────────────────────────────────────────
# Quality checks
# ─────────────────────────────────────────────────────────────────────────────

# Run Rust unit tests
test:
	cd src-tauri && cargo test --all

# Lint (Clippy)
lint:
	cd src-tauri && cargo clippy --all-targets -- -D warnings

# Format check
fmt:
	cd src-tauri && cargo fmt --all -- --check

# ─────────────────────────────────────────────────────────────────────────────
# Cleanup
# ─────────────────────────────────────────────────────────────────────────────

clean:
	cd src-tauri && cargo clean
	rm -rf dist/
