package prompt

import (
	"strings"
)

type Builder interface {
	Build(template string, data map[string]string) (string, error)
}

type SimpleBuilder struct{}

func NewSimpleBuilder() *SimpleBuilder {
	return &SimpleBuilder{}
}

func (b *SimpleBuilder) Build(
	template string,
	data map[string]string,
) (string, error) {
	result := template

	for k, v := range data {
		placeholder := "{{" + k + "}}"
		result = strings.ReplaceAll(result, placeholder, v)
	}

	return result, nil
}
