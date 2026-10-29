package enum_test

import (
	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseBlogStatus(t *testing.T) {
	type testCase struct {
		name     string
		value    string
		expected enum.BlogStatus
		valid    bool
	}
	tests := []testCase{
		{name: "draft", value: "draft", expected: enum.BlogDraft, valid: true},
		{name: "published", value: "published", expected: enum.BlogPublished, valid: true},
		{name: "empty"},
		{name: "deleted is not publication status", value: "deleted"},
		{name: "task status", value: "done"},
		{name: "uppercase", value: "DRAFT"},
		{name: "whitespace", value: " draft "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, valid := enum.ParseBlogStatus(tt.value)
			assert.Equal(t, tt.expected, actual)
			assert.Equal(t, tt.valid, valid)
		})
	}
}
