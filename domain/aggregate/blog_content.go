package aggregate

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/rfanazhari/go-boilerplate/domain/service"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

var errInvalidBlogContent = errors.New("blog content length is outside the allowed range")

type BlogContent struct {
	html   string
	length int
}

func NewBlogContent(sanitizer service.HTMLSanitizer, raw string, status enum.PublicationStatus) (BlogContent, error) {
	html := sanitizer.Sanitize(raw)
	length := measurePlainText(html)
	if err := validateContentLength(length, status); err != nil {
		return BlogContent{}, err
	}
	return BlogContent{html: html, length: length}, nil
}

func RestoreBlogContent(html string) BlogContent {
	return BlogContent{html: html, length: measurePlainText(html)}
}

func (c BlogContent) HTML() string {
	return c.html
}

func (c BlogContent) PlainTextLength() int {
	return c.length
}

func validateContentLength(length int, status enum.PublicationStatus) error {
	min := 0
	max := 1000
	if status == enum.PublicationPublished {
		min = 100
	}
	if length < min || length > max {
		return errs.New(errs.CodeInvalidBlogContent, errs.KindInvalidInput, errInvalidBlogContent)
	}
	return nil
}

func measurePlainText(fragment string) int {
	nodes, err := html.ParseFragment(strings.NewReader(fragment), &html.Node{
		Type:     html.ElementNode,
		Data:     "body",
		DataAtom: atom.Body,
	})
	if err != nil {
		return utf8.RuneCountInString(collapseWhitespace(fragment))
	}
	var b strings.Builder
	for _, node := range nodes {
		writePlain(node, &b)
	}
	return utf8.RuneCountInString(collapseWhitespace(b.String()))
}

func writePlain(n *html.Node, b *strings.Builder) {
	if n.Type == html.TextNode {
		b.WriteString(n.Data)
	}
	if n.Type == html.ElementNode {
		switch n.Data {
		case "br":
			b.WriteByte(' ')
		case "p", "h1", "h2", "h3", "h4", "h5":
			b.WriteByte(' ')
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		writePlain(c, b)
	}
	if n.Type == html.ElementNode {
		switch n.Data {
		case "p", "h1", "h2", "h3", "h4", "h5":
			b.WriteByte(' ')
		}
	}
}

func collapseWhitespace(value string) string {
	var b strings.Builder
	prevSpace := false
	for _, r := range value {
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		prevSpace = false
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}
