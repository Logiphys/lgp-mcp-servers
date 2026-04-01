package mcputil

import (
	"regexp"
	"strings"
)

const defaultMaxChars = 25000

var (
	brRe         = regexp.MustCompile(`<br\s*/?>`)
	pCloseRe     = regexp.MustCompile(`</p>`)
	pOpenRe      = regexp.MustCompile(`<p[^>]*>`)
	liRe         = regexp.MustCompile(`<li[^>]*>`)
	tagRe        = regexp.MustCompile(`<[^>]+>`)
	spacesRe     = regexp.MustCompile(`[ \t]+`)
	blankLinesRe = regexp.MustCompile(`\n{3,}`)
)

var entities = strings.NewReplacer(
	"&amp;", "&",
	"&lt;", "<",
	"&gt;", ">",
	"&quot;", "\"",
	"&#39;", "'",
	"&apos;", "'",
	"&nbsp;", " ",
)

func StripHTML(html string) string {
	return StripHTMLWithLimit(html, defaultMaxChars)
}

func StripHTMLWithLimit(html string, maxChars int) string {
	if html == "" {
		return ""
	}
	s := html
	s = brRe.ReplaceAllString(s, "\n")
	s = pCloseRe.ReplaceAllString(s, "\n\n")
	s = pOpenRe.ReplaceAllString(s, "")
	s = liRe.ReplaceAllString(s, "\n")
	s = tagRe.ReplaceAllString(s, "")
	s = entities.Replace(s)
	s = spacesRe.ReplaceAllString(s, " ")
	s = blankLinesRe.ReplaceAllString(s, "\n\n")
	s = strings.TrimSpace(s)
	if maxChars > 0 && len(s) > maxChars {
		s = s[:maxChars]
	}
	return s
}
