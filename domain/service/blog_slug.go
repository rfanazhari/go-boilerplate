package service

import (
	"strings"
	"unicode"

	"github.com/google/uuid"
	"golang.org/x/text/unicode/norm"
)

// BlogSlug derives a permanent public address from a title and server UUID.
// Call once at creation; renaming must retain the stored slug.
func BlogSlug(title string, id uuid.UUID) string {
	var base strings.Builder
	separator := false
	for _, r := range norm.NFKD.String(title) {
		if unicode.IsMark(r) {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			if separator && base.Len() > 0 {
				base.WriteByte('-')
			}
			base.WriteRune(r)
			separator = false
		} else {
			separator = true
		}
	}
	value := base.String()
	if len(value) > 80 {
		value = strings.TrimRight(value[:80], "-")
	}
	if value == "" {
		value = "blog"
	}
	return value + "-" + strings.ReplaceAll(id.String(), "-", "")[:12]
}
