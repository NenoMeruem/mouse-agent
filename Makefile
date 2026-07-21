.PHONY: dev-tauri build-tauri test-rust test clean

# ─────────────────────────────────────────────────────────────────────────────
# Rust / Tauri commands (Go sidecar has been removed — all logic is in Rust now)
# ─────────────────────────────────────────────────────────────────────────────

# Run Tauri in development mode
dev-tauri:
	cd src-tauri && cargo tauri dev

# Build a production .app + .dmg (macOS)
build-tauri:
	@mkdir -p dist
	cd src-tauri && cargo tauri build
	@echo ""
	@echo "✅ App:  src-tauri/target/release/bundle/macos/Promptly.app"
	@echo "✅ DMG:  src-tauri/target/release/bundle/dmg/Promptly_"*".dmg"

# Run Rust unit tests (all modules: models, config, prompt, storage, llm)
test-rust:
	cd src-tauri && cargo test

# Run all tests
test: test-rust

# Clean Rust build artifacts
clean:
	cd src-tauri && cargo clean
	rm -rf dist/
