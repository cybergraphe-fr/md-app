# Contributing / Contribuer

## English

### Prerequisites

- Go 1.25+
- Node.js 22+
- Docker & Docker Compose
- [lefthook](https://github.com/evilmartians/lefthook) (git hooks)
- [golangci-lint](https://golangci-lint.run/) v2+

### Setup

```bash
git clone https://github.com/cybergraphe-fr/md-app.git
cd md-app
lefthook install

# Backend
go run ./cmd/server

# Frontend (separate terminal)
cd web && npm ci && npm run dev
```

### Development Workflow

1. Fork the repository
2. Create a branch from `main` (`feature/my-feature` or `fix/my-fix`)
3. Make your changes
4. Run quality checks: `make ci`
5. Commit using [Conventional Commits](https://www.conventionalcommits.org/)
6. Open a Pull Request

### Conventional Commits

```
feat(api): add batch export endpoint
fix(editor): resolve cursor jump on save
docs: update API documentation
chore(deps): bump goldmark to v1.8
```

### Architecture

```
cmd/server/         — Entry point (main.go)
internal/
  api/              — HTTP handlers (chi router)
  config/           — Environment-based configuration
  storage/          — File system operations
  cache/            — Redis wrapper
  plugins/          — Plugin registry
  webhooks/         — Webhook manager
web/
  src/lib/
    components/     — Svelte 5 components
    stores/         — Svelte stores
    api.ts          — Typed API client
pandoc/             — Export stylesheets
```

### Code Style

- **Go**: `go vet`, `golangci-lint run`, race detector (`go test -race`)
- **Frontend**: TypeScript strict mode, `npm run check`, TailwindCSS utilities
- **No unnecessary comments** on unchanged code

---

## Français

### Prérequis

- Go 1.25+
- Node.js 22+
- Docker & Docker Compose
- [lefthook](https://github.com/evilmartians/lefthook) (hooks git)
- [golangci-lint](https://golangci-lint.run/) v2+

### Installation

```bash
git clone https://github.com/cybergraphe-fr/md-app.git
cd md-app
lefthook install

# Backend
go run ./cmd/server

# Frontend (terminal séparé)
cd web && npm ci && npm run dev
```

### Workflow

1. Forkez le dépôt
2. Créez une branche depuis `main` (`feature/ma-feature` ou `fix/mon-fix`)
3. Faites vos modifications
4. Lancez les vérifications : `make ci`
5. Committez en [Conventional Commits](https://www.conventionalcommits.org/)
6. Ouvrez une Pull Request

### Style de code

- **Go** : `go vet`, `golangci-lint run`, race detector (`go test -race`)
- **Frontend** : TypeScript strict, `npm run check`, utilitaires TailwindCSS
- **Pas de commentaires inutiles** sur le code non modifié
