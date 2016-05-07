package renderer

import (
	"Gob/css"
	"Gob/dom"
	"Gob/net"
	"fmt"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
	"golang.org/x/net/html"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/url"
	"os"
	//"strconv"
	"strings"
)

const (
	DefaultFontSize = 16
)

// A RenderableElement is something that can be rendered to
// an image.
type Renderer interface {
	// Returns an image representing this element.
	Render(containerWidth int) image.Image
}

type RenderableDomElement struct {
	*dom.Element
	Styles *css.StyledElement

	Parent      *RenderableDomElement
	FirstChild  *RenderableDomElement
	NextSibling *RenderableDomElement
	PrevSibling *RenderableDomElement

	CSSOuterBox    image.Image
	ContentOverlay image.Image

	ImageMap       ImageMap
	PageLocation   *url.URL
	FirstPageOnly  bool
	RenderAbort    chan bool
	ViewportHeight int
	contentWidth   int
	containerWidth int
}

func getFontHeight(face font.Face) int {
	metrics := face.Metrics()
	return (metrics.Ascent + metrics.Descent).Ceil()
}
func stringSize(fntDrawer font.Drawer, textContent string) (int, error) {
	var size int
	words := strings.Fields(textContent)
	fSize := getFontHeight(fntDrawer.Face)
	//firstRune, _ := utf8.DecodeRuneInString(textContent)

	for _, word := range words {
		wordSizeInPx := fntDrawer.MeasureString(word).Ceil()
		size += wordSizeInPx

		// Add a three per em space between words, an em-quad after a period,
		// and an en-quad after other punctuation
		switch word[len(word)-1] {
		case ',', ';', ':', '!', '?':
			size += (fSize / 2)
		case '.':
			size += fSize
		default:
			size += (fSize / 3)
		}
	}
	return size, nil
}

func (e *RenderableDomElement) GetLineHeight() int {
	// inheritd == yes
	// percentage relative to the font size of the element itself
	fSize := e.GetFontSize()
	if e.Styles == nil {
		if e.Parent == nil {
			fontFace := e.Styles.GetFontFace(fSize)
			return getFontHeight(fontFace)
		}
		//return e.Parent.GetLineHeight()
		fontFace := e.Styles.GetFontFace(fSize)
		return getFontHeight(fontFace)
	}
	stringVal := e.Styles.LineHeight.GetValue()
	if stringVal == "" {
		if e.Parent == nil {
			fontFace := e.Styles.GetFontFace(fSize)
			return getFontHeight(fontFace)
		}
		//return e.Parent.GetLineHeight()
		fontFace := e.Styles.GetFontFace(fSize)
		return getFontHeight(fontFace)

	}
	lHeightSize, err := css.ConvertUnitToPx(fSize, fSize, stringVal)
	if err != nil {
		fontFace := e.Styles.GetFontFace(fSize)
		return getFontHeight(fontFace)
	}
	fontFace := e.Styles.GetFontFace(lHeightSize)
	return getFontHeight(fontFace)
}

func (e *RenderableDomElement) GetFontSize() int {
	fromCSS, err := e.Styles.GetFontSize()
	switch err {
	case css.NoStyles, css.InheritValue:
		if e.Parent == nil {
			return DefaultFontSize
		}
		return e.Parent.GetFontSize()
	case nil:
		return fromCSS
	default:
		panic("Could not determine font size")

	}
}

func (e *RenderableDomElement) Walk(callback func(*RenderableDomElement)) {
	if e == nil {
		return
	}

	if e.Type == html.ElementNode {
		callback(e)
	}

	for c := e.FirstChild; c != nil; c = c.NextSibling {
		switch c.Type {
		case html.ElementNode:
			callback(c)
			c.Walk(callback)
		}
	}
}

func (e RenderableDomElement) GetBackgroundColor() color.Color {
	switch bg, err := e.Styles.GetBackgroundColor(dfltBackground); err {
	case css.InheritValue:
		if e.Parent == nil {
			//return dfltBackground
			return color.Transparent //dfltBackground
			//&color.RGBA{0xE0, 0xE0, 0xE0, 0xFF}
		}
		return e.Parent.GetBackgroundColor()
	case css.NoStyles:
		return color.Transparent
		//return dfltBackground
	default:
		return bg
	}
}

func (e RenderableDomElement) GetColor() color.Color {
	var deflt color.RGBA
	if e.Type == html.ElementNode && e.Data == "a" {
		deflt = color.RGBA{0, 0, 0xFF, 0xFF}
	} else {
		deflt = color.RGBA{0, 0, 0, 0xFF}
	}
	switch cssColor, err := e.Styles.GetColor(deflt); err {
	case css.InheritValue:
		if e.Parent == nil {
			return deflt
		}
		return e.Parent.GetColor()
	case css.NoStyles:
		if e.Parent == nil {
			return deflt
		}
		return e.Parent.GetColor()
	default:
		return cssColor
	}
}

func (e RenderableDomElement) GetDisplayProp() string {
	if e.Type == html.TextNode {
		return "inline"
	}
	if cssVal := e.Styles.DisplayProp(); cssVal != "" {
		return cssVal
	}
	// CSS Level 1 default is block, CSS Level 2 is inline
	return "block"
	//return "inline"
}

func (e RenderableDomElement) GetTextDecoration() string {
	if e.Styles == nil {
		return "none"
	}

	switch decoration := e.Styles.TextDecoration.GetValue(); decoration {
	case "inherit":
		return e.Parent.GetTextDecoration()
	default:
		return strings.TrimSpace(decoration)
	}
}
func (e RenderableDomElement) GetTextTransform() string {
	if e.Styles == nil {
		return "none"
	}

	switch transformation := e.Styles.TextTransform.GetValue(); transformation {
	case "inherit":
		return e.Parent.GetTextTransform()
	case "capitalize", "uppercase", "lowercase", "none":
		return transformation
	default:
		if e.Parent == nil {
			return "none"
		}
		return e.Parent.GetTextTransform()
	}
}
func (e RenderableDomElement) renderLineBox(remainingWidth int, textContent string) (img *image.RGBA, unconsumed string) {
	switch e.GetTextTransform() {
	case "capitalize":
		textContent = strings.Title(textContent)
	case "uppercase":
		textContent = strings.ToUpper(textContent)
	case "lowercase":
		textContent = strings.ToLower(textContent)
	}
	words := strings.Fields(textContent)
	fSize := e.GetFontSize()
	fontFace := e.Styles.GetFontFace(fSize)
	var dot int
	clr := e.GetColor()
	if clr == nil {
		clr = color.RGBA{0xff, 0xff, 0xff, 0xff}
	}
	fntDrawer := font.Drawer{
		Dst:  nil,
		Src:  &image.Uniform{clr},
		Face: fontFace,
	}

	ssize, _ := stringSize(fntDrawer, textContent)
	if ssize > remainingWidth {
		ssize = remainingWidth
	}
	lineheight := e.GetLineHeight()
	img = image.NewRGBA(image.Rectangle{image.ZP, image.Point{ssize, lineheight}})

	//BUG(driusan): This math is wrong
	fntDrawer.Dot = fixed.P(0, fontFace.Metrics().Ascent.Floor())
	fntDrawer.Dst = img

	if decoration := e.GetTextDecoration(); decoration != "" && decoration != "none" && decoration != "blink" {
		if strings.Contains(decoration, "underline") {
			y := fntDrawer.Dot.Y.Floor()
			for px := 0; px < ssize; px++ {
				img.Set(px, y, clr)
			}
		}
		if strings.Contains(decoration, "overline") {
			y := fntDrawer.Dot.Y.Floor() - fontFace.Metrics().Ascent.Floor()
			for px := 0; px < ssize; px++ {
				img.Set(px, y, clr)
			}
		}
		if strings.Contains(decoration, "line-through") {
			y := fntDrawer.Dot.Y.Floor() - (fontFace.Metrics().Ascent.Floor() / 2)
			for px := 0; px < ssize; px++ {
				img.Set(px, y, clr)
			}
		}
	}
	for i, word := range words {
		wordSizeInPx := int(fntDrawer.MeasureString(word)) >> 6
		if dot+wordSizeInPx > remainingWidth {
			if i == 0 {
				// make sure at least one word gets consumed to avoid an infinite loop.
				// this isn't ideal, since some words will disappear, but if we reach this
				// point we're already in a pretty bad state..
				unconsumed = strings.Join(words[i+1:], " ")
			} else {
				unconsumed = strings.Join(words[i:], " ")
			}
			return
		}
		fntDrawer.DrawString(word)

		// Add a three per em space between words, an em-quad after a period,
		// and an en-quad after other punctuation
		switch word[len(word)-1] {
		case ',', ';', ':', '!', '?':
			dot = (int(fntDrawer.Dot.X) >> 6) + (fSize / 2)
		case '.':
			dot = (int(fntDrawer.Dot.X) >> 6) + fSize
		default:
			dot = (int(fntDrawer.Dot.X) >> 6) + (fSize / 3)
		}
		fntDrawer.Dot.X = fixed.Int26_6(dot << 6)
	}
	unconsumed = ""
	return
}

func (e RenderableDomElement) GetTextIndent(containerWidth int) int {
	// it's inherited, with the initial value of 0
	if e.Styles == nil {
		if e.Parent == nil {
			return 0
		}
		return e.Parent.GetTextIndent(containerWidth)
	}
	val := e.Styles.TextIndent.GetValue()
	if val == "" {
		if e.Parent == nil {
			return 0
		}
		return e.Parent.GetTextIndent(containerWidth)
	}
	px, err := css.ConvertUnitToPx(e.GetFontSize(), containerWidth, val)
	if err != nil {
		return 0
	}
	return px
}

func (e RenderableDomElement) GetContentWidth(containerWidth int) int {
	width := containerWidth - (e.GetMarginLeftSize() + e.GetMarginRightSize() + e.GetBorderLeftWidth() + e.GetBorderRightWidth() + e.GetPaddingLeft() + e.GetPaddingRight())
	if e.Styles == nil {
		return width
	}
	cssVal := e.Styles.Width.GetValue()
	switch cssVal {
	case "inherit":
		if e.Parent == nil {
			return width
		}
		return e.Parent.GetContentWidth(containerWidth)
	case "", "auto":
		return width
	default:
		calVal, err := css.ConvertUnitToPx(e.GetFontSize(), containerWidth, cssVal)
		if err == nil {
			return calVal
		}
		return width
	}
}
func (e *RenderableDomElement) Render(containerWidth int) image.Image {
	size, _ := e.realRender(containerWidth, true, image.ZR, image.Point{0, 0})
	img, _ := e.realRender(containerWidth, false, size.Bounds(), image.Point{0, 0})
	return img
}

func (e *RenderableDomElement) realRender(containerWidth int, measureOnly bool, r image.Rectangle, dot image.Point) (image.Image, image.Point) {
	var dst draw.Image
	e.RenderAbort = make(chan bool)
	select {
	case <-e.RenderAbort:
		for c := e.FirstChild; c != nil; c = c.NextSibling {
			if c.RenderAbort != nil {
				c.RenderAbort <- true
			}
		}
		close(e.RenderAbort)
		return dst, image.ZP
	default:
		dot := image.Point{dot.X, dot.Y}

		width := e.GetContentWidth(containerWidth)
		e.contentWidth = width
		e.containerWidth = containerWidth
		height := 0

		// special cases
		if e.Type == html.ElementNode {
			switch strings.ToLower(e.Data) {
			case "img":
				var loadedImage bool
				for _, attr := range e.Attr {
					if loadedImage {
						return e.ContentOverlay, dot
					}
					switch attr.Key {
					case "src":
						fmt.Printf("Should load: %s\n", attr.Val)
						u, err := url.Parse(attr.Val)
						if err != nil {
							panic("At the disco")
							loadedImage = true
							break
						}
						newURL := e.PageLocation.ResolveReference(u)
						r, err := net.GetURLReader(newURL)
						if err != nil {
							panic(err)
						}
						content, format, err := image.Decode(r)
						if err == nil {
							e.ContentOverlay = content
						} else {
							fmt.Fprintf(os.Stderr, "Unknown image format: %s Err: %s", format, err)
						}

						loadedImage = true

						/*
							case "width":
								width, _ = strconv.Atoi(attr.Val)
							case "height":
								height, _ = strconv.Atoi(attr.Val)
							}
						*/
					}
				}
			}
		}

		var mst *DynamicMemoryDrawer
		if measureOnly {
			mst = NewDynamicMemoryDrawer(image.Rectangle{image.ZP, image.Point{width, height}})
			dst = mst
		} else {
			dst = image.NewRGBA(r)
		}

		firstLine := true
		imageMap := NewImageMap()
		for c := e.FirstChild; c != nil; c = c.NextSibling {
			c.ViewportHeight = e.ViewportHeight
			switch c.Type {
			case html.TextNode:
				// text nodes are inline elements that didn't match
				// anything when adding styles, but that's okay,
				// because their style should be identical to their
				// parent.
				c.Styles = e.Styles

				if firstLine == true {
					dot.X += c.GetTextIndent(width)
					firstLine = false
				}

				remainingTextContent := strings.TrimSpace(c.Data)
				for remainingTextContent != "" {
					childImage, rt := c.renderLineBox(width-dot.X, remainingTextContent)
					remainingTextContent = rt
					sr := childImage.Bounds()
					r := image.Rectangle{dot, dot.Add(sr.Size())}
					if measureOnly {
						mst.GrowBounds(r)
					} else {
						draw.Draw(dst, r, childImage, sr.Min, draw.Src)
					}
					if r.Max.X >= width {
						dot.X = 0
						dot.Y += e.GetLineHeight()
					} else {
						dot.X = r.Max.X
					}
					// add this line box to the image map.
					imageMap.Add(c, r)
				}
			case html.ElementNode:
				if c.Data == "br" {
					dot.X = 0
					dot.Y += c.GetLineHeight()
					continue
				}
				switch c.GetDisplayProp() {
				case "none":
					continue
				case "inline":
					if firstLine == true {
						dot.X += c.GetTextIndent(width)
						firstLine = false
					}
					size, _ := c.realRender(width, true, image.ZR, dot)
					childContent, newDot := c.realRender(width, measureOnly, size.Bounds(), dot)

					if measureOnly == false {
						c.ContentOverlay = childContent
						bounds := childContent.Bounds()
						draw.Draw(
							dst,
							image.Rectangle{image.ZP, bounds.Max},
							c.ContentOverlay,
							bounds.Min,
							draw.Over,
						)

						// Populate this image map. This is an inline, so we actually only care
						// about the line boxes that were generated by the children.
						childImageMap := c.ImageMap
						for _, area := range childImageMap {
							// translate the coordinate systems from the child's to this one
							newArea := area.Area //.Add(dot)

							if area.Content.Type == html.TextNode {
								// it was a text node, so for all intents and purposes we're actually
								// hovering over this element
								imageMap.Add(c, newArea)
							} else {
								// it was a child element node, so it's more precise to say we were hovering
								// over the child
								imageMap.Add(area.Content, newArea)
							}
						}
					}
					dot.X = newDot.X
					dot.Y = newDot.Y

				case "block":
					fallthrough
				default:
					if dot.X != 0 {
						// This means the previous child was an inline item, and we should position dot
						// as if there were an implicit box around it.
						dot.X = 0
						if c.PrevSibling != nil {
							dot.Y += c.PrevSibling.GetLineHeight()
						}
					}
					// draw the border, background, and CSS outer box.
					childContent, _ := c.realRender(width, true, image.ZR, image.ZP)
					if measureOnly == false {
						childContent, _ = c.realRender(width, measureOnly, childContent.Bounds(), image.ZP)
					}
					c.ContentOverlay = childContent
					box, contentorigin := c.getCSSBox(childContent, measureOnly)
					sr := box.Bounds()
					r := image.Rectangle{dot, dot.Add(sr.Size())}

					if measureOnly {
						mst.GrowBounds(r)
					} else {
						// draw the box
						draw.Draw(
							dst,
							r,
							c.CSSOuterBox,
							sr.Min,
							draw.Over,
						)
					}

					// populate the imagemap by adding the child, then adding the children's
					// children.
					// add the child
					childImageMap := c.ImageMap
					imageMap.Add(c, r)
					// add the grandchildren
					for _, area := range childImageMap {
						// translate the coordinate systems from the child's to this one
						newArea := area.Area.Add(dot).Add(contentorigin)
						imageMap.Add(area.Content, newArea)
					}

					// now draw the content on top of the outer box
					contentStart := dot.Add(contentorigin)
					contentBounds := c.ContentOverlay.Bounds()
					cr := image.Rectangle{contentStart, contentStart.Add(contentBounds.Size())}
					if measureOnly {
						mst.GrowBounds(cr)
					} else {
						draw.Draw(
							dst,
							cr,
							c.ContentOverlay,
							contentBounds.Min,
							draw.Over,
						)
					}

					dot.X = 0
					dot.Y = r.Max.Y
				}

			}
			if e.FirstPageOnly && dot.Y > e.ViewportHeight {
				return dst, dot
			}
		}
		e.ImageMap = imageMap
		return dst, dot
	}
}
