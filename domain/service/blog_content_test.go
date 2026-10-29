package service_test

import (
	"github.com/rfanazhari/go-boilerplate/domain/service"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestProcessBlogContent(t *testing.T) {
	type testCase struct {
		name    string
		source  string
		html    string
		text    string
		wantErr bool
	}
	tests := []testCase{
		{name: "empty"},
		{name: "formatting", source: `<p class="x">Hello<br>世界</p><h1>Title</h1><h2>Two</h2><h3>Three</h3><h4>Four</h4><h5>Five</h5>`, html: `<p>Hello<br/>世界</p><h1>Title</h1><h2>Two</h2><h3>Three</h3><h4>Four</h4><h5>Five</h5>`, text: "Hello 世界 Title Two Three Four Five"},
		{name: "subtrees", source: `a<script>alert(1)</script><style>x</style><svg><text>hidden</text></svg><math><mi>x</mi></math><template>secret</template><iframe>x</iframe><object>x</object><embed><noscript>x</noscript><!--x-->b`, html: "ab", text: "ab"},
		{name: "unwrap blocks", source: `<div>one</div><section>two <b>bold</b></section>`, html: " one  two bold ", text: "one two bold"},
		{name: "safe link", source: `<a href=" HTTPS://example.com/a?x=1&amp;y=2 " target="_blank" onclick="x">link</a>`, html: `<a href="HTTPS://example.com/a?x=1&amp;y=2">link</a>`, text: "link"},
		{name: "unsafe links", source: `<a href="jav&#97;script:alert(1)">a</a><a href="//example.com">b</a><a href="https://user:pass@example.com">c</a><a href="https://example.com" href="https://other.com">d</a>`, html: `<a>a</a><a>b</a><a>c</a><a>d</a>`, text: "abcd"},
		{name: "entities once", source: `<p>&amp;lt; &lt;x&gt; &nbsp; é</p>`, html: `<p>&amp;lt; &lt;x&gt;   é</p>`, text: "&lt; <x> é"},
		{name: "malformed repaired", source: `<p>Hello<b> world`, html: `<p>Hello world</p>`, text: "Hello world"},
		{name: "no length limit in processor", source: strings.Repeat("界", 1001), html: strings.Repeat("界", 1001), text: strings.Repeat("界", 1001)},
		{name: "invalid utf8", source: string([]byte{0xff}), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := service.ProcessBlogContent(tt.source)
			if tt.wantErr {
				var publicErr *errs.Error
				require.ErrorAs(t, err, &publicErr)
				assert.Equal(t, errs.CodeInvalidBlogContent, publicErr.Code)
				assert.Equal(t, errs.KindInvalidInput, publicErr.Kind)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.html, got.HTML)
			assert.Equal(t, tt.text, got.Text)
			assert.Equal(t, len([]rune(tt.text)), got.TextLength)
		})
	}
}
