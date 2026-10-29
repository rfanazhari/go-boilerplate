package service_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	infraservice "github.com/rfanazhari/go-boilerplate/infrastructure/service"
)

type sanitizerCase struct {
	name       string
	raw        string
	wantKeep   []string
	wantOmit   []string
	wantAbsent []string
}

func TestHTMLSanitizerSanitize(t *testing.T) {
	cases := []sanitizerCase{
		{
			name:     "keeps allowed tags and absolute https href",
			raw:      `<p>hello</p><br><h1>t</h1><h2>u</h2><h3>v</h3><h4>w</h4><h5>x</h5><a href="https://example.com">link</a>`,
			wantKeep: []string{"<p>", "<br", "<h1>", "<h2>", "<h3>", "<h4>", "<h5>", `href="https://example.com"`},
		},
		{
			name:       "removes disallowed tags and keeps text",
			raw:        `<p>safe</p><script>alert(1)</script><img src="x"><span onclick="x">y</span>`,
			wantKeep:   []string{"safe"},
			wantOmit:   []string{"<script", "<img", "onclick"},
			wantAbsent: []string{"alert(1)"},
		},
		{
			name:     "strips javascript mailto and relative href",
			raw:      `<a href="javascript:alert(1)">a</a><a href="mailto:a@b.c">b</a><a href="/path">c</a><a href="https://ok.example">d</a>`,
			wantKeep: []string{`href="https://ok.example"`},
			wantOmit: []string{"javascript:", "mailto:", `href="/path"`},
		},
		{
			name:     "keeps absolute http href",
			raw:      `<a href="http://example.com">link</a>`,
			wantKeep: []string{`href="http://example.com"`},
		},
	}

	sanitizer := infraservice.HTMLSanitizer{}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizer.Sanitize(tt.raw)
			require.NotEmpty(t, got)
			for _, frag := range tt.wantKeep {
				assert.Contains(t, got, frag)
			}
			for _, frag := range tt.wantOmit {
				assert.NotContains(t, got, frag)
			}
			for _, frag := range tt.wantAbsent {
				assert.NotContains(t, got, frag)
			}
		})
	}
}
