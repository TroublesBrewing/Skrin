// Package logo draws the official Obsidian logo in the terminal.
package logo

import (
	"bytes"
	"image"
	"image/color"
	_ "image/png"
	"math"
	"strings"

	"charm.land/lipgloss/v2"
	"golang.org/x/image/draw"

	"github.com/lurioso/skrin/assets"
)

// opaque is the alpha at which a scaled pixel counts as drawn.
const opaque = 110

// HalfBlock renders the logo rows cells tall. Each cell stacks two pixels
// (▀ / ▄), so pixels come out roughly square, and transparent pixels are
// left to the terminal background. It returns one string per row and the
// width in cells.
func HalfBlock(rows int) ([]string, int, error) {
	img, err := scaled(rows * 2)
	if err != nil {
		return nil, 0, err
	}
	w := img.Bounds().Dx()
	lines := make([]string, rows)
	for y := range lines {
		var b strings.Builder
		for x := 0; x < w; x++ {
			b.WriteString(cell(img.NRGBAAt(x, 2*y), img.NRGBAAt(x, 2*y+1)))
		}
		lines[y] = b.String()
	}
	return lines, w, nil
}

// scaled returns the gem, cropped and resized to h pixels tall.
func scaled(h int) (*image.NRGBA, error) {
	src, _, err := image.Decode(bytes.NewReader(assets.ObsidianLogo))
	if err != nil {
		return nil, err
	}
	gem := isolateGem(src)
	box := opaqueBounds(gem)
	w := max(1, int(math.Round(float64(h)*float64(box.Dx())/float64(box.Dy()))))
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(img, img.Bounds(), gem, box, draw.Src, nil)
	return img, nil
}

// isolateGem keeps only the purple crystal. The packaged icon puts it on a
// dark grey rounded tile, which at terminal sizes would become a dark block
// around a tiny gem, so dark grey pixels are made transparent. The gem's
// pale highlight is bright, not dark, so it survives.
func isolateGem(src image.Image) *image.NRGBA {
	b := src.Bounds()
	out := image.NewNRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := color.NRGBAModel.Convert(src.At(x, y)).(color.NRGBA)
			hi, lo := max(c.R, c.G, c.B), min(c.R, c.G, c.B)
			if hi-lo < 24 && hi < 0x60 {
				c.A = 0
			}
			out.SetNRGBA(x, y, c)
		}
	}
	return out
}

func cell(top, bot color.NRGBA) string {
	st := lipgloss.NewStyle()
	switch t, b := top.A >= opaque, bot.A >= opaque; {
	case t && b:
		return st.Foreground(solid(top)).Background(solid(bot)).Render("▀")
	case t:
		return st.Foreground(solid(top)).Render("▀")
	case b:
		return st.Foreground(solid(bot)).Render("▄")
	}
	return " "
}

func solid(c color.NRGBA) color.Color {
	c.A = 255
	return c
}

// opaqueBounds is the smallest rectangle holding every visible pixel.
func opaqueBounds(img image.Image) image.Rectangle {
	b := img.Bounds()
	minX, minY, maxX, maxY := b.Max.X, b.Max.Y, b.Min.X, b.Min.Y
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a > 0x1000 {
				minX, minY = min(minX, x), min(minY, y)
				maxX, maxY = max(maxX, x+1), max(maxY, y+1)
			}
		}
	}
	if maxX <= minX || maxY <= minY {
		return b
	}
	return image.Rect(minX, minY, maxX, maxY)
}
