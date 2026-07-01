package api

import (
	"net/http/httptest"
	"os"
	"path/filepath"
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
	req := httptest.NewRequest("GET", "/api/export/raw/pdf?header=%20Board%20Memo%20&footer=Confidential%0AInternal&header_align=right&footer_align=center&h1_underline_color=%232563eb&heading_text_color=%23000000&h2_text_color=%231e293b&h2_underline_color=%2394a3b8&heading_font=serif&heading_font_name=Tangerine&body_font_name=Lora", nil)
	decor := parsePageDecor(req)
	if decor.Header != "Board Memo" {
		t.Fatalf("unexpected header: %q", decor.Header)
	}
	if decor.Footer != "Confidential Internal" {
		t.Fatalf("unexpected footer: %q", decor.Footer)
	}
	if decor.HeaderAlign != "right" {
		t.Fatalf("unexpected header align: %q", decor.HeaderAlign)
	}
	if decor.FooterAlign != "center" {
		t.Fatalf("unexpected footer align: %q", decor.FooterAlign)
	}
	if decor.H1UnderlineColor != "#2563eb" {
		t.Fatalf("unexpected h1 underline color: %q", decor.H1UnderlineColor)
	}
	if decor.HeadingTextColor != "#000000" {
		t.Fatalf("unexpected heading text color: %q", decor.HeadingTextColor)
	}
	if decor.H2TextColor != "#1e293b" {
		t.Fatalf("unexpected h2 text color: %q", decor.H2TextColor)
	}
	if decor.H2UnderlineColor != "#94a3b8" {
		t.Fatalf("unexpected h2 underline color: %q", decor.H2UnderlineColor)
	}
	if decor.HeadingFont != "serif" {
		t.Fatalf("unexpected heading font: %q", decor.HeadingFont)
	}
	if decor.HeadingFontName != "Tangerine" {
		t.Fatalf("unexpected heading font name: %q", decor.HeadingFontName)
	}
	if decor.BodyFontName != "Lora" {
		t.Fatalf("unexpected body font name: %q", decor.BodyFontName)
	}
}

func TestParsePageDecor_InvalidAlignAndColorFallback(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/export/raw/pdf?header_align=side&footer_align=diag&h1_underline_color=orange&heading_text_color=oops&h2_text_color=bad&h2_underline_color=nope&heading_font=comic&heading_font_name=Comic+Sans&body_font_name=Unknown", nil)
	decor := parsePageDecor(req)
	if decor.HeaderAlign != defaultPDFDecorAlign {
		t.Fatalf("unexpected header align fallback: %q", decor.HeaderAlign)
	}
	if decor.FooterAlign != "left" {
		t.Fatalf("unexpected footer align fallback: %q", decor.FooterAlign)
	}
	if decor.H1UnderlineColor != "" {
		t.Fatalf("expected empty invalid color, got %q", decor.H1UnderlineColor)
	}
	if decor.HeadingTextColor != defaultExportHeadingTextColor {
		t.Fatalf("unexpected heading color fallback: %q", decor.HeadingTextColor)
	}
	if decor.H2TextColor != defaultExportH2TextColor {
		t.Fatalf("unexpected h2 text fallback: %q", decor.H2TextColor)
	}
	if decor.H2UnderlineColor != defaultExportH2UnderlineColor {
		t.Fatalf("unexpected h2 underline fallback: %q", decor.H2UnderlineColor)
	}
	if decor.HeadingFont != defaultExportHeadingFont {
		t.Fatalf("unexpected heading font fallback: %q", decor.HeadingFont)
	}
	if decor.HeadingFontName != "" {
		t.Fatalf("unexpected heading font name fallback: %q", decor.HeadingFontName)
	}
	if decor.BodyFontName != "" {
		t.Fatalf("unexpected body font name fallback: %q", decor.BodyFontName)
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
		Header:           "Q2 \"Plan\" \\ Draft",
		Footer:           "Confidential",
		HeaderAlign:      "right",
		FooterAlign:      "center",
		H1UnderlineColor: "#10b981",
		HeadingTextColor: "#000000",
		H2TextColor:      "#1e293b",
		H2UnderlineColor: "#94a3b8",
		HeadingFont:      "mono",
		HeadingFontName:  "Tangerine",
		BodyFontName:     "Lora",
	}

	css := pageOverridesCSS(margins, decor)
	expected := []string{
		"<style>@page {",
		"margin: 2cm 3cm 4cm 5cm;",
		"@top-right",
		`content: "Q2 \"Plan\" \\ Draft"`,
		"@bottom-center { content: ''; }",
		"@bottom-left",
		"width: 100%",
		"text-align: center",
		"white-space: nowrap",
		`content: "Confidential"`,
		"h1 { border-bottom-color: #10b981; }",
		"h1, h2, h3, h4, h5, h6 { color: #000000;",
		"font-family: 'Tangerine', 'Lora', 'Liberation Serif', 'DejaVu Serif', serif;",
		"h2 { color: #1e293b; border-bottom-color: #94a3b8; }",
		"body, p, li, td, th, blockquote, dd { font-family: 'Lora', 'Liberation Serif', 'DejaVu Serif', Georgia, serif; }",
		"@page:first { margin-top: 2cm; }",
		"</style>",
	}

	for _, token := range expected {
		if !strings.Contains(css, token) {
			t.Fatalf("expected token %q in css override, got: %s", token, css)
		}
	}
}

func TestPageOverridesCSS_LandscapeAloneEmitsPageSize(t *testing.T) {
	css := pageOverridesCSS(marginPresets["standard"], pdfPageDecor{Orientation: "landscape"})
	if !strings.Contains(css, "@page { size: A4 landscape;") {
		t.Fatalf("expected landscape @page size in css, got: %s", css)
	}
	// Standard margins must NOT be re-emitted (they stay inherited from print.css).
	if strings.Contains(css, "margin:") {
		t.Fatalf("did not expect margin override for standard margins, got: %s", css)
	}
}

func TestSanitizeOrientation(t *testing.T) {
	cases := map[string]string{
		"landscape":   "landscape",
		"LANDSCAPE":   "landscape",
		" landscape ": "landscape",
		"portrait":    "portrait",
		"":            "portrait",
		"garbage":     "portrait",
	}
	for in, want := range cases {
		if got := sanitizeOrientation(in); got != want {
			t.Fatalf("sanitizeOrientation(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTangerineStylesheetsUseWoff2Assets(t *testing.T) {
	webCSS, err := os.ReadFile(filepath.Join("..", "..", "web", "src", "app.css"))
	if err != nil {
		t.Fatalf("read web css: %v", err)
	}
	pdfCSS, err := os.ReadFile(filepath.Join("..", "..", "pandoc", "print.css"))
	if err != nil {
		t.Fatalf("read print css: %v", err)
	}

	if !strings.Contains(string(webCSS), "Tangerine-Regular.woff2") {
		t.Fatalf("web css does not reference Tangerine-Regular.woff2")
	}
	if !strings.Contains(string(pdfCSS), "Tangerine-Regular.woff2") {
		t.Fatalf("print css does not reference Tangerine-Regular.woff2")
	}
	if strings.Contains(string(webCSS), "Tangerine-Regular.ttf") {
		t.Fatalf("web css still references Tangerine-Regular.ttf")
	}
	if strings.Contains(string(pdfCSS), "Tangerine-Regular.ttf") {
		t.Fatalf("print css still references Tangerine-Regular.ttf")
	}
}
