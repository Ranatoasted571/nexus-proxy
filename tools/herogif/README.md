# herogif — landing/README hero generator

Renders **`docs/dashboard.gif`**: a deterministic, animated mock of the live NEXUS
dashboard — the savings banner, the six stat cards (including the privacy
**"secrets masked · 0 leaked"** card), the **routing-mix** bar with its legend, and
the live request feed. Output is **900×560, 30 frames**.

This is an **isolated Go module** (it has its own `go.mod`), so it is excluded from
the main module's `go build/test/vet ./...` and never affects the binary or CI.

## Run

```bash
# Windows — uses Segoe UI + Consolas from C:\Windows\Fonts by default
go run . ../../docs/dashboard.gif

# macOS / Linux — point NEXUS_FONT_DIR at a folder containing:
#   seguibl.ttf seguisb.ttf segoeui.ttf consola.ttf consolab.ttf
NEXUS_FONT_DIR=/path/to/fonts go run . ../../docs/dashboard.gif
```

Optional extra args dump verification PNGs:
`go run . out.gif final.png mid.png`.

Edit the numbers/feed/`mix` slices in `main.go` to change what the hero shows.
