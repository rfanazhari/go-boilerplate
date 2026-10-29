package aggregate

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

type BlogSlug struct {
	value string
}

func NewBlogSlug(trimmedTitle, suffix string) (BlogSlug, error) {
	base := slugBase(trimmedTitle)
	return BlogSlug{value: base + "-" + suffix}, nil
}

func RestoreBlogSlug(stored string) BlogSlug {
	return BlogSlug{value: stored}
}

func (s BlogSlug) String() string {
	return s.value
}

func slugBase(title string) string {
	decomposed := norm.NFKD.String(title)
	var b strings.Builder
	for _, r := range decomposed {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		r = unicode.ToLower(r)
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('-')
	}
	base := collapseHyphens(b.String())
	if base == "" {
		base = "blog"
	}
	runes := []rune(base)
	if len(runes) > 80 {
		base = strings.TrimRight(string(runes[:80]), "-")
	}
	return base
}

func collapseHyphens(value string) string {
	var b strings.Builder
	prevHyphen := false
	for _, r := range value {
		if r == '-' {
			if prevHyphen {
				continue
			}
			prevHyphen = true
			b.WriteByte('-')
			continue
		}
		prevHyphen = false
		b.WriteRune(r)
	}
	return strings.Trim(b.String(), "-")
}
