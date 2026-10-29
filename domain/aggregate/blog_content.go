package aggregate

import (
	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/rfanazhari/go-boilerplate/domain/service"
)

type BlogContent struct {
	html   string
	length int
}

func NewBlogContent(sanitizer service.HTMLSanitizer, raw string, status enum.PublicationStatus) (BlogContent, error) {
	return BlogContent{}, nil
}

func RestoreBlogContent(html string) BlogContent {
	return BlogContent{html: html}
}

func (c BlogContent) HTML() string {
	return c.html
}

func (c BlogContent) PlainTextLength() int {
	return c.length
}
