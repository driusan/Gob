package main

import (
	"fmt"

	"github.com/driusan/gob/css"
	"github.com/driusan/gob/dom"
	"github.com/driusan/gob/net"
	"github.com/driusan/gob/renderer"

	"golang.org/x/exp/shiny/driver"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
	"golang.org/x/mobile/event/touch"
	"golang.org/x/net/html"

	"context"
	"image"
	"image/color"
	"image/draw"
	"net/url"
	"os"
	// "runtime/pprof"
)

var (
	background = color.Color(color.RGBA{0xE0, 0xE0, 0xE0, 0xFF})
	//	background = color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}
)

// for debugging
var hover *renderer.RenderableDomElement
var activeEl *renderer.RenderableDomElement

func debugelement(el *renderer.RenderableDomElement) {
	cur := el.Element
	name := ""
	for {
		curname := cur.Data
		if cl := cur.GetAttribute("class"); cl != "" {
			curname += "." + cl
		}
		if id := cur.GetAttribute("id"); id != "" {
			curname += "#" + id
		}
		if name == "" {
			name = curname
		} else {
			name = curname + " " + name
		}
		if cur.Parent == nil {
			break
		}
		cur = (*dom.Element)(cur.Parent)
	}
	fmt.Printf("Hovering over: %v (%v)\n", name, el.GetTextContent())
	fmt.Printf("Styles applied: %v\n", el.ConditionalStyles)
}

type Viewport struct {
	// The size of the viewport
	Size size.Event

	// The whole, source image to be displayed in the viewport. It will be clipped
	// and displayed in the viewport according to the Size and Cursor
	//	Content image.Image

	// The location of the image to be displayed into the viewport.
	Cursor image.Point

	buffer screen.Buffer
}

var renderCtx context.Context
var cancelRender context.CancelFunc

func newContext() {
	if cancelRender != nil {
		cancelRender()
	}
	renderCtx, cancelRender = context.WithCancel(context.Background())
}
func paintWindow(s screen.Screen, w screen.Window, v *Viewport, page renderer.Page) {
	if v.buffer == nil {
		return
	}

	viewport := v.Size.Bounds()

	// Fill the buffer with the window background colour before
	// drawing the web page on top of it.
	dst := v.buffer.RGBA()
	draw.Draw(dst, dst.Bounds(), &image.Uniform{background}, image.ZP, draw.Src)

	if err := page.Content.RenderInto(renderCtx, dst, v.Cursor); err != nil {
		panic(err)
	}
	w.Upload(image.Point{0, 0}, v.buffer, viewport)
	w.Publish()
}

func main() {
	/*
		f, _ := os.Create("test.profile")
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	*/

	filename := "file:test.html"
	if len(os.Args) > 1 {
		filename = os.Args[1]

	}

	page, err := loadNewPage(nil, filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		return
	}

	driver.Main(func(s screen.Screen) {
		w, err := s.NewWindow(nil)
		if err != nil {
			panic(err)
		}
		defer w.Release()

		var v Viewport
		defer func() {
			if v.buffer != nil {
				v.buffer.Release()
			}
		}()
		// there will be a size event immediately after creating
		// the window which will trigger this.
		for {
			switch e := w.NextEvent().(type) {
			case lifecycle.Event:
				if e.To == lifecycle.StageDead {
					return
				}
			case key.Event:
				switch e.Code {
				case key.CodeEscape:
					fmt.Println("Restart: ", os.Args[0], page.URL)
					return
				case key.CodeLeftArrow:
					if e.Direction == key.DirPress {
						scrollSize := 50
						if e.Modifiers&key.ModControl == key.ModControl {
							scrollSize = 10
						}
						v.Cursor.X -= scrollSize

						if v.Cursor.X < 0 {
							//	v.Cursor.X = 0
						}
						paintWindow(s, w, &v, page)
					}
				case key.CodeRightArrow:
					if e.Direction == key.DirPress {
						scrollSize := 50
						if e.Modifiers&key.ModControl == key.ModControl {
							scrollSize = 10
						}
						v.Cursor.X += scrollSize

						/*
							if v.Cursor.X > v.Content.Bounds().Max.X {
								v.Cursor.X = v.Content.Bounds().Max.X - 10
							}
						*/
						paintWindow(s, w, &v, page)
					}
				case key.CodeDownArrow:
					if e.Direction == key.DirPress {
						scrollSize := v.Size.Size().Y / 2
						if e.Modifiers&key.ModControl == key.ModControl {
							scrollSize = 10
						}

						v.Cursor.Y += scrollSize
						/*
							if v.Cursor.Y > v.Content.Bounds().Max.Y {
								v.Cursor.Y = v.Content.Bounds().Max.Y - 10
							}
						*/
						paintWindow(s, w, &v, page)
					}
				case key.CodeUpArrow:
					if e.Direction == key.DirPress {
						scrollSize := v.Size.Size().Y / 2
						if e.Modifiers&key.ModControl == key.ModControl {
							scrollSize = 10
						}
						v.Cursor.Y -= scrollSize
						if v.Cursor.Y < 0 {
							//	v.Cursor.Y = 0
						}
						paintWindow(s, w, &v, page)
					}
				case key.CodeP:
					if e.Direction == key.DirPress {
						debugelement(hover)
					}
				default:
					// fmt.Printf("Unknown key: %s", e.Code)
				}
			case paint.Event:
				paintWindow(s, w, &v, page)
			case size.Event:
				css.ClearFontCache()
				v.Size = e
				if e.PixelsPerPt != 0 {
					css.PixelsPerPt = float64(e.PixelsPerPt)
				}
				if v.buffer != nil {
					v.buffer.Release()
				}
				v.buffer, err = s.NewBuffer(v.Size.Size())
				if err != nil {
					// If we couldn't get a buffer, there's
					// not much we can do..
					panic(err)
				}
				newContext()
				page.Content.InvalidateLayout()
				page.Content.Layout(renderCtx, v.Size.Size())
				renderNewPageIntoViewport(s, w, &v, page, false)
			case touch.Event:
				fmt.Printf("Touch event!")
			case mouse.Event:
				switch e.Button {
				case mouse.ButtonWheelDown:
					v.Cursor.Y += 10
					if v.Cursor.Y < 0 {
						v.Cursor.Y = 0
					}
					paintWindow(s, w, &v, page)
				case mouse.ButtonWheelUp:
					v.Cursor.Y -= 10
					if v.Cursor.Y < 0 {
						v.Cursor.Y = 0
					}
					paintWindow(s, w, &v, page)
				default:
					if page.Content != nil && page.Content.ImageMap != nil {

						el := page.Content.ImageMap.At(int(e.X)+v.Cursor.X, int(e.Y)+v.Cursor.Y)
						if el != nil {
							switch e.Direction {
							case mouse.DirRelease:
								el.OnClick()
								if el.Type == html.ElementNode && el.Data == "a" {
									newContext()

									p, err := loadNewPage(page.URL, el.GetAttribute("href"))
									page = p
									if err == nil {
										page.Content.Layout(renderCtx, v.Size.Size())
										renderNewPageIntoViewport(s, w, &v, p, true)
									}
								} else if activeEl != nil {
									activeEl.State.Active = false
									activeEl = nil
									page.ReapplyStyles()
									newContext()
									page.Content.InvalidateLayout()
									page.Content.Layout(context.TODO(), v.Size.Size())
									renderNewPageIntoViewport(s, w, &v, page, false)

								}
							case mouse.DirPress:
								if el.State.Link == true || el.State.Visited == true {
									activeEl = el
									el.State.Active = true
									page.ReapplyStyles()
									newContext()
									page.Content.InvalidateLayout()
									page.Content.Layout(context.TODO(), v.Size.Size())
									renderNewPageIntoViewport(s, w, &v, page, false)
								}
							default:
								if el.Type == html.ElementNode && el.Data == "a" {
									//fmt.Printf("Hovering over link %s\n", el.GetAttribute("href"))
								}
								if el != hover {
									if hover != nil {
										hover.State.Hover = false
									}
									el.State.Hover = true
								}
								if el.Type == html.ElementNode {
									hover = el
								}
							}
						}
					}
				}
			default:
				//	fmt.Printf("%s\n", e)
			}
		}
	})
}

func loadNewPage(context *url.URL, path string) (renderer.Page, error) {
	u, err := url.Parse(path)
	if err != nil {
		return renderer.Page{}, err
	}
	var newURL *url.URL
	if context == nil {
		newURL = u
	} else {
		newURL = context.ResolveReference(u)
	}

	loader := net.DefaultReader{}
	r, _, err := loader.GetURL(newURL)
	if err != nil {
		return renderer.Page{}, err
	}
	defer r.Close()

	p := renderer.LoadPage(r, loader, newURL)
	p.URL = newURL
	p.Content.InvalidateLayout()
	background = p.Background
	return p, nil
}

func renderNewPageIntoViewport(s screen.Screen, w screen.Window, v *Viewport, page renderer.Page, newpage bool) {
	if newpage {
		newContext()
		v.Cursor = image.ZP
		page.Content.Layout(renderCtx, v.Size.Size())
	}
	paintWindow(s, w, v, page)
}
