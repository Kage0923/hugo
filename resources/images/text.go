// Copyright 2021 The Hugo Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package images

import (
	"image"
	"image/draw"
	"io"
	"strings"

	"github.com/disintegration/gift"
	"github.com/gohugoio/hugo/common/hugio"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

var _ gift.Filter = (*textFilter)(nil)

type textFilter struct {
	text, color string
	x, y        int
	size        float64
	linespacing int
	fontSource  hugio.ReadSeekCloserProvider
}

func (f textFilter) Draw(dst draw.Image, src image.Image, options *gift.Options) {
	color, err := hexStringToColor(f.color)
	if err != nil {
		panic(err)
	}

	// Load and parse font
	ttf := goregular.TTF
	if f.fontSource != nil {
		rs, err := f.fontSource.ReadSeekCloser()
		if err != nil {
			panic(err)
		}
		defer rs.Close()
		ttf, err = io.ReadAll(rs)
		if err != nil {
			panic(err)
		}
	}
	otf, err := opentype.Parse(ttf)
	if err != nil {
		panic(err)
	}

	// Set font options
	face, err := opentype.NewFace(otf, &opentype.FaceOptions{
		Size:    f.size,
		DPI:     72,
		Hinting: font.HintingNone,
	})
	if err != nil {
		panic(err)
	}

	d := font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(color),
		Face: face,
	}

	gift.New().Draw(dst, src)

	// Draw text, consider and include linebreaks
	maxWidth := dst.Bounds().Dx() - 20
	fontHeight := face.Metrics().Ascent.Ceil()

	// Correct y position based on font and size
	f.y = f.y + fontHeight

	// Start position
	y := f.y
	d.Dot = fixed.P(f.x, f.y)

	// Draw text and break line at max width
	parts := strings.Fields(f.text)
	for _, str := range parts {
		strWith := font.MeasureString(face, str)
		if (d.Dot.X.Ceil() + strWith.Ceil()) >= maxWidth {
			y = y + fontHeight + f.linespacing
			d.Dot = fixed.P(f.x, y)
		}
		d.DrawString(str + " ")
	}
}

func (f textFilter) Bounds(srcBounds image.Rectangle) image.Rectangle {
	return image.Rect(0, 0, srcBounds.Dx(), srcBounds.Dy())
}
