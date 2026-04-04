package api

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSanitizePDFDecor_NormalizesWhitespaceAndControls(t *testing.T) {
	got := sanitizePDFDecor("  Team\tAlpha \n Q2\rPlan \x00  ")
	want := "Team Alpha Q2 Plan"
	if got != want {
		t.Fatalf("sanitizePDFDecor mismatch, got %q want %q", got, want)
	}
}

func TestSanitizePDFDecor_TruncatesLongValues(t *testing.T) {
	got := sanitizePDFDecor(strings.Repeat("a", maxPDFDecorLength+15))
	if len([]rune(got)) != maxPDFDecorLength {
		t.Fatalf("expected %d runes, got %d", maxPDFDecorLength, len([]rune(got)))
	}
}

func TestParsePageDecor_ReadsQueryParams(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/export/raw/pdf?header=%20Board%20Memo%20&footer=Confidential%0AInternal", nil)
	decor := parsePageDecor(req)
	if decor.Header != "Board Memo" {
		t.Fatalf("unexpected header: %q", decor.Header)
	}
	if decor.Footer != "Confidential Internal" {
		t.Fatalf("unexpected footer: %q", decor.Footer)
	}
}

func TestPageOverridesCSS_NoOverrideReturnsEmpty(t *testing.T) {
	css := pageOverridesCSS(marginPresets["standard"], pdfPageDecor{})
	if css != "" {
		t.Fatalf("expected empty css, got %q", css)
	}
}

func TestPageOverridesCSS_IncludesMarginHeaderFooterAndEscaping(t *testing.T) {
	margins := pdfMargins{Top: "2cm", Right: "3cm", Bottom: "4cm", Left: "5cm"}
	decor := pdfPageDecor{
		Header: "Q2 \"Plan\" \\ Draft",
		Footer: "Confidential",
	}

	css := pageOverridesCSS(margins, decor)
	expected := []string{
		"<style>@page {",
		"margin: 2cm 3cm 4cm 5cm;",
		"@top-center",
		`content: "Q2 \"Plan\" \\ Draft"`,
		"@bottom-left",
		`content: "Confidential"`,
		"@page:first { margin-top: 2cm; }",
		"</style>",
	}

	for _, token := range expected {
		if !strings.Contains(css, token) {
			t.Fatalf("expected token %q in css override, got: %s", token, css)
		}
	}
}
