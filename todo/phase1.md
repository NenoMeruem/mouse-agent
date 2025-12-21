Tốt 👍
Mình sẽ **bắt đầu Phase 1 thật chi tiết, mức “làm theo là chạy được”**, đúng kiểu **day-by-day / step-by-step cho Go**.

> 🎯 **Mục tiêu Phase 1**
>
> * Có CLI `prompt-agent` chạy được
> * Có command cơ bản
> * Load config từ file + env
> * Chuẩn bị nền tảng cho các phase sau
>   ❌ Chưa cần LLM, chưa cần selection, chưa cần trigger

---

# 🧱 PHASE 1 – CLI Skeleton & Config (CHI TIẾT)

## 1. Kết quả mong muốn sau Phase 1

Bạn có thể chạy:

```bash
prompt-agent version
prompt-agent init
prompt-agent prompt list
```

Và có file config + folder structure chuẩn.

---

## 2. Khởi tạo project (Step 1)

### 2.1 Tạo repo & module

```bash
```

---

### 2.2 Cấu trúc thư mục chuẩn ngay từ đầu

```text
prompt-agent/
├─ cmd/
│  └─ prompt-agent/
│     └─ main.go
├─ internal/
│  ├─ config/
│  │  └─ config.go
│  └─ cli/
│     ├─ root.go
│     ├─ version.go
│     ├─ init.go
│     └─ prompt.go
├─ pkg/
│  └─ models/
├─ go.mod
└─ README.md
```

📌 **Giải thích**

* `cmd/` → entrypoint binary
* `internal/` → logic nội bộ (không expose)
* `pkg/` → model dùng chung sau này

---

## 3. Setup Cobra CLI (Step 2)

### 3.1 Cài dependency

```bash
go get github.com/spf13/cobra@latest
go get github.com/spf13/viper@latest
```

---

### 3.2 main.go

```go
package main

import "github.com/yourname/prompt-agent/internal/cli"

func main() {
	cli.Execute()
}
```

---

### 3.3 Root command

📄 `internal/cli/root.go`

```go
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "prompt-agent",
	Short: "Prompt Agent - build & run AI prompts fast",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
```

---

## 4. Command: version (Step 3)

📄 `internal/cli/version.go`

```go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "0.1.0"

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("prompt-agent version", version)
	},
}
```

👉 Test:

```bash
go run ./cmd/prompt-agent version
```

---

## 5. Config system (Viper) (Step 4)

### 5.1 Quy ước config

| Item       | Giá trị                       |
| ---------- | ----------------------------- |
| Path       | `~/.prompt-agent/config.yaml` |
| Env prefix | `PROMPT_AGENT_`               |

---

### 5.2 Config struct

📄 `internal/config/config.go`

```go
package config

type Config struct {
	Engines map[string]EngineConfig `mapstructure:"engines"`
	UI      UIConfig                `mapstructure:"ui"`
}

type EngineConfig struct {
	APIKey string `mapstructure:"api_key"`
	Model  string `mapstructure:"model"`
}

type UIConfig struct {
	Output string `mapstructure:"output"`
}
```

---

### 5.3 Load config

📄 `internal/config/config.go`

```go
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

var AppConfig *Config

func Load() error {
	home, _ := os.UserHomeDir()
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(filepath.Join(home, ".prompt-agent"))

	viper.SetEnvPrefix("PROMPT_AGENT")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("cannot read config: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return err
	}

	AppConfig = &cfg
	return nil
}
```

---

## 6. Command: init (Step 5)

### Mục tiêu

* Tạo folder `~/.prompt-agent`
* Tạo `config.yaml` mặc định

📄 `internal/cli/init.go`

```go
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize prompt-agent config",
	RunE: func(cmd *cobra.Command, args []string) error {
		home, _ := os.UserHomeDir()
		dir := filepath.Join(home, ".prompt-agent")

		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}

		configPath := filepath.Join(dir, "config.yaml")
		if _, err := os.Stat(configPath); err == nil {
			fmt.Println("Config already exists:", configPath)
			return nil
		}

		defaultConfig := `
engines:
  openai:
    api_key: ""
    model: "gpt-4.1-mini"
ui:
  output: "stdout"
`
		return os.WriteFile(configPath, []byte(defaultConfig), 0644)
	},
}
```

👉 Test:

```bash
go run ./cmd/prompt-agent init
```

---

## 7. Prompt command stub (Step 6)

📄 `internal/cli/prompt.go`

```go
package cli

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(promptCmd)
}

var promptCmd = &cobra.Command{
	Use:   "prompt",
	Short: "Manage prompts",
}
```

Sau này bạn sẽ có:

```bash
prompt-agent prompt add
prompt-agent prompt list
```

---

## 8. Wire config load vào CLI (Step 7)

📄 `internal/cli/root.go`

```go
PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
	return config.Load()
},
```

👉 Đừng quên import `internal/config`

---

## 9. Checkpoint cuối Phase 1

✔️ CLI chạy
✔️ Config init + load
✔️ Version command
✔️ Structure sạch

---

## 10. Deliverables Phase 1

* [ ] Repo compile được
* [ ] `prompt-agent init` OK
* [ ] `prompt-agent version` OK
* [ ] Config load OK

---

## ➡️ PHASE 2 TIẾP THEO

Phase 2 mình sẽ làm:

* Prompt schema
* SQLite storage
* `prompt add / list / remove`
* Prompt builder

👉 **Bạn muốn mình tiếp tục Phase 2 ngay, hay muốn mình:**
1️⃣ Review lại Phase 1 (code style, best practice)
2️⃣ Điều chỉnh cấu trúc cho cross-platform
3️⃣ Viết sẵn repo skeleton hoàn chỉnh
