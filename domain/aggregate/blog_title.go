package aggregate

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

var errInvalidBlogTitle = errors.New("blog title is outside 10 to 200 code points")

type BlogTitle struct {
	value string
}

func NewBlogTitle(raw string) (BlogTitle, error) {
	trimmed := strings.TrimSpace(raw)
	n := utf8.RuneCountInString(trimmed)
	if n < 10 || n > 200 {
		return BlogTitle{}, errs.New(errs.CodeInvalidBlogTitle, errs.KindInvalidInput, errInvalidBlogTitle)
	}
	return BlogTitle{value: trimmed}, nil
}

func RestoreBlogTitle(stored string) BlogTitle {
	return BlogTitle{value: stored}
}

func (t BlogTitle) String() string {
	return t.value
}
