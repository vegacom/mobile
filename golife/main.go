// All rights reserved. Use of this source code is governed by a BSD-style license that can be found in the LICENSE file.

/*
Binary golife is an implementation of Conway's game of life for the Go mobile platform.

http://en.wikipedia.org/wiki/Conway%27s_Game_of_Life
*/
package main

import (
	"image"
	"log"
	"math/rand"
	"time"

	_ "image/png"

	"golang.org/x/mobile/app"
	"golang.org/x/mobile/event"
	"golang.org/x/mobile/f32"
	"golang.org/x/mobile/geom"
	"golang.org/x/mobile/gl"
	"golang.org/x/mobile/sprite"
	"golang.org/x/mobile/sprite/clock"
	"golang.org/x/mobile/sprite/glsprite"
)

// Units are in Pt.
const (
	systemBarHeight = 12
	buttonSize      = 14
	buttonSep       = 6
	buttonBarHeight = 15
)

const (
	maxUint32          = 1<<32 - 1
	initialRenderEvery = 5
)

var (
	cellSize  geom.Pt = 8
	start             = time.Now()
	lastClock         = clock.Time(-1)
	// renderEvery is used to decrease the frequency the next generation is rendered. A value of 3
	// means to render once every three render calls.
	renderEvery uint32 = initialRenderEvery

	eng       = glsprite.Engine()
	scene     *sprite.Node
	textures  map[string]*sprite.SubTex
	buttonBar buttonMap
)

// A universe contains what images to display for each cell state.
type universe struct {
	rows  int
	cols  int
	cells []*sprite.Node
	life  *Life
}

// A button is a clickable image that triggers an action.
type button struct {
	rect *geom.Rectangle // Uses absolute location.
}

// A buttonMap contains the buttons in the button bar.
type buttonMap map[string]*button

func main() {
	rand.Seed(time.Now().UnixNano())
	app.Run(app.Callbacks{
		Draw:  draw,
		Touch: touch,
	})
}

func (b button) contains(point geom.Point) bool {
	return b.rect.Min.X <= point.X && point.X <= b.rect.Max.X &&
		b.rect.Min.Y <= point.Y && point.Y <= b.rect.Max.Y
}

// newButtonMap creates a button bar. The buttons are centered on the top of the screen.
func newButtonMap(imgs ...string) buttonMap {
	var (
		number     = geom.Pt(len(imgs))
		leftMargin = (geom.Width - number*buttonSize - (number-1)*buttonSep) / 2
		buttonBar  = make(buttonMap)
	)
	for k, img := range imgs {
		n := &sprite.Node{}
		eng.Register(n)
		scene.AppendChild(n)
		x := leftMargin + (buttonSize+buttonSep)*geom.Pt(k)
		rect := &geom.Rectangle{
			Min: geom.Point{X: x, Y: systemBarHeight},
			Max: geom.Point{X: x + buttonSize, Y: systemBarHeight + buttonSize},
		}
		buttonBar[img] = &button{rect: rect}
		eng.SetTransform(n, f32.Affine{
			{buttonSize, 0, float32(x)},
			{0, buttonSize, 0},
		})
		eng.SetSubTex(n, *textures[img])
	}
	return buttonBar
}

// find returns the name of the button that contains point if any.
func (buttonBar buttonMap) find(point geom.Point) string {
	for img, b := range buttonBar {
		if b.contains(point) {
			return img
		}
	}
	return ""
}

func newUniverse(h, w geom.Pt) *universe {
	var (
		rows = int(h / cellSize)
		cols = int(w / cellSize)
		u    = &universe{
			rows: rows,
			cols: cols,
			life: NewLife(cols, rows),
		}
		siz = float32(cellSize)
	)
	for j := 0; j < u.rows; j++ {
		for i := 0; i < u.cols; i++ {
			n := &sprite.Node{}
			eng.Register(n)
			scene.AppendChild(n)
			eng.SetTransform(n, f32.Affine{
				{siz, 0, float32(i) * siz},
				{0, siz, buttonBarHeight + float32(j)*siz},
			})
			u.cells = append(u.cells, n)
		}
	}
	return u
}

func (u *universe) Step() {
	u.life.Step()
	var i, j int
	var img string
	for k, cell := range u.cells {
		j = k / u.cols
		i = k % u.cols
		img = emptyImage
		if u.life.A.Alive(i, j) {
			img = androidImage
		}
		// TODO: compare current with prev value. If equal, no-op.
		eng.SetSubTex(cell, *textures[img])
	}
}

func draw() {
	if scene == nil {
		loadScene()
	}

	now := clock.Time(time.Since(start) * 60 / time.Second)
	if now == lastClock {
		return
	}
	lastClock = now

	gl.ClearColor(1, 1, 1, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT)
	eng.Render(scene, now)
}

func touch(t event.Touch) {
	if t.Type != event.TouchEnd {
		// Naive implementation of button event handling: it only matters when/where the user stops
		// touching the screen.
		return
	}

	switch img := buttonBar.find(t.Loc); img {
	case incSpeedImage:
		if renderEvery > 1 {
			renderEvery--
		}
	case decSpeedImage:
		renderEvery++
	case pauseImage:
		if renderEvery == maxUint32 {
			// TODO(vegacom): add a 'play' button and flip it with 'pause'.
			renderEvery = initialRenderEvery
		} else {
			renderEvery = maxUint32
		}
	case replayImage:
		// TODO: implement replay.
	}
}

func loadScene() {
	textures = loadTextures()
	scene = &sprite.Node{}
	eng.Register(scene)
	eng.SetTransform(scene, f32.Affine{
		{1, 0, 0.1},
		{0, 1, systemBarHeight},
	})
	buttonBar = newButtonMap(pauseImage, decSpeedImage, incSpeedImage, replayImage)

	u := newUniverse(geom.Height-systemBarHeight-buttonBarHeight, geom.Width)
	count := uint32(1)
	scene.Arranger = arrangerFunc(func(eng sprite.Engine, n *sprite.Node, t clock.Time) {
		if count%renderEvery == 0 {
			u.Step()
		}
		if count == renderEvery {
			count = 0
		}
		count++
	})
}

// Images to load.
const (
	emptyImage    = "empty"
	androidImage  = "android"
	pauseImage    = "pause"
	decSpeedImage = "speed_decrease"
	incSpeedImage = "speed_increase"
	replayImage   = "replay"
)

func loadTextures() map[string]*sprite.SubTex {
	m := make(map[string]*sprite.SubTex)
	for _, name := range []string{androidImage, pauseImage, replayImage, incSpeedImage, decSpeedImage} {
		tex, err := newTexture(name)
		if err != nil {
			log.Fatal(err)
		}
		// Units are in px.
		m[name] = &sprite.SubTex{tex, image.Rect(0, 0, 72, 72)}
	}
	// Reuse the android image left-top corner (1 px square).
	m[emptyImage] = &sprite.SubTex{m[androidImage].T, image.Rect(1, 1, 2, 2)}
	return m
}

func newTexture(name string) (sprite.Texture, error) {
	a, err := app.Open(name + ".png")
	if err != nil {
		return nil, err
	}
	defer a.Close()

	img, _, err := image.Decode(a)
	if err != nil {
		return nil, err
	}
	return eng.LoadTexture(img)
}

type arrangerFunc func(e sprite.Engine, n *sprite.Node, t clock.Time)

func (a arrangerFunc) Arrange(e sprite.Engine, n *sprite.Node, t clock.Time) { a(e, n, t) }
