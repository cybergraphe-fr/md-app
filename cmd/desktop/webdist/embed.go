//go:build desktop

// Package webdist embeds the built SvelteKit frontend so the desktop
// binary is fully self-contained — no external web/ directory needed.
//
// Before compiling with -tags desktop, copy web/dist/ into this directory:
//
//	cp -r web/dist cmd/desktop/webdist/dist
//
// The dist/ directory is gitignored; it only exists during builds.
package webdist

import "embed"

//go:embed all:dist
var Assets embed.FS
