package main

import (
	"bytes"
	"image"
	"image/color"

	"fyne.io/fyne"
	"fyne.io/fyne/canvas"

	"github.com/icza/bitio"

	"osdapp/frskyosd"
)

// FontIcon is a convenience type wrapping a canvas.Image
// that allows setting font character data directly.
type FontIcon struct {
	canvas.Image
}

// SetFontData updates the icon to display the given font
// data, which must be 54 bytes.
func (i *FontIcon) SetFontData(data []byte) {
	bg := color.Gray{
		Y: (255 * 3) / 4,
	}
	img := image.NewRGBA(image.Rect(0, 0, 12, 18))
	for ii := 0; ii < img.Rect.Dx(); ii++ {
		for jj := 0; jj < img.Rect.Dy(); jj++ {
			img.Set(ii, jj, bg)
		}
	}
	if len(data) > 0 {
		r := bitio.NewReader(bytes.NewReader(data))
		for jj := 0; jj < img.Rect.Dy(); jj++ {
			for ii := 0; ii < img.Rect.Dx(); ii++ {
				var c color.Color = nil
				pix, err := r.ReadBits(2)
				if err != nil {
					panic(err)
				}
				switch pix {
				case 0:
					c = color.Black
				case 1:
				case 2:
					c = color.White
				case 3:
					c = color.Gray{Y: 127}
				}
				if c != nil {
					img.Set(ii, jj, c)
				}
			}
		}
	}
	i.Image.Image = img
	i.Image.Refresh()
}

// SetFont updates the displayed font image from a FontCharMessage
func (i *FontIcon) SetFont(msg *frskyosd.FontCharMessage) {
	var data []byte
	if msg != nil {
		data = msg.Data[:]
	}
	i.SetFontData(data)
}

// NewFontIcon returns a *FontIcon ready to be used
func NewFontIcon() *FontIcon {
	fi := &FontIcon{}
	fi.Image.FillMode = canvas.ImageFillContain
	fi.SetMinSize(fyne.NewSize(12, 18))
	fi.SetFont(nil)
	return fi
}
