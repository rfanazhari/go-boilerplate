package service_test

import (
	"github.com/google/uuid"
	"github.com/rfanazhari/go-boilerplate/domain/service"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestBlogSlug(t *testing.T) {
	type testCase struct {
		name     string
		title    string
		expected string
	}
	tests := []testCase{
		{name: "ascii", title: " Catatan Go! ", expected: "catatan-go"},
		{name: "nfkd", title: "Café ＧＯ ①", expected: "cafe-go-1"},
		{name: "fallback", title: "世界", expected: "blog"},
		{name: "separators", title: "a___b / c", expected: "a-b-c"},
		{name: "truncate", title: strings.Repeat("a", 79) + " b", expected: strings.Repeat("a", 79)},
		{name: "max base", title: strings.Repeat("a", 100), expected: strings.Repeat("a", 80)},
	}
	id := uuid.MustParse("ABCDEF12-3456-4789-8000-000000000001")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { assert.Equal(t, tt.expected+"-abcdef123456", service.BlogSlug(tt.title, id)) })
	}
}
