package main

import (
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const (
	W = 900
	H = 560
	N = 30
)

// ── design tokens ──
var (
	cBg      = rgb(5, 8, 22)
	cSurface = rgb(10, 14, 26)
	cSurf2   = rgb(17, 24, 47)
	cBorder  = rgb(26, 32, 53)
	cAccent  = rgb(124, 58, 237)
	cCyan    = rgb(6, 182, 212)
	cGreen   = rgb(16, 185, 129)
	cAmber   = rgb(245, 158, 11)
	cRed     = rgb(239, 68, 68)
	cText    = rgb(226, 232, 240)
	cMuted   = rgb(100, 116, 139)
	cWhite   = rgb(255, 255, 255)
)

func rgb(r, g, b uint8) color.RGBA  { return color.RGBA{r, g, b, 255} }
func a(c color.RGBA, al float64) color.RGBA {
	return color.RGBA{c.R, c.G, c.B, uint8(clamp(al, 0, 1) * 255)}
}

// blend c (with its alpha) over the existing dst pixel.
func blendPix(dst *image.RGBA, x, y int, c color.RGBA) {
	if x < 0 || y < 0 || x >= W || y >= H || c.A == 0 {
		return
	}
	al := float64(c.A) / 255
	o := dst.RGBAAt(x, y)
	dst.SetRGBA(x, y, color.RGBA{
		uint8(float64(c.R)*al + float64(o.R)*(1-al)),
		uint8(float64(c.G)*al + float64(o.G)*(1-al)),
		uint8(float64(c.B)*al + float64(o.B)*(1-al)),
		255,
	})
}

// inRounded reports whether pixel (i,j) is inside a w×h rect with corner radius r.
func inRounded(i, j, w, h, r int) bool {
	if r <= 0 {
		return true
	}
	cx, cy := i, j
	var dx, dy int = -1, -1
	if i < r {
		dx = r - i
	} else if i >= w-r {
		dx = i - (w - r - 1)
	}
	if j < r {
		dy = r - j
	} else if j >= h-r {
		dy = j - (h - r - 1)
	}
	_ = cx
	_ = cy
	if dx > 0 && dy > 0 {
		return dx*dx+dy*dy <= r*r
	}
	return true
}

func fillRoundRect(dst *image.RGBA, x, y, w, h, r int, c color.RGBA) {
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			if inRounded(i, j, w, h, r) {
				blendPix(dst, x+i, y+j, c)
			}
		}
	}
}

func strokeRoundRect(dst *image.RGBA, x, y, w, h, r int, c color.RGBA) {
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			if !inRounded(i, j, w, h, r) {
				continue
			}
			edge := i == 0 || j == 0 || i == w-1 || j == h-1 ||
				!inRounded(i-1, j, w, h, r) || !inRounded(i+1, j, w, h, r) ||
				!inRounded(i, j-1, w, h, r) || !inRounded(i, j+1, w, h, r)
			if edge {
				blendPix(dst, x+i, y+j, c)
			}
		}
	}
}

func disc(dst *image.RGBA, cx, cy, r int, c color.RGBA) {
	for j := -r; j <= r; j++ {
		for i := -r; i <= r; i++ {
			if i*i+j*j <= r*r {
				blendPix(dst, cx+i, cy+j, c)
			}
		}
	}
}

// ── fonts ──
type faces struct{ black, semib, body, mono, monob, bodyBig, monoBig font.Face }

func face(path string, px float64) font.Face {
	b, err := os.ReadFile(path)
	must(err)
	f, err := opentype.Parse(b)
	must(err)
	fc, err := opentype.NewFace(f, &opentype.FaceOptions{Size: px, DPI: 72, Hinting: font.HintingFull})
	must(err)
	return fc
}

func loadFaces() faces {
	dir := os.Getenv("NEXUS_FONT_DIR")
	if dir == "" {
		dir = `C:\Windows\Fonts\` // Segoe UI + Consolas (Windows default)
	}
	j := func(name string) string { return filepath.Join(dir, name) }
	return faces{
		black:   face(j("seguibl.ttf"), 30),
		semib:   face(j("seguisb.ttf"), 14),
		body:    face(j("segoeui.ttf"), 11),
		bodyBig: face(j("seguisb.ttf"), 12),
		mono:    face(j("consola.ttf"), 11),
		monob:   face(j("consolab.ttf"), 19),
		monoBig: face(j("consolab.ttf"), 22),
	}
}

func text(dst *image.RGBA, f font.Face, x, base int, s string, c color.RGBA) int {
	d := &font.Drawer{Dst: dst, Src: image.NewUniform(c), Face: f, Dot: fixed.P(x, base)}
	d.DrawString(s)
	return d.Dot.X.Round() - x
}

func measure(f font.Face, s string) int {
	d := &font.Drawer{Face: f}
	return d.MeasureString(s).Round()
}

func textRight(dst *image.RGBA, f font.Face, xr, base int, s string, c color.RGBA) {
	text(dst, f, xr-measure(f, s), base, s, c)
}

// ── helpers ──
func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
func easeOut(t float64) float64 { t = clamp(t, 0, 1); return 1 - math.Pow(1-t, 3) }
func ramp(frame, start, dur int) float64 {
	return easeOut(float64(frame-start) / float64(dur))
}
func comma(n int) string {
	s := fmt.Sprintf("%d", n)
	if n < 1000 {
		return s
	}
	out := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out += ","
		}
		out += string(c)
	}
	return out
}
func must(err error) {
	if err != nil {
		panic(err)
	}
}

// ── feed data ──
type row struct {
	prov, cx string
	cxCol    color.RGBA
	tokens   string
	cost     string
	lat      string
}

var feed = []row{
	{"groq", "SIMPLE", cGreen, "150t", "$0.00000", "320ms"},
	{"deepseek", "STANDARD", cCyan, "2300t", "$0.00210", "1100ms"},
	{"anthropic", "COMPLEX", cAccent, "7600t", "$0.05400", "2600ms"},
	{"venice", "STANDARD", cCyan, "2700t", "$0.00310", "1210ms"},
	{"groq", "SIMPLE", cGreen, "110t", "$0.00000", "280ms"},
}

// routing mix segments (last 200 window)
type seg struct {
	label string
	n     int
	col   color.RGBA
}

var mix = []seg{{"Simple", 92, cGreen}, {"Standard", 86, cCyan}, {"Complex", 18, cAccent}, {"Critical", 4, cRed}}

func renderFrame(fr int, fc faces) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, W, H))
	// background
	for i := range dst.Pix {
		dst.Pix[i] = 0
	}
	fillRoundRect(dst, 0, 0, W, H, 0, cBg)

	p := ramp(fr, 2, 18) // master count-up progress

	// ── header ──
	x := 24
	x += text(dst, fc.black, x, 50, "NE", cAccent)
	x += text(dst, fc.black, x, 50, "X", cCyan)
	text(dst, fc.black, x, 50, "US", cAccent)
	// LIVE
	pulse := 0.5 + 0.5*math.Sin(2*math.Pi*float64(fr)/15)
	disc(dst, 832, 42, 4, a(cGreen, 0.4+0.6*pulse))
	textRight(dst, fc.body, 876, 47, "LIVE", cGreen)

	// ── savings banner ──
	fillRoundRect(dst, 24, 70, 852, 40, 9, blend(cBg, cAccent, 0.16))
	strokeRoundRect(dst, 24, 70, 852, 40, 9, rgb(42, 33, 80))
	saved := 233.0 * p
	pct := int(97 * p)
	bx := 42
	bx += text(dst, fc.semib, bx, 95, "Saved ", cText)
	bx += text(dst, fc.semib, bx, 95, fmt.Sprintf("$%.2f", saved), cGreen)
	bx += text(dst, fc.semib, bx, 95, " this month — ", cText)
	bx += text(dst, fc.semib, bx, 95, fmt.Sprintf("%d%%", pct), cGreen)
	text(dst, fc.semib, bx, 95, " cheaper than Claude", cText)

	// ── stat cards ──
	type card struct {
		val   string
		col   color.RGBA
		label string
		lock  bool
	}
	cards := []card{
		{comma(int(342 * p)), cAccent, "REQUESTS TODAY", false},
		{fmt.Sprintf("$%.2f", 0.21*p), cGreen, "COST TODAY", false},
		{fmt.Sprintf("$%.4f", 0.06*p), cCyan, "CACHE SAVED", false},
		{fmt.Sprintf("$%.2f", 6.40*p), cText, "FORECAST / MO", false},
		{fmt.Sprintf("%dms", int(1124*p)), cText, "AVG LATENCY", false},
		{comma(int(18 * p)), cAmber, "SECRETS MASKED", true},
	}
	cardW, gap, cy := 133, 10, 124
	for i, c := range cards {
		cx := 24 + i*(cardW+gap)
		fillRoundRect(dst, cx, cy, cardW, 72, 8, cSurface)
		strokeRoundRect(dst, cx, cy, cardW, 72, 8, cBorder)
		vx := cx + 14
		if c.lock {
			drawLock(dst, vx, cy+18, cAmber)
			vx += 18
		}
		text(dst, fc.monob, vx, cy+36, c.val, c.col)
		text(dst, fc.body, cx+14, cy+58, c.label, cMuted)
	}

	// ── routing mix ──
	rmY := 208
	fillRoundRect(dst, 24, rmY, 852, 70, 8, cSurface)
	strokeRoundRect(dst, 24, rmY, 852, 70, 8, cBorder)
	text(dst, fc.body, 40, rmY+20, "ROUTING MIX · LAST 200 REQUESTS", cMuted)
	barX, barY, barW, barH, br := 40, rmY+28, 820, 12, 6
	fillRoundRect(dst, barX, barY, barW, barH, br, cSurf2)
	total := 0
	for _, s := range mix {
		total += s.n
	}
	filled := int(float64(barW) * ramp(fr, 4, 16))
	off := 0
	for _, s := range mix {
		segW := int(float64(barW) * float64(s.n) / float64(total))
		for j := 0; j < barH; j++ {
			for i := 0; i < segW; i++ {
				gx := off + i
				if gx >= filled {
					continue
				}
				if inRounded(gx, j, barW, barH, br) {
					blendPix(dst, barX+gx, barY+j, s.col)
				}
			}
		}
		off += segW
	}
	// legend
	lx := 40
	for _, s := range mix {
		disc(dst, lx+4, rmY+54, 4, s.col)
		lx += 14
		lx += text(dst, fc.body, lx, rmY+58, s.label+" ", cText)
		lx += text(dst, fc.bodyBig, lx, rmY+58, comma(s.n), cWhite)
		lx += 22
	}

	// ── feed ──
	text(dst, fc.body, 24, 300, "LIVE REQUEST FEED", cMuted)
	for i, r := range feed {
		al := clamp(float64(fr-(6+i*2))/3.0, 0, 1)
		if al <= 0 {
			continue
		}
		ry := 310 + i*44
		dxoff := int((1 - al) * 12)
		rx := 24 + dxoff
		fillRoundRect(dst, rx, ry, 852, 36, 7, a(cSurface, al))
		strokeRoundRect(dst, rx, ry, 852, 36, 7, a(cBorder, al))
		b := ry + 23
		text(dst, fc.semib, rx+16, b, r.prov, a(cCyan, al))
		text(dst, fc.body, rx+130, b, "claude-sonnet-4-6", a(cMuted, al))
		// badge
		bw := measure(fc.body, r.cx) + 18
		fillRoundRect(dst, rx+300, ry+10, bw, 17, 4, a(blend(cSurface, r.cxCol, 0.16), al))
		text(dst, fc.body, rx+309, ry+22, r.cx, a(r.cxCol, al))
		textRight(dst, fc.mono, rx+560, b, r.tokens, a(cText, al))
		textRight(dst, fc.mono, rx+670, b, r.cost, a(cGreen, al))
		textRight(dst, fc.mono, rx+760, b, r.lat, a(cMuted, al))
		textRight(dst, fc.mono, rx+836, b, "200", a(cMuted, al))
	}
	return dst
}

// drawLock renders a tiny padlock at (x, top) ~14px tall.
func drawLock(dst *image.RGBA, x, top int, c color.RGBA) {
	// shackle
	for ang := 180; ang <= 360; ang++ {
		rad := float64(ang) * math.Pi / 180
		px := x + 6 + int(4*math.Cos(rad))
		py := top + 5 + int(4*math.Sin(rad))
		blendPix(dst, px, py, c)
		blendPix(dst, px, py+1, c)
	}
	// body
	fillRoundRect(dst, x, top+5, 12, 9, 2, c)
}

func blend(base, top color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		uint8(float64(base.R)*(1-t) + float64(top.R)*t),
		uint8(float64(base.G)*(1-t) + float64(top.G)*t),
		uint8(float64(base.B)*(1-t) + float64(top.B)*t),
		255,
	}
}

func main() {
	fc := loadFaces()
	frames := make([]*image.RGBA, N)
	for i := 0; i < N; i++ {
		frames[i] = renderFrame(i, fc)
	}

	// popularity palette across all frames (flat UI → near-lossless).
	freq := map[color.RGBA]int{}
	for _, f := range frames {
		for i := 0; i < len(f.Pix); i += 4 {
			freq[color.RGBA{f.Pix[i], f.Pix[i+1], f.Pix[i+2], 255}]++
		}
	}
	type cc struct {
		c color.RGBA
		n int
	}
	list := make([]cc, 0, len(freq))
	for c, n := range freq {
		list = append(list, cc{c, n})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].n > list[j].n })
	pal := color.Palette{}
	for i := 0; i < len(list) && i < 256; i++ {
		pal = append(pal, list[i].c)
	}

	g := &gif.GIF{LoopCount: 0}
	for i, f := range frames {
		pi := image.NewPaletted(f.Bounds(), pal)
		for y := 0; y < H; y++ {
			for x := 0; x < W; x++ {
				pi.Set(x, y, f.RGBAAt(x, y))
			}
		}
		g.Image = append(g.Image, pi)
		d := 9
		if i >= N-1 {
			d = 160 // hold final frame
		}
		g.Delay = append(g.Delay, d)
	}
	out, err := os.Create(os.Args[1])
	must(err)
	must(gif.EncodeAll(out, g))
	must(out.Close())
	fmt.Println("wrote", os.Args[1], "colors:", len(pal))

	// also dump a verification PNG of a full-state frame
	if len(os.Args) > 2 {
		pf, _ := os.Create(os.Args[2])
		_ = png.Encode(pf, frames[N-1])
		_ = pf.Close()
	}
	if len(os.Args) > 3 {
		pf, _ := os.Create(os.Args[3])
		_ = png.Encode(pf, frames[11]) // mid-animation
		_ = pf.Close()
	}
}
