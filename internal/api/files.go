package api

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"go.abhg.dev/goldmark/frontmatter"

	"md/internal/cache"
	"md/internal/storage"
)

// markdown engine singleton
var md = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		extension.DefinitionList,
		extension.Footnote,
		extension.Typographer,
		emoji.Emoji,
		&frontmatter.Extender{},
		highlighting.NewHighlighting(
			highlighting.WithStyle("github"),
			highlighting.WithGuessLanguage(true),
		),
	),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
	),
	goldmark.WithRendererOptions(
		html.WithHardWraps(),
		html.WithXHTML(),
		html.WithUnsafe(), // allow raw HTML in markdown
	),
)

const renderCacheVersion = "v2"

// bufPool reuses bytes.Buffer for markdown rendering.
var bufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

// renderMarkdown converts markdown content to an HTML string.
func renderMarkdown(content string) (string, error) {
	content = preprocessMarkdown(content)
	content = replaceMermaidFences(content)
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)
	if err := md.Convert([]byte(content), buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// processMermaidFences parses mermaid code fences and calls handler for each
// block's source content.  The handler's return value replaces the fence in
// the output.  Non-mermaid content is passed through unchanged.
func processMermaidFences(content string, handler func(source string) string) string {
	if !strings.Contains(strings.ToLower(content), "mermaid") {
		return content
	}

	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	var blockLines []string
	inMermaid := false
	var fenceChar byte
	var fenceLen int

	parseFence := func(line string) (bool, byte, int, string) {
		trimmedLeft := strings.TrimLeft(line, " ")
		indent := len(line) - len(trimmedLeft)
		if indent > 3 || len(trimmedLeft) < 3 {
			return false, 0, 0, ""
		}
		if !strings.HasPrefix(trimmedLeft, "```") && !strings.HasPrefix(trimmedLeft, "~~~") {
			return false, 0, 0, ""
		}
		ch := trimmedLeft[0]
		count := 0
		for count < len(trimmedLeft) && trimmedLeft[count] == ch {
			count++
		}
		if count < 3 {
			return false, 0, 0, ""
		}
		rest := strings.TrimSpace(trimmedLeft[count:])
		if rest == "" {
			return true, ch, count, ""
		}
		return true, ch, count, strings.ToLower(strings.Fields(rest)[0])
	}

	isClosingFence := func(line string, ch byte, minLen int) bool {
		trimmedLeft := strings.TrimLeft(line, " ")
		indent := len(line) - len(trimmedLeft)
		if indent > 3 || len(trimmedLeft) < minLen {
			return false
		}
		count := 0
		for count < len(trimmedLeft) && trimmedLeft[count] == ch {
			count++
		}
		if count < minLen {
			return false
		}
		return strings.TrimSpace(trimmedLeft[count:]) == ""
	}

	flushMermaid := func() {
		out = append(out, handler(strings.Join(blockLines, "\n")))
		blockLines = nil
	}

	for _, line := range lines {
		if inMermaid {
			if isClosingFence(line, fenceChar, fenceLen) {
				flushMermaid()
				inMermaid = false
				continue
			}
			blockLines = append(blockLines, line)
			continue
		}

		if ok, ch, count, lang := parseFence(line); ok && lang == "mermaid" {
			inMermaid = true
			fenceChar = ch
			fenceLen = count
			blockLines = nil
			continue
		}

		out = append(out, line)
	}

	if inMermaid {
		flushMermaid()
	}

	return strings.Join(out, "\n")
}

func replaceMermaidFences(content string) string {
	return processMermaidFences(content, func(source string) string {
		return `<pre class="mermaid-block" data-mermaid="true">` + template.HTMLEscapeString(source) + `</pre>`
	})
}

// ---- Handlers ----

type filesHandler struct {
	basePath string
	cache    *cache.Client
}

func newFilesHandler(basePath string, c *cache.Client) *filesHandler {
	return &filesHandler{basePath: basePath, cache: c}
}

// store returns a workspace-scoped storage for the current request.
func (h *filesHandler) store(r *http.Request) *storage.Storage {
	return ScopedStorage(h.basePath, r)
}

// GET /api/files
func (h *filesHandler) list(w http.ResponseWriter, r *http.Request) {
	files, err := h.store(r).List()
	if err != nil {
		slog.Error("list files", "error", err)
		writeError(w, http.StatusInternalServerError, "could not list files")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"files": files, "count": len(files)})
}

// POST /api/files
func (h *filesHandler) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name    string `json:"name"`
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		body.Name = "untitled"
	}
	f, err := h.store(r).Create(body.Name, body.Path, body.Content)
	if err != nil {
		slog.Error("create file", "error", err)
		writeError(w, http.StatusInternalServerError, "could not create file")
		return
	}
	writeJSON(w, http.StatusCreated, f)
}

// GET /api/files/{id}
func (h *filesHandler) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	fwc, err := h.store(r).GetContent(id)
	if err != nil {
		if err == storage.ErrNotFound {
			writeError(w, http.StatusNotFound, "file not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not read file")
		return
	}
	writeJSON(w, http.StatusOK, fwc)
}

// PUT /api/files/{id}
func (h *filesHandler) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Name    string `json:"name"`
		Content string `json:"content"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Auto-version: save current content before overwriting.
	st := h.store(r)
	if current, err := st.GetContent(id); err == nil {
		_, _ = st.SaveVersion(id, current.Content, "auto-save")
	}

	f, err := st.Update(id, body.Name, body.Content)
	if err != nil {
		if err == storage.ErrNotFound {
			writeError(w, http.StatusNotFound, "file not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not update file")
		return
	}
	// Invalidate cache
	if h.cache != nil {
		_ = h.cache.Delete(r.Context(), "render:"+id)
		_ = h.cache.Delete(r.Context(), "render:"+renderCacheVersion+":"+id)
	}
	writeJSON(w, http.StatusOK, f)
}

// DELETE /api/files/{id}
func (h *filesHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.store(r).Delete(id); err != nil {
		if err == storage.ErrNotFound {
			writeError(w, http.StatusNotFound, "file not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not delete file")
		return
	}
	if h.cache != nil {
		_ = h.cache.Delete(r.Context(), "render:"+id)
		_ = h.cache.Delete(r.Context(), "render:"+renderCacheVersion+":"+id)
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/files/{id}/render
// Returns rendered HTML (from cache if available).
func (h *filesHandler) render(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	cacheKey := "render:" + renderCacheVersion + ":" + id

	// Check cache
	if h.cache != nil {
		cached, err := h.cache.Get(r.Context(), cacheKey)
		if err == nil && cached != "" {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("X-Cache", "HIT")
			fmt.Fprint(w, cached)
			return
		}
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

	rendered, err := renderMarkdown(fwc.Content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "render error")
		return
	}

	result := map[string]string{"html": rendered, "name": fwc.Name}
	if h.cache != nil {
		if b, err := marshalJSON(result); err == nil {
			if err := h.cache.Set(context.Background(), cacheKey, string(b)); err != nil {
				slog.Warn("cache set failed", "key", cacheKey, "error", err)
			}
		}
	}

	w.Header().Set("X-Cache", "MISS")
	writeJSON(w, http.StatusOK, result)
}

// POST /api/files/render  (ad-hoc render without saving)
func (h *filesHandler) renderRaw(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Content string `json:"content"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	rendered, err := renderMarkdown(body.Content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "render error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"html": rendered})
}

// POST /api/files/import  (multipart form upload)
func (h *filesHandler) importFile(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "bad multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer file.Close()

	name := strings.TrimSuffix(header.Filename, ".md")
	name = strings.TrimSuffix(name, ".txt")
	name = strings.TrimSuffix(name, ".html")

	f, err := h.store(r).ImportReader(name, file)
	if err != nil {
		slog.Error("import file", "error", err)
		writeError(w, http.StatusInternalServerError, "could not import file")
		return
	}
	writeJSON(w, http.StatusCreated, f)
}

// ---- Full-page HTML export template ----
var htmlExportTmpl = template.Must(template.New("export").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{.Title}}</title>
<style>
  :root { --font-body: 'Georgia', serif; --font-mono: 'JetBrains Mono', 'Fira Code', monospace; }
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: var(--font-body); font-size: 18px; line-height: 1.75;
         color: #1a1a1a; background: #fff; max-width: 780px; margin: 0 auto;
         padding: 3rem 2rem; }
  h1,h2,h3,h4,h5,h6 { font-weight: 600; margin: 2rem 0 0.75rem; line-height: 1.3; }
  h1 { font-size: 2.2rem; border-bottom: 2px solid #e5e7eb; padding-bottom: 0.5rem; }
  h2 { font-size: 1.7rem; } h3 { font-size: 1.35rem; }
  p { margin: 1rem 0; }
  a { color: #2563eb; text-decoration: underline; }
  code { font-family: var(--font-mono); font-size: 0.875em; background: #f3f4f6;
         padding: 0.15em 0.4em; border-radius: 4px; }
  pre { background: #1e1e2e; color: #cdd6f4; padding: 1.25rem; border-radius: 8px;
        overflow-x: auto; margin: 1.5rem 0; font-size: 0.875rem; line-height: 1.6; }
  pre code { background: none; padding: 0; color: inherit; }
  blockquote { border-left: 4px solid #93c5fd; margin: 1.5rem 0;
               padding: 0.75rem 1.25rem; background: #eff6ff;
               color: #1e40af; border-radius: 0 6px 6px 0; }
  table { border-collapse: collapse; width: 100%; margin: 1.5rem 0; }
  th, td { border: 1px solid #e5e7eb; padding: 0.6rem 1rem; text-align: left; }
  th { background: #f9fafb; font-weight: 600; }
  tr:nth-child(even) { background: #f9fafb; }
  ul, ol { margin: 1rem 0 1rem 1.75rem; }
  li { margin: 0.35rem 0; }
  img { max-width: 100%; height: auto; border-radius: 6px; margin: 1rem 0; }
  hr { border: none; border-top: 1px solid #e5e7eb; margin: 2rem 0; }
  .task-list-item { list-style: none; margin-left: -1.75rem; padding-left: 1.75rem; }
	.mermaid-block { background: #0f172a; color: #e2e8f0; padding: 1rem; border-radius: 8px; overflow-x: auto; white-space: pre; }
	.mermaid-diagram { border: 1px solid #e5e7eb; border-radius: 8px; background: #f8fafc; margin: 1.5rem 0; padding: 1rem; overflow-x: auto; }
	.mermaid-diagram svg { max-width: 100%; height: auto; }
	.mermaid-error { border: 1px solid #dc2626; color: #dc2626; border-radius: 8px; padding: 0.75rem; }
  @media print { body { padding: 0; max-width: none; }
                 pre { break-inside: avoid; } }
</style>
</head>
<body>
{{.Body}}
<script type="module">
	const blocks = Array.from(document.querySelectorAll('pre[data-mermaid]'));
	if (blocks.length > 0) {
		try {
			const mermaid = (await import('https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs')).default;
			mermaid.initialize({
				startOnLoad: false,
				theme: 'default',
				securityLevel: 'strict',
				flowchart: { htmlLabels: false },
			});
			let counter = 0;
			for (const block of blocks) {
				const src = block.textContent || '';
				try {
					counter += 1;
					  const { svg } = await mermaid.render('export-mermaid-' + counter, src);
					const div = document.createElement('div');
					div.className = 'mermaid-diagram';
					div.innerHTML = svg;
					block.replaceWith(div);
				} catch (error) {
					block.classList.add('mermaid-error');
					console.error('Mermaid export render failed', error);
				}
			}
		} catch (error) {
			console.error('Mermaid export bootstrap failed', error);
		}
	}
</script>
</body>
</html>`))

// GET /api/files/{id}/export/html
func (h *filesHandler) exportHTML(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	fwc, err := h.store(r).GetContent(id)
	if err != nil {
		if err == storage.ErrNotFound {
			writeError(w, http.StatusNotFound, "file not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not read file")
		return
	}

	rendered, err := renderMarkdown(fwc.Content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "render error")
		return
	}

	var buf bytes.Buffer
	if err := htmlExportTmpl.Execute(&buf, map[string]any{
		"Title": fwc.Name,
		"Body":  template.HTML(rendered), //nolint:gosec
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "template error")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.html"`, fwc.Slug))
	w.Write(buf.Bytes())
}
