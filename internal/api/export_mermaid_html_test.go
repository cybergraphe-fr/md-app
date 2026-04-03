package api

import (
	"strings"
	"testing"
)

func TestBuildMermaidExportHTML_IncludesPaginationGuards(t *testing.T) {
	html := buildMermaidExportHTML("data:image/png;base64,abc123")

	expected := []string{
		`class="mermaid-diagram"`,
		`src="data:image/png;base64,abc123"`,
		`alt="Mermaid diagram"`,
		`page-break-inside: avoid`,
		`break-inside: avoid`,
		`max-height: 21cm`,
	}

	for _, token := range expected {
		if !strings.Contains(html, token) {
			t.Fatalf("expected token %q in generated html, got: %s", token, html)
		}
	}
}
