# Distribution Plan — prompt-agent

> Kế hoạch đóng gói và publish lên Homebrew, AUR (pacman), APT, GitHub Releases.

---

## Kiến trúc tổng quan

```
prompt-agent binary (Go, CGO_ENABLED=0)
        │
        ├── GitHub Release (source of truth)
        │       ├── .tar.gz (Linux/macOS)
        │       └── .zip (Windows)
        │
        ├── Homebrew tap (macOS + Linux)
        │       └── Formula tự động update qua GoReleaser
        │
        ├── AUR / pacman (Arch Linux)
        │       └── PKGBUILD tự động push qua GoReleaser
        │
        └── APT / .deb (Debian, Ubuntu)
                └── GoReleaser + nfpm → .deb → upload lên packagecloud.io
```

---

## Checklist tổng (theo thứ tự)

- [ ] **Phase 0** — Chuẩn bị
- [ ] **Phase 1** — GoReleaser config
- [ ] **Phase 2** — Homebrew tap
- [ ] **Phase 3** — AUR
- [ ] **Phase 4** — APT / .deb
- [ ] **Phase 5** — GitHub Actions CI/CD
- [ ] **Phase 6** — Logging & Debug
- [ ] **Launch** — Tag v0.1.0 và verify

---

## Phase 0 — Chuẩn bị

### Tasks

- [ ] Xác nhận GitHub username và repo URL cuối cùng
- [ ] Đổi module path trong `go.mod`: `go mod edit -module github.com/<username>/prompt-agent`
- [ ] Thêm `version` injection vào `cmd/prompt-agent/main.go`
- [ ] Tạo GitHub repo public và push code
- [ ] Tạo git tag đầu tiên

### Version injection

```go
// cmd/prompt-agent/main.go
var version = "dev" // GoReleaser inject qua ldflags

rootCmd.Version = version
```

### Tạo tag

```bash
git tag v0.1.0
git push origin v0.1.0
```

---

## Phase 1 — GoReleaser config

### Tasks

- [ ] Cài GoReleaser: `brew install goreleaser`
- [ ] Tạo file `.goreleaser.yaml` ở root
- [ ] Test dry run: `goreleaser build --snapshot --clean --single-target`
- [ ] Test full snapshot: `goreleaser release --snapshot --clean`

### `.goreleaser.yaml`

```yaml
version: 2
project_name: prompt-agent

before:
  hooks:
    - go mod tidy
    - go test ./...

builds:
  - id: prompt-agent
    main: ./cmd/prompt-agent
    binary: prompt-agent
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    env:
      - CGO_ENABLED=0   # modernc/sqlite là pure Go → cross-compile OK
    ldflags:
      - -s -w
      - -X main.version={{.Version}}

archives:
  - id: default
    format: tar.gz
    format_overrides:
      - goos: windows
        format: zip
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    files:
      - LICENSE
      - README.md

# .deb / .rpm / .apk cho Linux
nfpms:
  - id: linux-packages
    package_name: prompt-agent
    vendor: "Your Name"
    homepage: "https://github.com/<username>/prompt-agent"
    maintainer: "Your Name <your@email.com>"
    description: "AI prompt runner with recipe system"
    license: MIT
    formats:
      - deb
      - rpm
      - apk
    contents:
      - src: ./prompt-agent
        dst: /usr/local/bin/prompt-agent

# Homebrew
brews:
  - name: prompt-agent
    ids: [default]
    repository:
      owner: "<username>"
      name: homebrew-prompt-agent
      token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"
    directory: Formula
    homepage: "https://github.com/<username>/prompt-agent"
    description: "AI prompt runner with recipe system"
    license: MIT
    install: |
      bin.install "prompt-agent"
    test: |
      system "#{bin}/prompt-agent", "version"

# AUR
aurs:
  - name: prompt-agent-bin
    homepage: "https://github.com/<username>/prompt-agent"
    description: "AI prompt runner with recipe system"
    maintainers:
      - "Your Name <your@email.com>"
    license: MIT
    private_key: "{{ .Env.AUR_SSH_PRIVATE_KEY }}"
    git_url: "ssh://aur@aur.archlinux.org/prompt-agent-bin.git"
    package: |-
      install -Dm755 "./prompt-agent" "${pkgdir}/usr/bin/prompt-agent"
      install -Dm644 "./LICENSE" "${pkgdir}/usr/share/licenses/prompt-agent/LICENSE"

checksum:
  name_template: checksums.txt

changelog:
  sort: asc
  filters:
    exclude: ['^docs:', '^test:', 'Merge pull request', '^chore:']
```

---

## Phase 2 — Homebrew tap

### Tasks

- [ ] Tạo repo `github.com/<username>/homebrew-prompt-agent`
- [ ] Tạo thư mục `Formula/` trong repo đó
- [ ] Tạo GitHub Personal Access Token (scope: `repo`) cho tap repo
- [ ] Lưu token vào GitHub Actions secret: `HOMEBREW_TAP_GITHUB_TOKEN`
- [ ] Verify sau khi release: `brew tap <username>/prompt-agent && brew install prompt-agent`

### Cách user cài

```bash
brew tap <username>/prompt-agent
brew install prompt-agent
```

### Notes

> GoReleaser tự động tạo và push `Formula/prompt-agent.rb` vào tap repo sau mỗi release tag.
> Không cần tạo file Formula thủ công.

---

## Phase 3 — AUR (pacman)

### Tasks

- [ ] Tạo account tại `aur.archlinux.org`
- [ ] Upload SSH public key vào AUR profile
- [ ] Test kết nối: `ssh aur@aur.archlinux.org`
- [ ] Lưu SSH private key vào GitHub Actions secret: `AUR_SSH_PRIVATE_KEY`
- [ ] Lần đầu: clone AUR package để init repo
- [ ] Verify sau khi release: `yay -S prompt-agent-bin`

### Init AUR package (chỉ làm 1 lần)

```bash
git clone ssh://aur@aur.archlinux.org/prompt-agent-bin.git
cd prompt-agent-bin
# GoReleaser sẽ tự push PKGBUILD sau mỗi release
```

### Cách user cài

```bash
# Dùng AUR helper
yay -S prompt-agent-bin
# hoặc manual
git clone https://aur.archlinux.org/prompt-agent-bin.git
cd prompt-agent-bin && makepkg -si
```

---

## Phase 4 — APT / .deb (Debian, Ubuntu)

### Strategy

| Option | Độ khó | Phù hợp khi |
|--------|--------|-------------|
| A: GitHub Releases (file .deb) | Dễ | v0.x — user download thủ công |
| B: packagecloud.io | Trung bình | v1.0+ — `apt install` thật sự |

**Hiện tại dùng Option A, chuyển sang B khi stable.**

### Tasks — Option A (hiện tại)

- [ ] GoReleaser đã config `nfpm` → .deb tự động đính kèm vào GitHub Release
- [ ] Hướng dẫn user trong README

```bash
# User download và cài .deb
wget https://github.com/<username>/prompt-agent/releases/latest/download/prompt-agent_<version>_linux_amd64.deb
sudo dpkg -i prompt-agent_<version>_linux_amd64.deb
```

### Tasks — Option B (v1.0+, packagecloud.io)

- [ ] Tạo account tại `packagecloud.io`
- [ ] Tạo repo: `packagecloud.io/<username>/prompt-agent`
- [ ] Lưu token vào GitHub Actions secret: `PACKAGECLOUD_TOKEN`
- [ ] Thêm publisher vào `.goreleaser.yaml`:

```yaml
publishers:
  - name: packagecloud
    ids: [linux-packages]
    cmd: >-
      package_cloud push <username>/prompt-agent/debian/bookworm
      {{ abs .ArtifactPath }}
    env:
      - PACKAGECLOUD_TOKEN={{ .Env.PACKAGECLOUD_TOKEN }}
```

```bash
# User cài qua apt
curl -s https://packagecloud.io/install/repositories/<username>/prompt-agent/script.deb.sh | sudo bash
sudo apt install prompt-agent
```

---

## Phase 5 — GitHub Actions CI/CD

### Tasks

- [ ] Tạo thư mục `.github/workflows/`
- [ ] Tạo file `release.yml`
- [ ] Tạo file `ci.yml`
- [ ] Setup tất cả secrets trong GitHub repo settings

### Secrets cần thiết

| Secret | Dùng cho |
|--------|----------|
| `GITHUB_TOKEN` | Tự động có sẵn |
| `HOMEBREW_TAP_GITHUB_TOKEN` | Push Formula vào tap repo |
| `AUR_SSH_PRIVATE_KEY` | Push PKGBUILD lên AUR |
| `PACKAGECLOUD_TOKEN` | Upload .deb (chỉ khi dùng Option B) |

### `.github/workflows/release.yml`

```yaml
name: Release

on:
  push:
    tags: ['v*']

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
          cache: true

      - name: Run tests
        run: go test ./...

      - name: GoReleaser
        uses: goreleaser/goreleaser-action@v6
        with:
          version: '~> v2'
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}
          AUR_SSH_PRIVATE_KEY: ${{ secrets.AUR_SSH_PRIVATE_KEY }}
```

### `.github/workflows/ci.yml`

```yaml
name: CI

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
          cache: true
      - run: go test ./...
      - run: go vet ./...
      - name: GoReleaser snapshot (dry run)
        uses: goreleaser/goreleaser-action@v6
        with:
          args: build --snapshot --clean
```

---

## Phase 6 — Logging & Debug

### Tasks

- [ ] Tạo `internal/logger/logger.go`
- [ ] Thêm `--debug` persistent flag vào root command
- [ ] Thêm log file mode cho Tauri sidecar (log ra `~/.prompt-agent/logs/`)
- [ ] Thêm debug logs vào LLM client, run_handler, storage

### `internal/logger/logger.go`

```go
package logger

import (
    "log/slog"
    "os"
)

var L *slog.Logger

func Init(debug bool) {
    level := slog.LevelInfo
    if debug {
        level = slog.LevelDebug
    }
    L = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
        Level: level,
    }))
}

func InitWithFile(debug bool, logPath string) {
    f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        Init(debug)
        return
    }
    level := slog.LevelInfo
    if debug {
        level = slog.LevelDebug
    }
    L = slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: level}))
}

func Debug(msg string, args ...any) { L.Debug(msg, args...) }
func Info(msg string, args ...any)  { L.Info(msg, args...) }
func Error(msg string, args ...any) { L.Error(msg, args...) }
```

### Thêm `--debug` flag

```go
// cmd/prompt-agent/main.go
var debugMode bool
rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug logging")

cobra.OnInitialize(func() {
    logger.Init(debugMode)
})
```

### Debug commands

```bash
# Test build cross-platform local
goreleaser build --snapshot --clean --single-target

# Test full release local (không push lên đâu)
goreleaser release --snapshot --clean

# Run với debug logs
./prompt-agent run explain --debug 2>debug.log
tail -f debug.log

# Verify cross-compile Linux từ macOS
GOOS=linux GOARCH=amd64 go build -o /tmp/prompt-agent-linux ./cmd/prompt-agent
file /tmp/prompt-agent-linux
# → ELF 64-bit LSB executable, x86-64

# Kiểm tra binary size sau khi strip
ls -lh prompt-agent
```

---

## Thứ tự ưu tiên

| Nền tảng | Độ khó | Ưu tiên | Trạng thái |
|----------|--------|---------|------------|
| GitHub Releases (.tar.gz + .deb) | Dễ | 1 | ⬜ Chưa làm |
| Homebrew | Trung bình | 2 | ⬜ Chưa làm |
| AUR | Trung bình | 3 | ⬜ Chưa làm |
| APT repo (packagecloud) | Cao | 4 (v1.0+) | ⬜ Chưa làm |

---

## Launch checklist (final)

- [ ] Module path đã đổi
- [ ] `version` flag hoạt động: `./prompt-agent version` → `v0.1.0`
- [ ] `go test ./...` pass
- [ ] `goreleaser release --snapshot --clean` thành công local
- [ ] GitHub repo public
- [ ] homebrew-prompt-agent repo tồn tại
- [ ] AUR SSH key đã setup
- [ ] GitHub Actions secrets đã thêm đủ
- [ ] `git tag v0.1.0 && git push origin v0.1.0`
- [ ] GitHub Release hiện ra đầy đủ assets
- [ ] `brew tap <username>/prompt-agent && brew install prompt-agent` → OK
- [ ] `yay -S prompt-agent-bin` → OK
- [ ] `dpkg -i *.deb` trên Ubuntu VM → OK
