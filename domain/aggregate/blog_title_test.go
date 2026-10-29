package aggregate_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rfanazhari/go-boilerplate/domain/aggregate"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

type blogTitleCase struct {
	name      string
	raw       string
	wantErr   bool
	wantCode  string
	wantValue string
}

func TestNewBlogTitle(t *testing.T) {
	tests := []blogTitleCase{
		{
			name:      "trim unicode space then accept catatan go at 10 runes",
			raw:       "  Catatan Go  ",
			wantValue: "Catatan Go",
		},
		{
			name:     "reject 9 runes after trim",
			raw:      "Catatan G",
			wantErr:  true,
			wantCode: errs.CodeInvalidBlogTitle,
		},
		{
			name:      "accept 10 runes",
			raw:       "Catatan Go",
			wantValue: "Catatan Go",
		},
		{
			name:      "accept 200 runes",
			raw:       strings.Repeat("あ", 200),
			wantValue: strings.Repeat("あ", 200),
		},
		{
			name:     "reject 201 runes",
			raw:      strings.Repeat("あ", 201),
			wantErr:  true,
			wantCode: errs.CodeInvalidBlogTitle,
		},
		{
			name:     "reject empty after trim",
			raw:      " \u00a0\t\n",
			wantErr:  true,
			wantCode: errs.CodeInvalidBlogTitle,
		},
		{
			name:      "keep angle brackets as plain text",
			raw:       "Title <ok> xx",
			wantValue: "Title <ok> xx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := aggregate.NewBlogTitle(tt.raw)
			if tt.wantErr {
				require.Error(t, err)
				var domainErr *errs.Error
				require.ErrorAs(t, err, &domainErr)
				assert.Equal(t, tt.wantCode, domainErr.Code)
				assert.NotEqual(t, errs.CodeInvalidTitle, domainErr.Code)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantValue, got.String())
		})
	}
}
