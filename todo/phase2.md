Rất tốt 👍
**Phase 2 là phase “linh hồn” của app** – nơi bạn xây dựng **Prompt System** (schema, storage, builder).
Mình sẽ mô tả **rất chi tiết**, đủ để bạn **code từng file và chạy được**.

---

# 🧩 PHASE 2 – PROMPT SYSTEM (CHI TIẾT)

> 🎯 **Mục tiêu Phase 2**
>
> * Thiết kế prompt schema chuẩn
> * Lưu prompt (SQLite)
> * CLI: `prompt add / list / show / delete`
> * Prompt builder: merge template + data
>   ❌ Chưa cần selection text tự động
>   ❌ Chưa cần LLM

---

# 1. Kết quả mong muốn sau Phase 2

Bạn có thể:

```bash
prompt-agent prompt add
prompt-agent prompt list
prompt-agent prompt show explain_code
prompt-agent prompt delete explain_code
```

Và trong code:

* Prompt được lưu SQLite
* Prompt builder tạo được final prompt string

---

# 2. Thiết kế Prompt Schema (RẤT QUAN TRỌNG)

## 2.1 Prompt là gì trong app này?

Một **prompt** =

> Template + metadata + engine info

---

## 2.2 Prompt struct (chuẩn)

📄 `pkg/models/prompt.go`

```go
package models

import "time"

type Prompt struct {
	ID          string    // explain_code
	Name        string    // Explain selected code
	Description string
	Engine      string    // openai | gemini
	Template    string    // prompt content
	Variables   []string  // ["selection"]
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
```

📌 **Giải thích**

* `ID`: dùng để gọi CLI
* `Variables`: để validate khi build
* `Template`: raw text, có `{{selection}}`

---

## 3. Storage Layer – SQLite (Phase 2 core)

### 3.1 Vì sao SQLite?

* Local-first
* Không cần server
* Go support tốt
* Sau này dễ migrate

---

### 3.2 Table schema

```sql
CREATE TABLE prompts (
  id TEXT PRIMARY KEY,
  name TEXT,
  description TEXT,
  engine TEXT,
  template TEXT,
  variables TEXT,
  created_at DATETIME,
  updated_at DATETIME
);
```

`variables` → JSON array

---

### 3.3 Storage interface

📄 `internal/storage/storage.go`

```go
package storage

import "github.com/yourname/prompt-agent/pkg/models"

type PromptStore interface {
	Create(prompt *models.Prompt) error
	List() ([]models.Prompt, error)
	Get(id string) (*models.Prompt, error)
	Delete(id string) error
}
```

---

### 3.4 SQLite implementation

📄 `internal/storage/sqlite.go`

```go
type SQLiteStore struct {
	db *sql.DB
}
```

Constructor:

```go
func NewSQLiteStore(path string) (*SQLiteStore, error)
```

📌 DB path:

```
~/.prompt-agent/prompts.db
```

---

### 3.5 Init DB

```go
func (s *SQLiteStore) Init() error {
	_, err := s.db.Exec(schemaSQL)
	return err
}
```

👉 Gọi trong `prompt-agent init` (Phase 1 nâng cấp nhẹ)

---

## 4. Prompt CLI Commands

## 4.1 `prompt add`

### UX flow

```bash
prompt-agent prompt add
```

CLI hỏi:

```text
ID: explain_code
Name: Explain selected code
Engine [openai]:
Description:
Template (end with EOF):
```

### Implementation

* Dùng `bufio.Scanner`
* Multi-line input cho template

📄 `internal/cli/prompt_add.go`

---

## 4.2 `prompt list`

```bash
prompt-agent prompt list
```

Output:

```text
ID            ENGINE   NAME
explain_code  openai   Explain selected code
```

📄 `internal/cli/prompt_list.go`

---

## 4.3 `prompt show`

```bash
prompt-agent prompt show explain_code
```

Output:

```text
ID: explain_code
Engine: openai
Variables: selection

Template:
----------------
Bạn là senior dev...
{{selection}}
```

---

## 4.4 `prompt delete`

```bash
prompt-agent prompt delete explain_code
```

Có confirm:

```text
Are you sure? (y/N)
```

---

## 5. Prompt Builder (CORE LOGIC)

## 5.1 Builder nhiệm vụ gì?

* Nhận:

  * Prompt template
  * Data map (`selection`, `filename`, ...)
* Trả về:

  * Final string

---

## 5.2 Builder interface

📄 `internal/prompt/builder.go`

```go
package prompt

type Builder interface {
	Build(template string, data map[string]string) (string, error)
}
```

---

## 5.3 Simple implementation (Phase 2)

```go
type SimpleBuilder struct{}
```

```go
func (b *SimpleBuilder) Build(
	tmpl string,
	data map[string]string,
) (string, error) {

	result := tmpl
	for k, v := range data {
		result = strings.ReplaceAll(
			result,
			"{{"+k+"}}",
			v,
		)
	}
	return result, nil
}
```

📌 Phase sau sẽ:

* Validate missing variable
* Escape
* Support condition

---

## 6. Variable detection (quan trọng)

### Detect variables trong template

```go
{{selection}}
{{language}}
```

Regex:

```go
\{\{(\w+)\}\}
```

📄 `internal/prompt/variables.go`

```go
func ExtractVariables(tmpl string) []string
```

👉 Dùng khi `prompt add`

---

## 7. Wiring storage vào CLI

### 7.1 Init store

📄 `internal/app/context.go`

```go
type AppContext struct {
	PromptStore storage.PromptStore
}
```

Tạo context trong `rootCmd.PersistentPreRunE`

---

## 8. Test cases cần làm

### Unit test

* ExtractVariables
* Builder.Build

### Manual test

```bash
prompt-agent prompt add
prompt-agent prompt list
prompt-agent prompt show explain_code
```

---

## 9. Checkpoint cuối Phase 2

✔️ Prompt CRUD chạy
✔️ SQLite hoạt động
✔️ Builder hoạt động
✔️ CLI UX ổn

---

## 10. Sau Phase 2 bạn đã có gì?

Bạn đã có:

* **Prompt database**
* **Prompt language**
* **Prompt builder engine**

👉 Tức là **70% “trí tuệ” của app đã xong**.

---

# ➡️ PHASE 3 TIẾP THEO (Preview)

Phase 3 sẽ làm:

* Selection text (pbpaste / xclip)
* Inject selection → builder
* `prompt-agent run <id>`

---

Nếu bạn muốn, mình có thể:
1️⃣ Viết **full code mẫu cho SQLite store**
2️⃣ Thiết kế **prompt DSL nâng cao**
3️⃣ Review UX của `prompt add`
4️⃣ Chuẩn bị **migration path cho Phase 3**

👉 Bạn muốn đi **theo hướng nào tiếp?**
