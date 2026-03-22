package main

import (
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var (
	playerSprites [4]*ebiten.Image
	bgImage       *ebiten.Image
)

func init() {
	// Initialize the 4 directional sprites
	for i := 0; i < 4; i++ {
		playerSprites[i] = createPlayerImage(i)
	}
	// Initialize the town background
	bgImage = createTownBackground()
}

// createTownBackground procedurally generates a simple 2D town background
func createTownBackground() *ebiten.Image {
	img := ebiten.NewImage(320, 240)

	// Grass (Medium Sea Green)
	img.Fill(color.NRGBA{60, 179, 113, 255})

	drawRect := func(x, y, w, h int, c color.Color) {
		for i := 0; i < w; i++ {
			for j := 0; j < h; j++ {
				if x+i >= 0 && x+i < 320 && y+j >= 0 && y+j < 240 {
					img.Set(x+i, y+j, c)
				}
			}
		}
	}

	// Dirt paths (Dark Goldenrod)
	dirt := color.RGBA{184, 134, 11, 255}
	drawRect(0, 100, 320, 40, dirt) // horizontal
	drawRect(140, 0, 40, 240, dirt) // vertical

	// Draw house helper
	drawHouse := func(bx, by int) {
		walls := color.RGBA{222, 184, 135, 255} // Burlywood
		drawRect(bx, by, 60, 40, walls)
		roof := color.RGBA{178, 34, 34, 255}    // FireBrick
		drawRect(bx-4, by-10, 68, 20, roof)
		door := color.RGBA{139, 69, 19, 255}    // SaddleBrown
		drawRect(bx+20, by+20, 20, 20, door)
		win := color.RGBA{135, 206, 235, 255}   // SkyBlue
		drawRect(bx+8, by+10, 12, 12, win)
		drawRect(bx+40, by+10, 12, 12, win)
	}

	// Draw 4 houses
	drawHouse(40, 40)
	drawHouse(220, 40)
	drawHouse(40, 160)
	drawHouse(220, 160)

	return img
}

// createPlayerImage procedurally generates a simple 2D pixel art character.
// dir: 0=down, 1=up, 2=left, 3=right
func createPlayerImage(dir int) *ebiten.Image {
	img := ebiten.NewImage(32, 32)

	skin := color.RGBA{255, 204, 153, 255}
	shirt := color.RGBA{0, 102, 204, 255}
	pants := color.RGBA{40, 40, 40, 255}
	hair := color.RGBA{100, 50, 0, 255}
	eyeColor := color.RGBA{0, 0, 0, 255}

	drawRect := func(x, y, w, h int, c color.Color) {
		for i := 0; i < w; i++ {
			for j := 0; j < h; j++ {
				img.Set(x+i, y+j, c)
			}
		}
	}

	if dir == 1 { // Up (Back view)
		drawRect(8, 4, 16, 16, hair) // Full hair head
		drawRect(10, 20, 12, 8, shirt)
		drawRect(11, 28, 4, 4, pants)
		drawRect(17, 28, 4, 4, pants)
		return img
	}

	// Head
	drawRect(8, 4, 16, 16, skin)
	// Hair
	drawRect(8, 2, 16, 4, hair)
	drawRect(7, 4, 2, 10, hair)
	drawRect(23, 4, 2, 10, hair)

	// Body
	drawRect(10, 20, 12, 8, shirt)
	// Legs
	drawRect(11, 28, 4, 4, pants)
	drawRect(17, 28, 4, 4, pants)

	// Eyes
	if dir == 0 { // Down
		drawRect(11, 10, 2, 4, eyeColor)
		drawRect(19, 10, 2, 4, eyeColor)
	} else if dir == 2 { // Left
		drawRect(9, 10, 2, 4, eyeColor)
	} else if dir == 3 { // Right
		drawRect(21, 10, 2, 4, eyeColor)
	}

	return img
}

type Rect struct {
	X, Y, W, H float64
}

var houses = []Rect{
	{36, 30, 68, 50},  // House 1
	{216, 30, 68, 50}, // House 2
	{36, 150, 68, 50}, // House 3
	{216, 150, 68, 50},// House 4
}

func isColliding(x, y float64) bool {
	pw, ph := 32.0, 32.0

	// Screen bounds
	if x < 0 || x+pw > 320 || y < 0 || y+ph > 240 {
		return true
	}

	// Houses
	for _, h := range houses {
		if x < h.X+h.W && x+pw > h.X && y < h.Y+h.H && y+ph > h.Y {
			return true
		}
	}
	return false
}

type Game struct {
	x, y float64
	dir  int // 0: down, 1: up, 2: left, 3: right
}

func (g *Game) Update() error {
	// Toggle full screen
	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}

	newX, newY := g.x, g.y

	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		newX -= 2
		g.dir = 2
	} else if ebiten.IsKeyPressed(ebiten.KeyRight) {
		newX += 2
		g.dir = 3
	} else if ebiten.IsKeyPressed(ebiten.KeyUp) {
		newY -= 2
		g.dir = 1
	} else if ebiten.IsKeyPressed(ebiten.KeyDown) {
		newY += 2
		g.dir = 0
	}

	// Try moving in X
	if newX != g.x && !isColliding(newX, g.y) {
		g.x = newX
	}
	// Try moving in Y
	if newY != g.y && !isColliding(g.x, newY) {
		g.y = newY
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Draw the town background
	screen.DrawImage(bgImage, nil)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(g.x, g.y)
	
	// Draw the player sprite based on the current direction
	screen.DrawImage(playerSprites[g.dir], op)
}

func (g *Game) Layout(w, h int) (int, int) {
	return 320, 240
}

func main() {
	game := &Game{x: 144, y: 104, dir: 0}
	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Pokemon Master - Moving Test")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
