package api

import "testing"

// TestRenderMarkdown_StripsXSS ensures the bluemonday pass neutralizes raw-HTML
// XSS that goldmark's html.WithUnsafe() would otherwise emit verbatim.
func TestRenderMarkdown_StripsXSS(t *testing.T) {
	cases := []struct {
		name       string
		md         string
		mustNot    []string
		mustHave   []string
		allowEmpty bool
	}{
		{
			name:    "script tag",
			md:      "hello\n\n<script>alert('xss')</script>\n",
			mustNot: []string{"<script", "alert("},
		},
		{
			name:    "img onerror",
			md:      "<img src=x onerror=\"alert(1)\">",
			mustNot: []string{"onerror"},
		},
		{
			name:    "javascript: href",
			md:      "[click](javascript:alert(1))",
			mustNot: []string{"javascript:"},
		},
		{
			name:     "benign heading preserved",
			md:       "# Title",
			mustHave: []string{"<h1"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := renderMarkdown(tc.md)
			if err != nil {
				t.Fatalf("renderMarkdown: %v", err)
			}
			for _, bad := range tc.mustNot {
				if containsFold(out, bad) {
					t.Fatalf("sanitized output still contains %q: %s", bad, out)
				}
			}
			for _, good := range tc.mustHave {
				if !containsFold(out, good) {
					t.Fatalf("expected %q in output, got: %s", good, out)
				}
			}
		})
	}
}

// TestRenderMarkdown_PreservesMermaidHook ensures sanitization keeps the
// data-mermaid attribute and class the frontend relies on to locate diagrams.
func TestRenderMarkdown_PreservesMermaidHook(t *testing.T) {
	out, err := renderMarkdown("```mermaid\ngraph TD;A-->B;\n```")
	if err != nil {
		t.Fatalf("renderMarkdown: %v", err)
	}
	for _, token := range []string{"data-mermaid", "mermaid-block"} {
		if !containsFold(out, token) {
			t.Fatalf("expected %q preserved after sanitization, got: %s", token, out)
		}
	}
}

func containsFold(haystack, needle string) bool {
	hl, nl := toLower(haystack), toLower(needle)
	return indexOf(hl, nl) >= 0
}

func toLower(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
}

func indexOf(s, sub string) int {
	if len(sub) == 0 {
		return 0
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
