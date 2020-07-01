package main

import (
	"image"
	"image/color"
	"image/draw"
	"strings"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

const textWidthPercentage = 0.76
const bottomMargin = 0.07

func textWidth(fnt *truetype.Font, scale int, text string) fixed.Int26_6 {
	var sum fixed.Int26_6

	s := fixed.Int26_6(scale << 6)

	var prevIdx truetype.Index
	for i, c := range text {
		curIdx := fnt.Index(c)
		h := fnt.HMetric(s, curIdx)
		sum += h.AdvanceWidth
		if i > 0 {
			sum += fnt.Kern(s, prevIdx, curIdx)
		}
		prevIdx = curIdx
	}

	return sum
}

func textHeight(fnt *truetype.Font, scale int, text string) fixed.Int26_6 {
	var max fixed.Int26_6

	s := fixed.Int26_6(scale << 6)

	for _, c := range text {
		curIdx := fnt.Index(c)
		v := fnt.VMetric(s, curIdx)
		h := v.AdvanceHeight
		if h > max {
			max = h
		}
	}

	return max
}

func drawCaption(img image.Image, fnt *truetype.Font, text string) (image.Image, error) {
	width10 := textWidth(fnt, 10, text).Ceil()
	width100 := textWidth(fnt, 100, text).Ceil()

	// Approximate mean glyph width
	k := ((float64(width10) / 10) + (float64(width100) / 100)) / 2
	h := img.Bounds().Max.Y
	w := img.Bounds().Max.X
	size := float64(w) * textWidthPercentage / k

	if size < 48.0 {
		size = 48.0
	}

	if size > 150.0 {
		size = 150.0
	}

	size = float64(int(size))
	tw := textWidth(fnt, int(size), text).Ceil()

	var lines []string
	if tw > w {
		words := strings.Split(text, " ")
		if len(words) == 1 {
			lines = []string{text}
		} else {
			n := len(words) / 2
			lines = []string{strings.Join(words[:n], " "), strings.Join(words[n:], " ")}
		}
	} else {
		lines = []string{text}
	}

	// Construct font face
	face := truetype.NewFace(fnt, &truetype.Options{
		Size:    size,
		DPI:     72,
		Hinting: 3,
	})

	// Allocate new image
	out := image.NewRGBA(img.Bounds())
	// Copy decoded image to new image
	draw.Draw(out, out.Bounds(), img, img.Bounds().Min, draw.Src)

	yOffset := 0

	botLine := int(bottomMargin * float64(h))
	topLine := botLine + 80
	meanLuminosity := 0.0

	for i := botLine; i < topLine; i++ {
		for j := 0; j < w; j++ {
			r, g, b, _ := img.At(j, h-i).RGBA()
			meanLuminosity += 0.2989*float64(r) + 0.5870*float64(g) + 0.1140*float64(b)
		}
	}

	meanLuminosity /= float64((topLine - botLine) * w * 65536)

	c := color.RGBA{R: 255, G: 255, B: 255, A: 255}

	if meanLuminosity > 0.7 {
		c = color.RGBA{A: 255}
	}

	for idx := range lines {
		jdx := len(lines) - idx - 1
		line := lines[jdx]

		tw = textWidth(fnt, int(size), line).Ceil()

		x := int((float64(w) - float64(tw)) / 2)

		y := h - int(bottomMargin*float64(h)) - yOffset
		yOffset += textHeight(fnt, int(size), line).Ceil()

		// Draw string
		drawer := &font.Drawer{
			Dst:  out,
			Src:  image.NewUniform(c),
			Face: face,
			Dot:  fixed.Point26_6{X: fixed.Int26_6(x << 6), Y: fixed.Int26_6(y << 6)},
		}
		drawer.DrawString(line)
	}

	return out, nil
}
