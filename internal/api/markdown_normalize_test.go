package api

import (
	"fmt"
	"strings"
	"testing"
)

func TestPreprocessMarkdown_InlineHeadings_AllLevels(t *testing.T) {
	for level := 1; level <= 6; level++ {
		hashes := strings.Repeat("#", level)
		in := fmt.Sprintf("Intro paragraph. %s Heading L%d\nMore text", hashes, level)
		out := preprocessMarkdown(in)

		expected := fmt.Sprintf("\n\n%s Heading L%d", hashes, level)
		if !strings.Contains(out, expected) {
			t.Fatalf("inline heading level %d not normalized as expected:\n%s", level, out)
		}
	}
}

func TestPreprocessMarkdown_TightATXHeadings(t *testing.T) {
	in := "##Heading 2\n###Heading 3\nText ##Heading Inline"
	out := preprocessMarkdown(in)

	if !strings.Contains(out, "## Heading 2") {
		t.Fatalf("tight h2 heading not normalized:\n%s", out)
	}
	if !strings.Contains(out, "### Heading 3") {
		t.Fatalf("tight h3 heading not normalized:\n%s", out)
	}
	if !strings.Contains(out, "\n\n## Heading Inline") {
		t.Fatalf("tight inline heading not normalized:\n%s", out)
	}
}

func TestPreprocessMarkdown_UnicodeBullets(t *testing.T) {
	in := "• Parent item\n  ◦ Child A\n  ◦ Child B\nText • Another top item"
	out := preprocessMarkdown(in)

	if !strings.Contains(out, "- Parent item") {
		t.Fatalf("top bullet not normalized:\n%s", out)
	}
	if !strings.Contains(out, "  - Child A") || !strings.Contains(out, "  - Child B") {
		t.Fatalf("sub-bullets not normalized:\n%s", out)
	}
	if !strings.Contains(out, "\n- Another top item") {
		t.Fatalf("inline bullet not normalized:\n%s", out)
	}
}

func TestPreprocessMarkdown_CodeFenceUntouched(t *testing.T) {
	in := "```md\n## not-a-heading\n• not-a-list\n```\nOutside ## real-heading"
	out := preprocessMarkdown(in)

	if !strings.Contains(out, "```md\n## not-a-heading\n• not-a-list\n```") {
		t.Fatalf("code fence content was altered:\n%s", out)
	}
	if !strings.Contains(out, "\n\n## real-heading") {
		t.Fatalf("outside heading was not normalized:\n%s", out)
	}
}

func TestRenderMarkdown_AppliesNormalization(t *testing.T) {
	in := "Text before ## Section\n• Item 1\n• Item 2"
	html, err := renderMarkdown(in)
	if err != nil {
		t.Fatalf("renderMarkdown error: %v", err)
	}

	if !strings.Contains(html, "<h2") {
		t.Fatalf("expected h2 in rendered HTML, got:\n%s", html)
	}
	if strings.Contains(html, "## Section") {
		t.Fatalf("raw markdown heading leaked into HTML:\n%s", html)
	}
	if strings.Count(html, "<li>") < 2 {
		t.Fatalf("expected list items in rendered HTML, got:\n%s", html)
	}
}

func TestPreprocessMarkdown_FlattenedInlineTable(t *testing.T) {
	in := "Hypothèses : Churn faible. | Métrique | Année 1 | Année 2 || —|—|— || CA | 125 k€ | 491 k€ || EBITDA | 63 k€ | 101 k€"
	out := preprocessMarkdown(in)

	if !strings.Contains(out, "| Métrique | Année 1 | Année 2 |") {
		t.Fatalf("table header not normalized:\n%s", out)
	}
	if !strings.Contains(out, "| --- | --- | --- |") {
		t.Fatalf("table separator not normalized:\n%s", out)
	}
	if !strings.Contains(out, "| CA | 125 k€ | 491 k€ |") {
		t.Fatalf("table row CA missing:\n%s", out)
	}
	if !strings.Contains(out, "| EBITDA | 63 k€ | 101 k€ |") {
		t.Fatalf("table row EBITDA missing:\n%s", out)
	}
}

func TestPreprocessMarkdown_FlattenedInlineTable_ScreenshotPattern(t *testing.T) {
	in := "Hypothèses : Churn B2B très faible (<3%). Les achats de packs augmentent avec la maturité de l’étude (1 pack en Y1, 2 en Y2, 3 en Y3, etc.). | Métrique | Année 1 (2026) | Année 2 (2027) | Année 3 (2028) | Année 4 (2029) | Année 5 (2030) || —|—|—|—|—|— || Parc Clients “Pro” | 50 | 200 | 600 | 1 500 | 3 000 || EBITDA (Résultat Brut) | ~63 k€ | ~101 k€ | ~391 k€ | ~1 391 k€ | ~4 005 k€"
	out := preprocessMarkdown(in)

	if !strings.Contains(out, "| Métrique | Année 1 (2026) | Année 2 (2027) | Année 3 (2028) | Année 4 (2029) | Année 5 (2030) |") {
		t.Fatalf("header row not reconstructed:\n%s", out)
	}
	if !strings.Contains(out, "| --- | --- | --- | --- | --- | --- |") {
		t.Fatalf("separator row not reconstructed:\n%s", out)
	}
	if !strings.Contains(out, "| Parc Clients “Pro” | 50 | 200 | 600 | 1 500 | 3 000 |") {
		t.Fatalf("data row not reconstructed:\n%s", out)
	}
}

func TestPreprocessMarkdown_AsteriskBulletsAndInlineSplit(t *testing.T) {
	in := "Le succès repose sur la confiance.\n * Profil du Porteur : Expert cybersécurité\n * Savoir-faire : Gouvernance\nOffre Pro\n * Licences Coffres (Upsell) : * Pack de 10 : 290 € * Pack de 50 : 950 €"
	out := preprocessMarkdown(in)

	if !strings.Contains(out, "\n\n- Profil du Porteur : Expert cybersécurité") {
		t.Fatalf("asterisk list interruption not normalized:\n%s", out)
	}
	if !strings.Contains(out, "\n- Savoir-faire : Gouvernance") {
		t.Fatalf("second asterisk list item missing:\n%s", out)
	}
	if !strings.Contains(out, "\n- Licences Coffres (Upsell) :") ||
		!strings.Contains(out, "\n  - Pack de 10 : 290 €") ||
		!strings.Contains(out, "\n  - Pack de 50 : 950 €") {
		t.Fatalf("inline asterisk bullets not split:\n%s", out)
	}
}

func TestRenderMarkdown_AsteriskBulletsBecomeList(t *testing.T) {
	in := "Paragraphe\n * Élément A\n * Élément B"
	html, err := renderMarkdown(in)
	if err != nil {
		t.Fatalf("renderMarkdown error: %v", err)
	}

	if strings.Count(html, "<li>") < 2 {
		t.Fatalf("expected list items from asterisk bullets, got:\n%s", html)
	}
}

func TestRenderMarkdown_PreservesSimpleLineBreaks(t *testing.T) {
	in := "**Entreprise :** Cybergraphe\n**Secteur :** LegalTech\n**Positionnement :** ZKG"
	html, err := renderMarkdown(in)
	if err != nil {
		t.Fatalf("renderMarkdown error: %v", err)
	}

	if !strings.Contains(html, "<br") {
		t.Fatalf("expected hard line breaks in rendered HTML, got:\n%s", html)
	}
	if strings.Count(html, "<strong>") < 3 {
		t.Fatalf("expected bold labels to remain intact, got:\n%s", html)
	}
}

func TestPreprocessMarkdown_StandaloneLeadInBeforeList(t *testing.T) {
	in := "- Parent\nOffre Enterprise\n- Setup sur-mesure"
	out := preprocessMarkdown(in)

	if !strings.Contains(out, "- Parent\n\nOffre Enterprise\n\n- Setup sur-mesure") {
		t.Fatalf("standalone lead-in was not isolated from list blocks:\n%s", out)
	}
}

func TestRenderMarkdown_ReplacesMermaidFenceWithRenderableBlock(t *testing.T) {
	in := "```mermaid\ngraph TD\nA-->B\n```"
	html, err := renderMarkdown(in)
	if err != nil {
		t.Fatalf("renderMarkdown error: %v", err)
	}

	if !strings.Contains(html, `class="mermaid-block"`) || !strings.Contains(html, `data-mermaid="true"`) {
		t.Fatalf("expected Mermaid placeholder block, got:\n%s", html)
	}
	if strings.Contains(html, "<code") {
		t.Fatalf("expected Mermaid fence to bypass regular code rendering, got:\n%s", html)
	}
	if !strings.Contains(html, "graph TD") {
		t.Fatalf("expected Mermaid source to remain in placeholder, got:\n%s", html)
	}
}

func TestRenderMarkdown_MermaidPreservesHtmlLikeLabelsAsText(t *testing.T) {
	in := "```mermaid\ngraph TD\nA -->|foo<br/>bar| B\n```"
	html, err := renderMarkdown(in)
	if err != nil {
		t.Fatalf("renderMarkdown error: %v", err)
	}

	if !strings.Contains(html, "foo&lt;br/&gt;bar") {
		t.Fatalf("expected Mermaid HTML-like labels to stay escaped in source block, got:\n%s", html)
	}
}
