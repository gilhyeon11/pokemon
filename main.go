package main

import (
	"image"
	"image/color"
	_ "image/png"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type GameState int

const (
	StateMenu GameState = iota
	StatePlaying
)

var (
	playerSpriteBoy  *ebiten.Image
	playerSpriteGirl *ebiten.Image
	tilesetImage     *ebiten.Image
	audioContext     *audio.Context
	bgmPlayer        *audio.Player
)

const (
	mapWidth  = 20
	mapHeight = 15
	tileSize  = 32
)

func init() {
	var err error
	playerSpriteBoy, _, err = ebitenutil.NewImageFromFile("assets/graphics/characters/boy_run.png")
	if err != nil {
		log.Printf("Failed to load boy sprite: %v", err)
	}

	playerSpriteGirl, _, err = ebitenutil.NewImageFromFile("assets/graphics/characters/girl_run.png")
	if err != nil {
		log.Printf("Failed to load girl sprite: %v", err)
	}

	tilesetImage, _, err = ebitenutil.NewImageFromFile("assets/graphics/tilesets/Outside.png")
	if err != nil {
		log.Printf("Failed to load tileset: %v", err)
	}

	audioContext = audio.NewContext(44100)
	f, err := os.Open("assets/audio/bgm/Title.ogg")
	if err == nil {
		s, err := vorbis.DecodeWithSampleRate(44100, f)
		if err == nil {
			loop := audio.NewInfiniteLoop(s, s.Length())
			bgmPlayer, _ = audioContext.NewPlayer(loop)
			bgmPlayer.Play()
		}
	}
}

type Rect struct {
	X, Y, W, H float64
}

var houses = []Rect{
	{36, 30, 68, 50},
	{400, 30, 68, 50},
	{36, 350, 68, 50},
	{400, 350, 68, 50},
}

func isColliding(x, y float64) bool {
	pw, ph := 32.0, 32.0
	if x < 0 || x+pw > mapWidth*tileSize || y < 0 || y+ph > mapHeight*tileSize {
		return true
	}
	for _, h := range houses {
		if x < h.X+h.W && x+pw > h.X && y < h.Y+h.H && y+ph > h.Y {
			return true
		}
	}
	return false
}

type Game struct {
	state      GameState
	isGirl     bool
	x, y       float64
	dir        int
	step       int
	tick       int
	camX, camY float64
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}

	if g.state == StateMenu {
		if inpututil.IsKeyJustPressed(ebiten.KeyLeft) || inpututil.IsKeyJustPressed(ebiten.Key1) {
			g.isGirl = false
			g.state = StatePlaying
		} else if inpututil.IsKeyJustPressed(ebiten.KeyRight) || inpututil.IsKeyJustPressed(ebiten.Key2) {
			g.isGirl = true
			g.state = StatePlaying
		}
		return nil
	}

	newX, newY := g.x, g.y
	moved := false

	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		newX -= 2
		g.dir = 1
		moved = true
	} else if ebiten.IsKeyPressed(ebiten.KeyRight) {
		newX += 2
		g.dir = 2
		moved = true
	} else if ebiten.IsKeyPressed(ebiten.KeyUp) {
		newY -= 2
		g.dir = 3
		moved = true
	} else if ebiten.IsKeyPressed(ebiten.KeyDown) {
		newY += 2
		g.dir = 0
		moved = true
	}

	if moved {
		g.tick++
		if g.tick > 8 {
			g.step = (g.step + 1) % 4
			g.tick = 0
		}
	} else {
		g.step = 0
	}

	if newX != g.x && !isColliding(newX, g.y) {
		g.x = newX
	}
	if newY != g.y && !isColliding(g.x, newY) {
		g.y = newY
	}

	g.camX = g.x - 160 + 16
	g.camY = g.y - 120 + 24

	if g.camX < 0 {
		g.camX = 0
	}
	if g.camY < 0 {
		g.camY = 0
	}
	if g.camX > mapWidth*tileSize-320 {
		g.camX = mapWidth*tileSize - 320
	}
	if g.camY > mapHeight*tileSize-240 {
		g.camY = mapHeight*tileSize - 240
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.state == StateMenu {
		screen.Fill(color.NRGBA{30, 40, 60, 255})
		var msg string
		msg += "Welcome to Pokemon Prototype\n\n"
		msg += "Choose Your Character:\n"
		msg += "[1] or [Left Arrow] for Boy\n"
		msg += "[2] or [Right Arrow] for Girl"
		ebitenutil.DebugPrintAt(screen, msg, 60, 50)

		if playerSpriteBoy != nil {
			opBoy := &ebiten.DrawImageOptions{}
			opBoy.GeoM.Translate(100, 150)
			subBoy := playerSpriteBoy.SubImage(image.Rect(0, 0, 32, 48)).(*ebiten.Image)
			screen.DrawImage(subBoy, opBoy)
		}

		if playerSpriteGirl != nil {
			opGirl := &ebiten.DrawImageOptions{}
			opGirl.GeoM.Translate(180, 150)
			subGirl := playerSpriteGirl.SubImage(image.Rect(0, 0, 32, 48)).(*ebiten.Image)
			screen.DrawImage(subGirl, opGirl)
		}
		return
	}

	op := &ebiten.DrawImageOptions{}
	if tilesetImage != nil {
		for y := 0; y < mapHeight; y++ {
			for x := 0; x < mapWidth; x++ {
				tileID := 16
				if y == 7 || x == 10 {
					tileID = 17
				}
				vx := float64(x*tileSize) - g.camX
				vy := float64(y*tileSize) - g.camY
				if vx < -tileSize || vy < -tileSize || vx > 320 || vy > 240 {
					continue
				}
				sx := (tileID % 8) * tileSize
				sy := (tileID / 8) * tileSize
				op.GeoM.Reset()
				op.GeoM.Translate(vx, vy)
				sub := tilesetImage.SubImage(image.Rect(sx, sy, sx+tileSize, sy+tileSize)).(*ebiten.Image)
				screen.DrawImage(sub, op)
			}
		}
	} else {
		screen.Fill(color.NRGBA{60, 179, 113, 255})
	}

	op.GeoM.Reset()
	screenX := g.x - g.camX
	screenY := g.y - g.camY - 16
	op.GeoM.Translate(screenX, screenY)

	activeSprite := playerSpriteBoy
	if g.isGirl && playerSpriteGirl != nil {
		activeSprite = playerSpriteGirl
	}

	if activeSprite != nil {
		sx := g.step * 32
		sy := g.dir * 48
		sub := activeSprite.SubImage(image.Rect(sx, sy, sx+32, sy+48)).(*ebiten.Image)
		screen.DrawImage(sub, op)
	}
}

func (g *Game) Layout(w, h int) (int, int) {
	return 320, 240
}

func main() {
	game := &Game{
		state: StateMenu,
		x:     144,
		y:     104,
		dir:   0,
	}
	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Pokemon Master - Prototype")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
