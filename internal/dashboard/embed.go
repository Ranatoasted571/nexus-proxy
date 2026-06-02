package dashboard

import (
	"embed"
	"io/fs"
)

// distFS holds the built dashboard UI, embedded into the binary at compile time.
// During development the committed dist/index.html is a self-contained fallback;
// `make build-web && make embed` overwrites dist/ with the full Svelte build.
//
//go:embed all:dist
var distFS embed.FS

// distFileSystem returns the embedded dashboard files rooted at dist/.
func distFileSystem() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}
