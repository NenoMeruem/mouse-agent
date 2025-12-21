package output

import (
	"fmt"
	"strings"
)

type Renderer interface {
	Render(title, content string) error
}

type StdoutRenderer struct {
	formatted bool
}

func NewStdoutRenderer(formatted bool) *StdoutRenderer {
	return &StdoutRenderer{
		formatted: formatted,
	}
}

func (r *StdoutRenderer) Render(title, content string) error {
	if r.formatted {
		separator := strings.Repeat("=", len(title)+4)
		fmt.Println(separator)
		fmt.Println("  " + title)
		fmt.Println(separator)
		fmt.Println(content)
		fmt.Println(separator)
	} else {
		fmt.Println(content)
	}
	return nil
}
