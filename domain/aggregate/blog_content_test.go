package aggregate_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rfanazhari/go-boilerplate/domain/aggregate"
	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

type identitySanitizer struct{}

func (identitySanitizer) Sanitize(raw string) string { return raw }

type prefixSanitizer struct{}

func (prefixSanitizer) Sanitize(raw string) string { return "<p>" + raw + "</p>" }

type blogContentCase struct {
	name     string
	raw      string
	status   enum.PublicationStatus
	wantErr  bool
	wantCode string
	wantHTML string
	wantLen  int
}

func TestNewBlogContent(t *testing.T) {
	tests := []blogContentCase{
		{
			name:     "measure adjacent paragraphs as hello world length 11",
			raw:      "<p>hello</p><p>world</p>",
			status:   enum.PublicationDraft,
			wantHTML: "<p>hello</p><p>world</p>",
			wantLen:  11,
		},
		{
			name:     "measure br as a word separator length 11",
			raw:      "<p>hello<br>world</p>",
			status:   enum.PublicationDraft,
			wantHTML: "<p>hello<br>world</p>",
			wantLen:  11,
		},
		{
			name:     "decode entities before counting",
			raw:      "<p>&amp;</p>",
			status:   enum.PublicationDraft,
			wantHTML: "<p>&amp;</p>",
			wantLen:  1,
		},
		{
			name:     "whitespace only html has length 0",
			raw:      "<p>   </p>",
			status:   enum.PublicationDraft,
			wantHTML: "<p>   </p>",
			wantLen:  0,
		},
		{
			name:     "count 100 repeated a as 100",
			raw:      strings.Repeat("a", 100),
			status:   enum.PublicationDraft,
			wantHTML: strings.Repeat("a", 100),
			wantLen:  100,
		},
		{
			name:     "draft accepts empty content",
			raw:      "",
			status:   enum.PublicationDraft,
			wantHTML: "",
			wantLen:  0,
		},
		{
			name:     "draft accepts 1000 runes",
			raw:      strings.Repeat("a", 1000),
			status:   enum.PublicationDraft,
			wantHTML: strings.Repeat("a", 1000),
			wantLen:  1000,
		},
		{
			name:     "draft rejects 1001 runes",
			raw:      strings.Repeat("a", 1001),
			status:   enum.PublicationDraft,
			wantErr:  true,
			wantCode: errs.CodeInvalidBlogContent,
		},
		{
			name:     "published rejects 99 runes",
			raw:      strings.Repeat("a", 99),
			status:   enum.PublicationPublished,
			wantErr:  true,
			wantCode: errs.CodeInvalidBlogContent,
		},
		{
			name:     "published accepts 100 runes",
			raw:      strings.Repeat("a", 100),
			status:   enum.PublicationPublished,
			wantHTML: strings.Repeat("a", 100),
			wantLen:  100,
		},
		{
			name:     "published accepts 1000 runes",
			raw:      strings.Repeat("a", 1000),
			status:   enum.PublicationPublished,
			wantHTML: strings.Repeat("a", 1000),
			wantLen:  1000,
		},
		{
			name:     "published rejects 1001 runes",
			raw:      strings.Repeat("a", 1001),
			status:   enum.PublicationPublished,
			wantErr:  true,
			wantCode: errs.CodeInvalidBlogContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := aggregate.NewBlogContent(identitySanitizer{}, tt.raw, tt.status)
			if tt.wantErr {
				require.Error(t, err)
				var domainErr *errs.Error
				require.ErrorAs(t, err, &domainErr)
				assert.Equal(t, tt.wantCode, domainErr.Code)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantHTML, got.HTML())
			assert.Equal(t, tt.wantLen, got.PlainTextLength())
		})
	}
}

func TestNewBlogContentUsesSanitizerOutput(t *testing.T) {
	got, err := aggregate.NewBlogContent(prefixSanitizer{}, "hello", enum.PublicationDraft)
	require.NoError(t, err)
	assert.Equal(t, "<p>hello</p>", got.HTML())
	assert.Equal(t, 5, got.PlainTextLength())
}
