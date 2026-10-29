package aggregate_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rfanazhari/go-boilerplate/domain/aggregate"
)

type blogSlugCase struct {
	name   string
	title  string
	suffix string
	want   string
}

func TestNewBlogSlug(t *testing.T) {
	tests := []blogSlugCase{
		{
			name:   "catatan go folds to hyphenated base plus suffix",
			title:  "Catatan Go",
			suffix: "a1b2c3d4",
			want:   "catatan-go-a1b2c3d4",
		},
		{
			name:   "symbol only title falls back to blog",
			title:  "!!!",
			suffix: "deadbeef",
			want:   "blog-deadbeef",
		},
		{
			name:   "collapse repeated hyphens and trim edges",
			title:  "--Hello---World--",
			suffix: "aabbccdd",
			want:   "hello-world-aabbccdd",
		},
		{
			name:   "cut base to 80 runes then append suffix",
			title:  strings.Repeat("a", 90),
			suffix: "12345678",
			want:   strings.Repeat("a", 80) + "-12345678",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := aggregate.NewBlogSlug(tt.title, tt.suffix)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.String())
		})
	}
}
