package mcputil

import (
	"strings"
	"testing"
)

func TestStripHTML_Basic(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain text", "hello world", "hello world"},
		{"simple tag", "<b>bold</b>", "bold"},
		{"paragraph", "<p>first</p><p>second</p>", "first\n\nsecond"},
		{"br tag", "line1<br>line2", "line1\nline2"},
		{"br self-closing", "line1<br/>line2", "line1\nline2"},
		{"br with space", "line1<br />line2", "line1\nline2"},
		{"list items", "<ul><li>one</li><li>two</li></ul>", "one\ntwo"},
		{"nested", "<div><p>hello <b>world</b></p></div>", "hello world"},
		{"empty", "", ""},
		{"entities", "&amp; &lt; &gt; &quot;", "& < > \""},
		{"nbsp", "hello&nbsp;world", "hello world"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripHTML(tt.input)
			if got != tt.want {
				t.Errorf("StripHTML(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestStripHTMLWithLimit(t *testing.T) {
	long := "<p>" + strings.Repeat("a", 100) + "</p>"
	got := StripHTMLWithLimit(long, 50)
	if len(got) > 50 {
		t.Errorf("len = %d, want <= 50", len(got))
	}
}

func TestStripHTMLWithLimit_Default(t *testing.T) {
	long := "<p>" + strings.Repeat("a", 30000) + "</p>"
	got := StripHTML(long)
	if len(got) > 25000 {
		t.Errorf("default limit exceeded: len = %d", len(got))
	}
}

func TestStripHTML_WhitespaceCollapse(t *testing.T) {
	input := "<p>  lots   of    spaces  </p>"
	got := StripHTML(input)
	if strings.Contains(got, "  ") {
		t.Errorf("should collapse whitespace: %q", got)
	}
}
