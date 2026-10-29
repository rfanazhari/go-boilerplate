package enum_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rfanazhari/go-boilerplate/domain/enum"
)

type parsePublicationStatusCase struct {
	name   string
	value  string
	want   enum.PublicationStatus
	wantOK bool
}

func TestParsePublicationStatus(t *testing.T) {
	tests := []parsePublicationStatusCase{
		{
			name:   "accept draft",
			value:  "draft",
			want:   enum.PublicationDraft,
			wantOK: true,
		},
		{
			name:   "accept published",
			value:  "published",
			want:   enum.PublicationPublished,
			wantOK: true,
		},
		{
			name:   "reject task todo status",
			value:  "todo",
			wantOK: false,
		},
		{
			name:   "reject deleted as a publication status",
			value:  "deleted",
			wantOK: false,
		},
		{
			name:   "reject empty value",
			value:  "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := enum.ParsePublicationStatus(tt.value)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}
