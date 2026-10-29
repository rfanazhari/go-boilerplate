package service

import (
	"io"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/rfanazhari/go-boilerplate/pkg/errs"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// BlogContent is sanitized HTML and its normalized readable text.
// Length limits depend on publication state and are enforced by the aggregate.
type BlogContent struct {
	HTML       string
	Text       string
	TextLength int
}

func blogAllowedTag(tag string) bool {
	switch tag {
	case "p", "br", "h1", "h2", "h3", "h4", "h5", "a":
		return true
	}
	return false
}

func blogDroppedTag(tag string) bool {
	switch tag {
	case "script", "style", "iframe", "object", "embed", "template", "noscript", "svg", "math":
		return true
	}
	return false
}

func blogBlockTag(tag string) bool {
	switch tag {
	case "p", "br", "h1", "h2", "h3", "h4", "h5", "div", "section", "article", "ul", "ol", "li", "table", "thead", "tbody", "tfoot", "tr", "td", "th", "blockquote", "pre", "h6", "header", "footer", "main", "aside", "nav", "dl", "dt", "dd", "hr", "figure", "figcaption", "address", "form", "fieldset":
		return true
	}
	return false
}

func blogSafeHref(attrs []html.Attribute) string {
	value, count := "", 0
	for _, attr := range attrs {
		if attr.Namespace == "" && attr.Key == "href" {
			value = attr.Val
			count++
		}
	}
	if count != 1 {
		return ""
	}
	// Reject controls before trimming: tabs/newlines must not disguise a URL.
	if strings.ContainsFunc(value, unicode.IsControl) {
		return ""
	}
	value = strings.TrimSpace(value)
	parsed, err := url.Parse(value)
	if err != nil || parsed.Opaque != "" || parsed.Hostname() == "" || parsed.User != nil ||
		(!strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https")) {
		return ""
	}
	return value
}

// Count raw attribute names because x/net/html v0.60.0 deduplicates them
// during tokenization. Values are skipped with HTML quote rules, never decoded.
func blogHrefCount(raw string) int {
	i, count := 2, 0 // The caller has already identified an anchor start tag.
	space := func(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' }
	for i < len(raw) {
		for i < len(raw) && (space(raw[i]) || raw[i] == '/') {
			i++
		}
		if i == len(raw) || raw[i] == '>' {
			break
		}
		start := i
		for i < len(raw) && !space(raw[i]) && raw[i] != '=' && raw[i] != '/' && raw[i] != '>' {
			i++
		}
		if strings.EqualFold(raw[start:i], "href") {
			count++
		}
		if i == start {
			i++
			continue
		}
		for i < len(raw) && space(raw[i]) {
			i++
		}
		if i < len(raw) && raw[i] == '=' {
			i++
			for i < len(raw) && space(raw[i]) {
				i++
			}
			if i < len(raw) && (raw[i] == '\'' || raw[i] == '"') {
				quote := raw[i]
				i++
				for i < len(raw) && raw[i] != quote {
					i++
				}
				if i < len(raw) {
					i++
				}
			} else {
				for i < len(raw) && !space(raw[i]) && raw[i] != '>' {
					i++
				}
			}
		}
	}
	return count
}

// Preserve duplicate-attribute evidence before the parser discards it.
func blogUnambiguousLinks(source string) string {
	tokenizer := html.NewTokenizer(strings.NewReader(source))
	var result strings.Builder
	for {
		kind := tokenizer.Next()
		if kind == html.ErrorToken {
			if tokenizer.Err() == io.EOF {
				return result.String()
			}
			return source
		}
		raw := string(tokenizer.Raw())
		if kind == html.StartTagToken || kind == html.SelfClosingTagToken {
			token := tokenizer.Token()
			if token.Data == "a" && blogHrefCount(raw) > 1 {
				attrs := make([]html.Attribute, 0, len(token.Attr))
				for _, attr := range token.Attr {
					if attr.Key != "href" {
						attrs = append(attrs, attr)
					}
				}
				token.Attr = attrs
				raw = token.String()
			}
		}
		result.WriteString(raw)
	}
}

func rebuildBlogNode(parent, source *html.Node) {
	if source.Type == html.TextNode {
		parent.AppendChild(&html.Node{Type: html.TextNode, Data: source.Data})
		return
	}
	if source.Type != html.ElementNode || source.Namespace != "" || blogDroppedTag(source.Data) {
		return
	}
	allowed := blogAllowedTag(source.Data)
	block := blogBlockTag(source.Data)
	target := parent
	if allowed {
		target = &html.Node{Type: html.ElementNode, Data: source.Data, DataAtom: source.DataAtom}
		if source.Data == "a" {
			if href := blogSafeHref(source.Attr); href != "" {
				target.Attr = []html.Attribute{{Key: "href", Val: href}}
			}
		}
		parent.AppendChild(target)
	} else if block {
		parent.AppendChild(&html.Node{Type: html.TextNode, Data: " "})
	}
	for child := source.FirstChild; child != nil; child = child.NextSibling {
		rebuildBlogNode(target, child)
	}
	if !allowed && block {
		parent.AppendChild(&html.Node{Type: html.TextNode, Data: " "})
	}
}

func blogReadableText(node *html.Node, text *strings.Builder) {
	if node.Type == html.TextNode {
		text.WriteString(node.Data)
		return
	}
	block := node.Type == html.ElementNode && blogBlockTag(node.Data)
	if block {
		text.WriteByte(' ')
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		blogReadableText(child, text)
	}
	if block {
		text.WriteByte(' ')
	}
}

// ProcessBlogContent rebuilds a parsed body fragment using a closed HTML policy.
// Text is decoded by the parser exactly once; rendering escapes rebuilt nodes.
func ProcessBlogContent(source string) (BlogContent, error) {
	if !utf8.ValidString(source) {
		return BlogContent{}, errs.New(errs.CodeInvalidBlogContent, errs.KindInvalidInput, nil)
	}
	body := &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}
	nodes, err := html.ParseFragment(strings.NewReader(blogUnambiguousLinks(source)), body)
	if err != nil {
		return BlogContent{}, errs.New(errs.CodeInvalidBlogContent, errs.KindInvalidInput, err)
	}
	clean := &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}
	for _, node := range nodes {
		rebuildBlogNode(clean, node)
	}
	var rendered, readable strings.Builder
	for child := clean.FirstChild; child != nil; child = child.NextSibling {
		if err := html.Render(&rendered, child); err != nil {
			return BlogContent{}, errs.New(errs.CodeInvalidBlogContent, errs.KindInvalidInput, err)
		}
		blogReadableText(child, &readable)
	}
	text := strings.Join(strings.Fields(readable.String()), " ")
	return BlogContent{HTML: rendered.String(), Text: text, TextLength: utf8.RuneCountInString(text)}, nil
}
