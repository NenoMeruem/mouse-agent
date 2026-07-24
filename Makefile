.PHONY: dev build build-debug test lint fmt clean

# ─────────────────────────────────────────────────────────────────────────────
# Development
# ─────────────────────────────────────────────────────────────────────────────

# Run Tauri in development mode (hot-reload)
dev:
	cd src-tauri && cargo tauri dev

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
