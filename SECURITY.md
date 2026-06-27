# Security Policy / Politique de sécurité

## Hardening changelog

### 2026-06-27 — Export pipeline & rendering hardening

- **SSRF in the export pipeline (HIGH).** The PDF/HTML export converts
  100% user-controlled Markdown via pandoc + WeasyPrint, which by default
  fetch any referenced resource — turning the (potentially unauthenticated)
  export endpoint into an SSRF primitive (e.g.
  `![](http://169.254.169.254/latest/meta-data/…)` exfiltrated into the PDF)
  and a local-file disclosure primitive (`<img src="file:///etc/passwd">`).
  Mitigations:
  - WeasyPrint now runs through `pandoc/weasyprint_safe.py`, a wrapper whose
    `url_fetcher` allows `data:` URIs, allows `file://` only under an explicit
    allow-list (`/app/pandoc`, system font dirs, the export temp dir), and
    allows `http(s)` only to hosts that resolve exclusively to public IPs —
    private/loopback/link-local/reserved/multicast and cloud-metadata ranges
    are rejected. It is the single, guarded egress for the whole pipeline.
  - `--embed-resources` was removed from the pandoc HTML stage so pandoc is no
    longer a second (unguarded) network egress; local resources are loaded by
    the guarded WeasyPrint instead. Override (not recommended) with
    `MD_WEASYPRINT_BINARY=weasyprint`.
- **Stored/exported XSS (MEDIUM).** Markdown is rendered with goldmark's
  `html.WithUnsafe()`, passing raw user HTML through verbatim. The rendered
  HTML is now run through a bluemonday policy (derived from `UGCPolicy`,
  extended to keep syntax-highlight classes/styles, heading ids, task-list
  checkboxes and the `data-mermaid` hook) before it is cached, exported, or
  served. CSP also hardened with `object-src 'none'`, `base-uri 'self'`,
  `frame-ancestors 'self'`, `form-action 'self'`. (`'unsafe-eval'` is retained
  because the bundled client-side mermaid/katex renderers require the
  `Function` constructor; server-side sanitization is the primary control.)
- **Resource-exhaustion / DoS (MEDIUM).** Each export forks pandoc + WeasyPrint
  and one Chromium per Mermaid block. A bounded semaphore now caps concurrent
  export pipelines (`MD_MAX_CONCURRENT_CONVERSIONS`, default 4; over-capacity
  requests get `503`), Mermaid blocks per document are capped
  (`MD_MAX_MERMAID_BLOCKS`, default 50; excess kept as code fences), and the
  conversion/CRUD API fails closed when no auth is configured unless
  `MD_ALLOW_ANONYMOUS=true` (API key or OIDC otherwise required).
- **Information leak (LOW).** Sub-process (pandoc/WeasyPrint/Chromium) stderr is
  logged via `slog` but no longer returned to the client; callers get a generic
  `export conversion failed` message.
- **Container hardening (LOW).** The `md-api` service in
  `docker-compose.nas.yml` now runs `read_only: true` with a `/tmp` tmpfs for
  the export scratch space, plus `mem_limit`/`cpus`/`pids_limit` ceilings
  (already had `cap_drop: ALL` + `no-new-privileges`). Production CORS is
  pinned to the configured origin (no `*`; `*` remains dev-only).

## Supported Versions / Versions supportées

| Version | Supported |
|---------|-----------|
| latest  | Yes       |

## Reporting a Vulnerability / Signaler une vulnérabilité

**English:**
If you discover a security vulnerability, please report it responsibly.

- **Non-sensitive issues**: Open a [GitHub Issue](https://github.com/cybergraphe-fr/md-app/issues)
- **Sensitive issues**: Contact us privately via GitHub Security Advisories on the [repository](https://github.com/cybergraphe-fr/md-app/security/advisories)

Please include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

We will acknowledge your report within 48 hours and provide a fix as soon as possible.

---

**Français :**
Si vous découvrez une vulnérabilité de sécurité, merci de la signaler de manière responsable.

- **Problèmes non sensibles** : Ouvrez une [Issue GitHub](https://github.com/cybergraphe-fr/md-app/issues)
- **Problèmes sensibles** : Contactez-nous en privé via les GitHub Security Advisories sur le [dépôt](https://github.com/cybergraphe-fr/md-app/security/advisories)

Merci d'inclure :
- Description de la vulnérabilité
- Étapes pour reproduire
- Impact potentiel
- Correction suggérée (le cas échéant)

Nous accuserons réception sous 48 heures et fournirons un correctif dès que possible.
