package service

import (
	"sync"

	"github.com/microcosm-cc/bluemonday"
)

type HTMLSanitizer struct{}

var (
	policyOnce sync.Once
	policy     *bluemonday.Policy
)

func blogPolicy() *bluemonday.Policy {
	policyOnce.Do(func() {
		p := bluemonday.NewPolicy()
		p.AllowElements("p", "br", "h1", "h2", "h3", "h4", "h5", "a")
		p.AllowAttrs("href").OnElements("a")
		p.AllowURLSchemes("http", "https")
		p.RequireParseableURLs(true)
		p.AllowRelativeURLs(false)
		policy = p
	})
	return policy
}

func (HTMLSanitizer) Sanitize(raw string) string {
	return blogPolicy().Sanitize(raw)
}
