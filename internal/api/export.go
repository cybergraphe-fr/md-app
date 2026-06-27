package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"md/internal/config"
	"md/internal/storage"
)

type exportHandler struct {
	basePath string
	cfg      *config.Config
	// sem caps simultaneous export pipelines (DoS hardening). Each export
	// forks pandoc + WeasyPrint and, for PDF, one headless Chromium per
	// Mermaid block; an unbounded fan-out would exhaust CPU/RAM/PIDs. Sized
	// from MD_MAX_CONCURRENT_CONVERSIONS (default 4).
	sem chan struct{}
}

func newExportHandler(basePath string, cfg *config.Config) *exportHandler {
	return &exportHandler{basePath: basePath, cfg: cfg, sem: make(chan struct{}, exportEnvInt("MD_MAX_CONCURRENT_CONVERSIONS", 4))}
}

// exportEnvInt reads a positive integer tunable from the environment, falling
// back to def. Backs the DoS-hardening knobs (export concurrency, Mermaid cap)
// so they are self-contained and do not depend on optional config fields.
func exportEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

// acquire takes a conversion slot without blocking; it returns false (so the
// caller can reply 503) when the pipeline is already at capacity.
func (h *exportHandler) acquire() bool {
	select {
	case h.sem <- struct{}{}:
		return true
	default:
		return false
	}
}

func (h *exportHandler) release() { <-h.sem }

func (h *exportHandler) store(r *http.Request) *storage.Storage {
	return ScopedStorage(h.basePath, r)
}

// ─── Pandoc input format ────────────────────────────────────
// Comprehensive format string that matches the GFM-like rendering of
// marked.js in the webapp preview.  Every extension is explicit so that
// upgrades to the Pandoc version never silently remove support.
// stderrBufPool reuses bytes.Buffer instances for command stderr capture.
var stderrBufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

const pandocInputFmt = "markdown" +
	"+pipe_tables" +
	"+grid_tables" +
	"+multiline_tables" +
	"+simple_tables" +
	"+hard_line_breaks" +
	"+table_captions" +
	"+strikeout" +
	"+task_lists" +
	"+definition_lists" +
	"+footnotes" +
	"+smart" +
	"+emoji" +
	"+autolink_bare_uris" +
	"+raw_html" +
	"+fenced_code_blocks" +
	"+backtick_code_blocks" +
	"+fenced_code_attributes" +
	"+inline_code_attributes" +
	"+yaml_metadata_block" +
	"+tex_math_dollars" +
	"+superscript" +
	"+subscript" +
	"+abbreviations" +
	"+header_attributes"

// ─── Page-break support ─────────────────────────────────────
const pageBreakDiv = `<div class="pagebreak"></div>`

var rePageBreak = regexp.MustCompile(`(?m)^\\(?:newpage|pagebreak)\s*$|^<!--\s*pagebreak\s*-->\s*$|^---\s*pagebreak\s*---\s*$`)

func preprocessPageBreaks(content string) string {
	return rePageBreak.ReplaceAllString(content, pageBreakDiv)
}

// ─── Mermaid SSR for export ─────────────────────────────────
// mermaidSSRScript is the path to the Node.js mermaid renderer.
const mermaidSSRScript = "/app/mmdc/render.mjs"

const (
	// Keep exported Mermaid diagrams within a single printable page area.
	mermaidExportImageMaxHeightCM = "21cm"
	mermaidExportWrapperStyle     = "page-break-inside: avoid; break-inside: avoid; text-align: center; margin: 1.2em 0;"
	mermaidExportImageStyle       = "display: block; margin: 0 auto; max-width: 100%; width: auto; height: auto; max-height: " + mermaidExportImageMaxHeightCM + "; object-fit: contain; page-break-inside: avoid; break-inside: avoid;"
)

func buildMermaidExportHTML(dataURI string) string {
	return "<div class=\"mermaid-diagram\" style=\"" + mermaidExportWrapperStyle +
		"\"><img src=\"" + dataURI + "\" alt=\"Mermaid diagram\" style=\"" + mermaidExportImageStyle + "\" /></div>"
}

// renderMermaid calls the mermaid SSR script to convert source to the
// requested format (svg or png). For png, stdout is base64-encoded binary.
func renderMermaid(ctx context.Context, source, format string) (string, error) {
	f, err := os.CreateTemp("", "mermaid-*.mmd")
	if err != nil {
		return "", err
	}
	name := f.Name()
	defer os.Remove(name)

	if _, err = f.WriteString(source); err != nil {
		f.Close()
		return "", err
	}
	f.Close()

	var stdout bytes.Buffer
	stderr := stderrBufPool.Get().(*bytes.Buffer)
	stderr.Reset()
	defer stderrBufPool.Put(stderr)

	cmd := exec.CommandContext(ctx, "node", mermaidSSRScript, name, format)
	cmd.Stdout = &stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w — %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

// preprocessMermaidForExport replaces mermaid fences with rendered diagrams.
// For PDF reliability, we embed rendered PNG data URIs (no JS, no SVG
// foreignObject support dependency in WeasyPrint).
// Falls back to a plain code block if SSR fails.
func preprocessMermaidForExport(ctx context.Context, content string, maxBlocks int) string {
	if maxBlocks < 1 {
		maxBlocks = 1
	}
	blocks := 0
	return processMermaidFences(content, func(source string) string {
		blocks++
		if blocks > maxBlocks {
			slog.Warn("mermaid block cap reached, keeping as code block", "cap", maxBlocks)
			return "```\n" + source + "\n```"
		}
		pngB64, err := renderMermaid(ctx, source, "png")
		if err != nil {
			slog.Warn("mermaid SSR render failed, keeping as code block", "error", err)
			return "```\n" + source + "\n```"
		}
		dataURI := "data:image/png;base64," + strings.TrimSpace(pngB64)
		return buildMermaidExportHTML(dataURI)
	})
}

// ─── Margin presets ─────────────────────────────────────────
type pdfMargins struct {
	Top    string
	Right  string
	Bottom string
	Left   string
}

type pdfPageDecor struct {
	Header           string
	Footer           string
	HeaderAlign      string
	FooterAlign      string
	H1UnderlineColor string
	HeadingTextColor string
	H2TextColor      string
	H2UnderlineColor string
	HeadingFont      string
	HeadingFontName  string
	BodyFontName     string
}

const maxPDFDecorLength = 120
const defaultPDFDecorAlign = "center"
const defaultExportHeadingTextColor = "#111111"
const defaultExportH2TextColor = "#111111"
const defaultExportH2UnderlineColor = "#cbd5e1"
const defaultExportHeadingFont = "sans"

var allowedExportFontNames = map[string]struct{}{
	"Lora": {}, "Merriweather": {}, "Playfair Display": {}, "Source Serif 4": {}, "Tangerine": {},
	"Inter": {}, "Roboto": {}, "Open Sans": {}, "Poppins": {}, "Exo 2": {},
	"Ubuntu": {}, "Nunito Sans": {}, "Raleway": {}, "Helvetica": {},
}

var cssContentEscaper = strings.NewReplacer("\\", "\\\\", "\"", "\\\"")
var pdfHexColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

var marginPresets = map[string]pdfMargins{
	"standard": {"2.2cm", "2.5cm", "2.5cm", "2.5cm"},
	"narrow":   {"1.5cm", "1.5cm", "1.5cm", "1.5cm"},
	"wide":     {"2.5cm", "3.5cm", "2.5cm", "3.5cm"},
}

// parseMargins reads margin settings from the HTTP request.
// Accepts: ?margin=standard|narrow|wide  (preset)
//
//	?mt=2&mr=2&mb=2&ml=2     (custom, in cm)
func parseMargins(r *http.Request) pdfMargins {
	preset := r.URL.Query().Get("margin")
	if m, ok := marginPresets[preset]; ok {
		return m
	}
	// Custom margins (cm) — all four must be set, otherwise use standard
	mt := r.URL.Query().Get("mt")
	mr := r.URL.Query().Get("mr")
	mb := r.URL.Query().Get("mb")
	ml := r.URL.Query().Get("ml")
	if mt != "" && mr != "" && mb != "" && ml != "" {
		return pdfMargins{asCM(mt), asCM(mr), asCM(mb), asCM(ml)}
	}
	return marginPresets["standard"]
}

func asCM(v string) string {
	if _, err := strconv.ParseFloat(v, 64); err == nil {
		return v + "cm"
	}
	return "2.5cm"
}

func parsePageDecor(r *http.Request) pdfPageDecor {
	return pdfPageDecor{
		Header:           sanitizePDFDecor(r.URL.Query().Get("header")),
		Footer:           sanitizePDFDecor(r.URL.Query().Get("footer")),
		HeaderAlign:      sanitizePDFDecorAlign(r.URL.Query().Get("header_align"), defaultPDFDecorAlign),
		FooterAlign:      sanitizePDFDecorAlign(r.URL.Query().Get("footer_align"), "left"),
		H1UnderlineColor: sanitizePDFHexColor(r.URL.Query().Get("h1_underline_color")),
		HeadingTextColor: sanitizePDFHexColorWithDefault(r.URL.Query().Get("heading_text_color"), defaultExportHeadingTextColor),
		H2TextColor:      sanitizePDFHexColorWithDefault(r.URL.Query().Get("h2_text_color"), defaultExportH2TextColor),
		H2UnderlineColor: sanitizePDFHexColorWithDefault(r.URL.Query().Get("h2_underline_color"), defaultExportH2UnderlineColor),
		HeadingFont:      sanitizeHeadingFont(r.URL.Query().Get("heading_font")),
		HeadingFontName:  sanitizeExportFontName(r.URL.Query().Get("heading_font_name")),
		BodyFontName:     sanitizeExportFontName(r.URL.Query().Get("body_font_name")),
	}
}

func pageDecorBoxRule(position string, align string, content string) string {
	if align == "center" {
		return fmt.Sprintf(" @%s-center { content: ''; } @%s-left { content: %s; width: 100%%; text-align: center; white-space: nowrap; font-family: 'Liberation Sans', 'DejaVu Sans', sans-serif; font-size: 8.5pt; color: #6b7280; }", position, position, content)
	}

	return fmt.Sprintf(" @%s-%s { content: %s; font-family: 'Liberation Sans', 'DejaVu Sans', sans-serif; font-size: 8.5pt; color: #6b7280; white-space: nowrap; }", position, align, content)
}

func sanitizeExportFontName(raw string) string {
	name := strings.TrimSpace(raw)
	if _, ok := allowedExportFontNames[name]; ok {
		return name
	}
	return ""
}

func sanitizePDFHexColorWithDefault(raw string, fallback string) string {
	color := sanitizePDFHexColor(raw)
	if color == "" {
		return fallback
	}
	return color
}

func sanitizeHeadingFont(raw string) string {
	font := strings.ToLower(strings.TrimSpace(raw))
	switch font {
	case "sans", "serif", "mono":
		return font
	default:
		return defaultExportHeadingFont
	}
}

func headingFontFamily(font string) string {
	switch font {
	case "serif":
		return "'Liberation Serif', 'DejaVu Serif', Georgia, serif"
	case "mono":
		return "'Liberation Mono', 'DejaVu Sans Mono', monospace"
	default:
		return "'Liberation Sans', 'DejaVu Sans', sans-serif"
	}
}

func exportFontFamilyFromName(name string, fallback string) string {
	switch name {
	case "Lora":
		return "'Lora', 'Liberation Serif', 'DejaVu Serif', Georgia, serif"
	case "Merriweather":
		return "'Merriweather', 'Lora', 'Liberation Serif', 'DejaVu Serif', Georgia, serif"
	case "Playfair Display":
		return "'Playfair Display', 'Lora', 'Liberation Serif', 'DejaVu Serif', Georgia, serif"
	case "Source Serif 4":
		return "'Source Serif 4', 'Lora', 'Liberation Serif', 'DejaVu Serif', Georgia, serif"
	case "Tangerine":
		return "'Tangerine', 'Lora', 'Liberation Serif', 'DejaVu Serif', serif"
	case "Inter":
		return "'Inter', 'Liberation Sans', 'DejaVu Sans', sans-serif"
	case "Roboto":
		return "'Roboto', 'Inter', 'Liberation Sans', 'DejaVu Sans', sans-serif"
	case "Open Sans":
		return "'Open Sans', 'Inter', 'Liberation Sans', 'DejaVu Sans', sans-serif"
	case "Poppins":
		return "'Poppins', 'Inter', 'Liberation Sans', 'DejaVu Sans', sans-serif"
	case "Exo 2":
		return "'Exo 2', 'Inter', 'Liberation Sans', 'DejaVu Sans', sans-serif"
	case "Ubuntu":
		return "'Ubuntu', 'Inter', 'Liberation Sans', 'DejaVu Sans', sans-serif"
	case "Nunito Sans":
		return "'Nunito Sans', 'Inter', 'Liberation Sans', 'DejaVu Sans', sans-serif"
	case "Raleway":
		return "'Raleway', 'Inter', 'Liberation Sans', 'DejaVu Sans', sans-serif"
	case "Helvetica":
		return "Helvetica, Arial, 'Liberation Sans', 'DejaVu Sans', sans-serif"
	default:
		return fallback
	}
}

func exportPandocFontFromName(name string) string {
	switch name {
	case "Lora":
		return "Lora"
	case "Inter":
		return "Inter"
	case "Helvetica":
		return "Liberation Sans"
	case "Merriweather", "Playfair Display", "Source Serif 4", "Tangerine":
		return "Liberation Serif"
	case "Roboto", "Open Sans", "Poppins", "Exo 2", "Ubuntu", "Nunito Sans", "Raleway":
		return "Liberation Sans"
	default:
		return ""
	}
}

func pandocFontVars(decor pdfPageDecor) (mainfont string, sansfont string, monofont string) {
	if explicitBody := exportPandocFontFromName(decor.BodyFontName); explicitBody != "" {
		mainfont = explicitBody
	}
	if explicitHeading := exportPandocFontFromName(decor.HeadingFontName); explicitHeading != "" {
		sansfont = explicitHeading
	}

	if mainfont == "" || sansfont == "" {
		fallbackMain, fallbackSans, fallbackMono := "", "", ""
		switch decor.HeadingFont {
		case "serif":
			fallbackMain, fallbackSans, fallbackMono = "Liberation Serif", "Liberation Serif", "Liberation Mono"
		case "mono":
			fallbackMain, fallbackSans, fallbackMono = "Liberation Mono", "Liberation Mono", "Liberation Mono"
		default:
			fallbackMain, fallbackSans, fallbackMono = "Liberation Sans", "Liberation Sans", "Liberation Mono"
		}
		if mainfont == "" {
			mainfont = fallbackMain
		}
		if sansfont == "" {
			sansfont = fallbackSans
		}
		monofont = fallbackMono
	} else {
		monofont = "Liberation Mono"
	}

	return mainfont, sansfont, monofont
}

func sanitizePDFDecorAlign(raw string, fallback string) string {
	align := strings.ToLower(strings.TrimSpace(raw))
	switch align {
	case "left", "center", "right":
		return align
	default:
		return fallback
	}
}

func sanitizePDFHexColor(raw string) string {
	color := strings.TrimSpace(raw)
	if pdfHexColorPattern.MatchString(color) {
		return strings.ToLower(color)
	}
	return ""
}

func sanitizePDFDecor(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	var normalized strings.Builder
	normalized.Grow(len(trimmed))
	for _, ch := range trimmed {
		switch ch {
		case '\n', '\r', '\t':
			normalized.WriteRune(' ')
		default:
			if ch >= 32 && ch != 127 {
				normalized.WriteRune(ch)
			}
		}
	}

	cleaned := strings.Join(strings.Fields(normalized.String()), " ")
	if cleaned == "" {
		return ""
	}

	runes := []rune(cleaned)
	if len(runes) > maxPDFDecorLength {
		return string(runes[:maxPDFDecorLength])
	}
	return cleaned
}

func cssContentString(value string) string {
	return `"` + cssContentEscaper.Replace(value) + `"`
}

// pageOverridesCSS generates a <style> block for custom PDF margins/header/footer.
func pageOverridesCSS(m pdfMargins, decor pdfPageDecor) string {
	headingColor := decor.HeadingTextColor
	if headingColor == "" {
		headingColor = defaultExportHeadingTextColor
	}
	h2TextColor := decor.H2TextColor
	if h2TextColor == "" {
		h2TextColor = defaultExportH2TextColor
	}
	h2UnderlineColor := decor.H2UnderlineColor
	if h2UnderlineColor == "" {
		h2UnderlineColor = defaultExportH2UnderlineColor
	}

	headingFont := decor.HeadingFont
	if headingFont == "" {
		headingFont = defaultExportHeadingFont
	}

	overrideMargins := m != marginPresets["standard"]
	overrideHeader := decor.Header != ""
	overrideFooter := decor.Footer != ""
	overrideUnderline := decor.H1UnderlineColor != ""
	overrideHeadingColor := headingColor != defaultExportHeadingTextColor
	overrideH2TextColor := h2TextColor != defaultExportH2TextColor
	overrideH2UnderlineColor := h2UnderlineColor != defaultExportH2UnderlineColor
	overrideHeadingFont := headingFont != defaultExportHeadingFont
	overrideHeadingFontName := decor.HeadingFontName != ""
	overrideBodyFontName := decor.BodyFontName != ""
	overridePageDecor := overrideMargins || overrideHeader || overrideFooter

	if !overridePageDecor && !overrideUnderline && !overrideHeadingColor && !overrideH2TextColor && !overrideH2UnderlineColor && !overrideHeadingFont && !overrideHeadingFontName && !overrideBodyFontName {
		return ""
	}

	var css strings.Builder
	css.WriteString("<style>")

	if overridePageDecor {
		css.WriteString("@page {")
		if overrideMargins {
			fmt.Fprintf(&css, " margin: %s %s %s %s;", m.Top, m.Right, m.Bottom, m.Left)
		}
		if overrideHeader {
			css.WriteString(pageDecorBoxRule("top", decor.HeaderAlign, cssContentString(decor.Header)))
		}
		if overrideFooter {
			css.WriteString(pageDecorBoxRule("bottom", decor.FooterAlign, cssContentString(decor.Footer)))
		}
		css.WriteString(" }")

		if overrideMargins {
			fmt.Fprintf(&css, " @page:first { margin-top: %s; }", m.Top)
		}
	}

	if overrideUnderline {
		fmt.Fprintf(&css, " h1 { border-bottom-color: %s; }", decor.H1UnderlineColor)
	}

	if overrideHeadingColor || overrideHeadingFont || overrideHeadingFontName {
		headingFamily := headingFontFamily(headingFont)
		if overrideHeadingFontName {
			headingFamily = exportFontFamilyFromName(decor.HeadingFontName, headingFamily)
		}

		css.WriteString(" h1, h2, h3, h4, h5, h6 {")
		if overrideHeadingColor {
			fmt.Fprintf(&css, " color: %s;", headingColor)
		}
		if overrideHeadingFont || overrideHeadingFontName {
			fmt.Fprintf(&css, " font-family: %s;", headingFamily)
		}
		css.WriteString(" }")
	}

	if overrideBodyFontName {
		fmt.Fprintf(&css,
			" body, p, li, td, th, blockquote, dd { font-family: %s; }",
			exportFontFamilyFromName(decor.BodyFontName, "'Lora', 'Liberation Serif', 'DejaVu Serif', Georgia, serif"),
		)
	}

	if overrideH2TextColor || overrideH2UnderlineColor {
		css.WriteString(" h2 {")
		if overrideH2TextColor {
			fmt.Fprintf(&css, " color: %s;", h2TextColor)
		}
		if overrideH2UnderlineColor {
			fmt.Fprintf(&css, " border-bottom-color: %s;", h2UnderlineColor)
		}
		css.WriteString(" }")
	}

	css.WriteString("</style>")
	return css.String()
}

// ─── Supported export formats ───────────────────────────────
var pandocFormats = map[string]struct {
	ext         string
	contentType string
	to          string
}{
	"docx":      {".docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "docx"},
	"odt":       {".odt", "application/vnd.oasis.opendocument.text", "odt"},
	"epub":      {".epub", "application/epub+zip", "epub"},
	"rst":       {".rst", "text/plain; charset=utf-8", "rst"},
	"latex":     {".tex", "application/x-tex", "latex"},
	"pdf":       {".pdf", "application/pdf", "pdf"},
	"mediawiki": {".wiki", "text/plain; charset=utf-8", "mediawiki"},
	"asciidoc":  {".adoc", "text/plain; charset=utf-8", "asciidoc"},
	"textile":   {".textile", "text/plain; charset=utf-8", "textile"},
	"jira":      {".jira", "text/plain; charset=utf-8", "jira"},
	"plain":     {".txt", "text/plain; charset=utf-8", "plain"},
}

// POST /api/files/{id}/export/{format}
func (h *exportHandler) export(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	format := strings.ToLower(chi.URLParam(r, "format"))

	fmtInfo, ok := pandocFormats[format]
	if !ok {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("unsupported format: %s", format))
		return
	}

	fwc, err := h.store(r).GetContent(id)
	if err != nil {
		if err == storage.ErrNotFound {
			writeError(w, http.StatusNotFound, "file not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not read file")
		return
	}

	if !h.acquire() {
		writeError(w, http.StatusServiceUnavailable, "export service is at capacity, please retry shortly")
		return
	}
	defer h.release()

	// Write input to a temp file (pandoc reads from stdin or file)
	tmpDir, err := os.MkdirTemp("", "md-export-*")
	if err != nil {
		slog.Error("create tmpdir", "error", err)
		writeError(w, http.StatusInternalServerError, "export failed")
		return
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			slog.Warn("cleanup tmp dir failed", "path", tmpDir, "error", err)
		}
	}()

	inputFile := filepath.Join(tmpDir, "input.md")
	outputFile := filepath.Join(tmpDir, "output"+fmtInfo.ext)

	// Preprocess page breaks for PDF export
	content := preprocessMarkdown(fwc.Content)
	if format == "pdf" {
		content = preprocessMermaidForExport(r.Context(), content, exportEnvInt("MD_MAX_MERMAID_BLOCKS", 50))
		content = preprocessPageBreaks(content)
	}

	if err := os.WriteFile(inputFile, []byte(content), 0600); err != nil {
		writeError(w, http.StatusInternalServerError, "export failed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()

	decor := parsePageDecor(r)

	// PDF: two-step pipeline (Pandoc → HTML → WeasyPrint → PDF)
	if format == "pdf" {
		margins := parseMargins(r)
		htmlFile := filepath.Join(tmpDir, "output.html")
		if err := h.runPDFExport(ctx, inputFile, htmlFile, outputFile, margins, decor); err != nil {
			slog.Error("pdf export failed", "file", fwc.Name, "error", err)
			writeError(w, http.StatusInternalServerError, "export conversion failed")
			return
		}
	} else {
		if err := h.runPandocExport(ctx, inputFile, outputFile, fmtInfo.to, decor); err != nil {
			slog.Error("pandoc export failed", "format", format, "file", fwc.Name, "error", err)
			writeError(w, http.StatusInternalServerError, "export conversion failed")
			return
		}
	}

	if err := streamFile(w, outputFile, fmtInfo.contentType, fwc.Slug+fmtInfo.ext); err != nil {
		slog.Warn("write export response failed", "error", err)
	}
}

// streamFile opens a file and streams it to the response writer.
func streamFile(w http.ResponseWriter, path, contentType, filename string) error {
	f, err := os.Open(path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read export output")
		return err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read export output")
		return err
	}

	safeFilename := strings.NewReplacer(`"`, "", `\`, "", "\r", "", "\n", "").Replace(filename)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, safeFilename))
	w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))
	w.WriteHeader(http.StatusOK)
	_, err = io.Copy(w, f)
	return err
}

// runPandocExport runs Pandoc for non-PDF formats.
func (h *exportHandler) runPandocExport(ctx context.Context, inputFile, outputFile, toFmt string, decor pdfPageDecor) error {
	mainfont, sansfont, monofont := pandocFontVars(decor)
	args := []string{
		// --sandbox blocks pandoc's reader/media IO so malicious markdown
		// (e.g. <img src="file:///run/secrets/...">) cannot make pandoc read
		// local files or fetch remote URLs while embedding media into
		// docx/odt/epub output (LFI + SSRF). Verified blocked on pandoc 3.x.
		"--sandbox",
		"-f", pandocInputFmt,
		"-t", toFmt,
		"--standalone",
		"--highlight-style", "zenburn",
		"-V", "mainfont=" + mainfont,
		"-V", "sansfont=" + sansfont,
		"-V", "monofont=" + monofont,
		"-o", outputFile,
		inputFile,
	}

	if toFmt == "html" || toFmt == "epub" {
		headingFamily := headingFontFamily(decor.HeadingFont)
		if decor.HeadingFontName != "" {
			headingFamily = exportFontFamilyFromName(decor.HeadingFontName, headingFamily)
		}
		bodyFamily := exportFontFamilyFromName(decor.BodyFontName, "'Lora', 'Liberation Serif', 'DejaVu Serif', Georgia, serif")
		h2TextColor := decor.H2TextColor
		if h2TextColor == "" {
			h2TextColor = defaultExportH2TextColor
		}
		h2UnderlineColor := decor.H2UnderlineColor
		if h2UnderlineColor == "" {
			h2UnderlineColor = defaultExportH2UnderlineColor
		}
		headingCSS := fmt.Sprintf("h1,h2,h3,h4,h5,h6{color:%s;font-family:%s;} body,p,li,td,th,blockquote,dd{font-family:%s;} h1{border-bottom-color:%s;} h2{color:%s;border-bottom-color:%s;}", decor.HeadingTextColor, headingFamily, bodyFamily, defaultExportHeadingTextColor, h2TextColor, h2UnderlineColor)
		cssFile := filepath.Join(filepath.Dir(outputFile), "export-heading.css")
		if err := os.WriteFile(cssFile, []byte(headingCSS), 0600); err != nil {
			return fmt.Errorf("write export css: %w", err)
		}
		args = append(args, "--css", cssFile)
	}
	stderr := stderrBufPool.Get().(*bytes.Buffer)
	stderr.Reset()
	defer stderrBufPool.Put(stderr)
	cmd := exec.CommandContext(ctx, h.cfg.PandocBinary, args...)
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w — %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// runPDFExport converts Markdown to PDF via a two-step pipeline:
//
//  1. Pandoc converts Markdown → self-contained HTML5 (print.css embedded
//     inline via --embed-resources, syntax highlighted via zenburn).
//  2. If custom margins/header/footer are requested, a <style> override is
//     injected into the HTML before passing to WeasyPrint.
//  3. WeasyPrint renders the standalone HTML → PDF.
//
// No filename/title metadata is injected; any title in the output comes
// exclusively from the document's own content (YAML frontmatter or headings).
func (h *exportHandler) runPDFExport(ctx context.Context, inputFile, htmlFile, outputFile string, margins pdfMargins, decor pdfPageDecor) error {
	// Step 1: Pandoc → self-contained HTML
	// NOTE: --embed-resources is intentionally omitted. With it, *pandoc*
	// becomes the SSRF egress point (it fetches & inlines remote resources at
	// this stage, bypassing the WeasyPrint url_fetcher guard). Local resources
	// (print.css, bundled fonts via file:// URLs) are loaded by WeasyPrint
	// instead, where _make_fetcher in weasyprint_safe.py enforces the
	// allow-list. See SECURITY.md (2026-06-27).
	pandocArgs := []string{
		// Defense-in-depth: --sandbox blocks any pandoc-side file/URL reads in
		// this stage. print.css is referenced as a <link> (not embedded here)
		// and resolved later by the hardened WeasyPrint fetcher, so the sandbox
		// does not affect styling.
		"--sandbox",
		"-f", pandocInputFmt,
		"-t", "html5",
		"--standalone",
		"--mathml",
		"--highlight-style", "zenburn",
		"--css", "/app/pandoc/print.css",
		"-o", htmlFile,
		inputFile,
	}
	buf1 := stderrBufPool.Get().(*bytes.Buffer)
	buf1.Reset()
	defer stderrBufPool.Put(buf1)
	cmd1 := exec.CommandContext(ctx, h.cfg.PandocBinary, pandocArgs...)
	cmd1.Stderr = buf1
	if err := cmd1.Run(); err != nil {
		return fmt.Errorf("pandoc html stage: %w — %s", err, strings.TrimSpace(buf1.String()))
	}

	// Step 2 (optional): inject page overrides into the HTML
	if overrides := pageOverridesCSS(margins, decor); overrides != "" {
		htmlBytes, err := os.ReadFile(htmlFile)
		if err != nil {
			return fmt.Errorf("read html: %w", err)
		}
		// Inject right before </head>
		modified := strings.Replace(string(htmlBytes), "</head>", overrides+"\n</head>", 1)
		if err := os.WriteFile(htmlFile, []byte(modified), 0600); err != nil {
			return fmt.Errorf("write page override: %w", err)
		}
	}

	// Step 3: WeasyPrint → PDF
	buf2 := stderrBufPool.Get().(*bytes.Buffer)
	buf2.Reset()
	defer stderrBufPool.Put(buf2)
	cmd2 := exec.CommandContext(ctx, h.cfg.WeasyprintBinary, htmlFile, outputFile)
	cmd2.Stderr = buf2
	if err := cmd2.Run(); err != nil {
		return fmt.Errorf("weasyprint stage: %w — %s", err, strings.TrimSpace(buf2.String()))
	}
	return nil
}

// GET /api/export/formats  — list available formats
func (h *exportHandler) listFormats(w http.ResponseWriter, r *http.Request) {
	formats := make([]string, 0, len(pandocFormats))
	for k := range pandocFormats {
		formats = append(formats, k)
	}
	writeJSON(w, http.StatusOK, map[string]any{"formats": formats})
}

// POST /api/export/raw/{format} — export raw markdown content without saving
func (h *exportHandler) exportRaw(w http.ResponseWriter, r *http.Request) {
	format := strings.ToLower(chi.URLParam(r, "format"))

	fmtInfo, ok := pandocFormats[format]
	if !ok {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("unsupported format: %s", format))
		return
	}

	var body struct {
		Content string `json:"content"`
		Name    string `json:"name"`
	}
	if err := decodeJSON(r, &body); err != nil || body.Content == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}
	if body.Name == "" {
		body.Name = "document"
	}

	if !h.acquire() {
		writeError(w, http.StatusServiceUnavailable, "export service is at capacity, please retry shortly")
		return
	}
	defer h.release()

	tmpDir, err := os.MkdirTemp("", "md-export-raw-*")
	if err != nil {
		slog.Error("create tmpdir", "error", err)
		writeError(w, http.StatusInternalServerError, "export failed")
		return
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			slog.Warn("cleanup tmp dir failed", "path", tmpDir, "error", err)
		}
	}()

	inputFile := filepath.Join(tmpDir, "input.md")
	outputFile := filepath.Join(tmpDir, "output"+fmtInfo.ext)

	rawContent := preprocessMarkdown(body.Content)
	if format == "pdf" {
		rawContent = preprocessMermaidForExport(r.Context(), rawContent, exportEnvInt("MD_MAX_MERMAID_BLOCKS", 50))
		rawContent = preprocessPageBreaks(rawContent)
	}

	if err := os.WriteFile(inputFile, []byte(rawContent), 0600); err != nil {
		writeError(w, http.StatusInternalServerError, "export failed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()

	if format == "pdf" {
		margins := parseMargins(r)
		decor := parsePageDecor(r)
		htmlFile := filepath.Join(tmpDir, "output.html")
		if err := h.runPDFExport(ctx, inputFile, htmlFile, outputFile, margins, decor); err != nil {
			slog.Error("pdf export raw failed", "error", err)
			writeError(w, http.StatusInternalServerError, "export conversion failed")
			return
		}
	} else {
		decor := parsePageDecor(r)
		if err := h.runPandocExport(ctx, inputFile, outputFile, fmtInfo.to, decor); err != nil {
			slog.Error("pandoc export raw failed", "format", format, "error", err)
			writeError(w, http.StatusInternalServerError, "export conversion failed")
			return
		}
	}

	slug := strings.TrimSuffix(strings.ReplaceAll(strings.ToLower(body.Name), " ", "-"), ".md")
	if err := streamFile(w, outputFile, fmtInfo.contentType, slug+fmtInfo.ext); err != nil {
		slog.Warn("write export response failed", "error", err)
	}
}
