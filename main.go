package main

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type GameState int

const (
	StateTitle GameState = iota
	StateMenu
	StatePlaying
	StatePokedex
	StateBattle
)

var (
	playerSpriteBoy  *ebiten.Image
	playerSpriteGirl *ebiten.Image
	pikachuSprite    *ebiten.Image
	tilesetImage     *ebiten.Image
	audioContext     *audio.Context
	bgmPlayer        *audio.Player
	pokedex          []string
)

const tileSize = 32

type Rect struct {
	X, Y, W, H float64
}

type MapData struct {
	Width, Height int
	BaseTile      int
	PathTile      int
	TallGrassTile int
	Houses        []Rect
	GrassRects    []Rect
	LeftMap       int
	RightMap      int
	UpMap         int
	DownMap       int
}

var maps = []MapData{
	{ // Map 0 (Start Town)
		Width: 20, Height: 15,
		BaseTile: 16, PathTile: 17, TallGrassTile: 24,
		Houses: []Rect{
			{36, 30, 68, 50},
			{400, 30, 68, 50},
			{36, 350, 68, 50},
			{400, 350, 68, 50},
		},
		GrassRects: []Rect{
			{100, 100, 160, 128}, // Patch of tall grass below the top left house
		},
		LeftMap: -1, RightMap: 1, UpMap: -1, DownMap: -1,
	},
	{ // Map 1 (Route 1 / Second Town)
		Width: 30, Height: 20,
		BaseTile: 55, PathTile: 56, TallGrassTile: 63,
		Houses: []Rect{
			{200, 100, 68, 50},
			{600, 100, 68, 50},
			{700, 400, 68, 50},
		},
		GrassRects: []Rect{
			{300, 200, 256, 128},
			{100, 400, 128, 192},
		},
		LeftMap: 0, RightMap: -1, UpMap: -1, DownMap: -1,
	},
}

type NPC struct {
	X, Y   float64
	MapIdx int
	Dir    int
	Step   int
}

var npcs []NPC

func loadPokedexNames() []string {
	pbsPath := "extracted_orphan/Pokemon Orphan main/PBS/pokemon.txt"
	f, err := os.Open(pbsPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var names []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			name := line[1 : len(line)-1]
			names = append(names, name)
			if len(names) == 649 {
				break
			}
		}
	}
	return names
}

func init() {
	rand.Seed(time.Now().UnixNano())

	var err error
	pokedex = loadPokedexNames()

	playerSpriteBoy, _, err = ebitenutil.NewImageFromFile("assets/graphics/characters/boy_run.png")
	if err != nil {
		log.Printf("Failed to load boy sprite: %v", err)
	}

	playerSpriteGirl, _, err = ebitenutil.NewImageFromFile("assets/graphics/characters/girl_run.png")
	if err != nil {
		log.Printf("Failed to load girl sprite: %v", err)
	}

	pikachuSprite, _, err = ebitenutil.NewImageFromFile("extracted_orphan/Pokemon Orphan main/Graphics/Characters/Followers/PIKACHU.png")
	if err != nil {
		log.Printf("Failed to load pikachu sprite: %v", err)
	} else {
		npcs = append(npcs, NPC{
			X:      200,
			Y:      50,
			MapIdx: 0,
			Dir:    0,
			Step:   0,
		})
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

func isColliding(x, y float64, mapIdx int) bool {
	pw, ph := 32.0, 32.0
	m := &maps[mapIdx]

	if x < 0 && m.LeftMap == -1 {
		return true
	}
	if x+pw > float64(m.Width*tileSize) && m.RightMap == -1 {
		return true
	}
	if y < 0 && m.UpMap == -1 {
		return true
	}
	if y+ph > float64(m.Height*tileSize) && m.DownMap == -1 {
		return true
	}

	for _, h := range m.Houses {
		if x < h.X+h.W && x+pw > h.X && y < h.Y+h.H && y+ph > h.Y {
			return true
		}
	}
	return false
}

func inGrass(x, y float64, m *MapData) bool {
	for _, r := range m.GrassRects {
		if x >= r.X && x <= r.X+r.W && y >= r.Y && y <= r.Y+r.H {
			return true
		}
	}
	return false
}

type Game struct {
	state         GameState
	isGirl        bool
	currentMapIdx int
	x, y          float64
	dir           int
	step          int
	tick          int
	camX, camY    float64
	pokedexIdx    int
	pokedexImages map[string]*ebiten.Image
	enemyName     string
	enemyHP       int
	enemyMaxHP    int
	playerHP      int
	playerMaxHP   int
}

func (g *Game) getPokedexImage(name string) *ebiten.Image {
	if g.pokedexImages == nil {
		g.pokedexImages = make(map[string]*ebiten.Image)
	}
	if img, ok := g.pokedexImages[name]; ok {
		return img
	}
	path := "assets/pokemon/front/" + name + ".png"
	img, _, err := ebitenutil.NewImageFromFile(path)
	if err != nil {
		img = ebiten.NewImage(1, 1)
	}
	g.pokedexImages[name] = img
	return img
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}

	if g.state == StateTitle {
		g.tick++
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			g.state = StateMenu
			g.tick = 0
		}
		return nil
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

	if g.state == StatePokedex {
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyP) {
			g.state = StatePlaying
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
			g.pokedexIdx = (g.pokedexIdx + 1) % len(pokedex)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
			g.pokedexIdx--
			if g.pokedexIdx < 0 {
				g.pokedexIdx = len(pokedex) - 1
			}
		}
		return nil
	}

	if g.state == StateBattle {
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyX) {
			g.state = StatePlaying
		}
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			damage := rand.Intn(10) + 5
			g.enemyHP -= damage
			if g.enemyHP <= 0 {
				g.enemyHP = 0
				g.state = StatePlaying // Win battle
			} else {
				// Enemy retaliates
				eDamage := rand.Intn(8) + 3
				g.playerHP -= eDamage
				if g.playerHP < 0 {
					g.playerHP = 0
				}
			}
		}
		return nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyP) && len(pokedex) > 0 {
		g.state = StatePokedex
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

			// Wild Encounter check
			m := &maps[g.currentMapIdx]
			if inGrass(g.x+16, g.y+24, m) {
				if rand.Float64() < 0.15 && len(pokedex) > 0 {
					g.state = StateBattle
					g.enemyName = pokedex[rand.Intn(len(pokedex))]
					g.enemyMaxHP = rand.Intn(30) + 20
					g.enemyHP = g.enemyMaxHP
					if g.playerMaxHP == 0 {
						g.playerMaxHP = 50
						g.playerHP = 50
					}
					return nil
				}
			}
		}
	} else {
		g.step = 0
	}

	m := &maps[g.currentMapIdx]
	pw, ph := 32.0, 32.0

	// Map transitions
	if newX < 0 && m.LeftMap != -1 {
		g.currentMapIdx = m.LeftMap
		nextMap := &maps[g.currentMapIdx]
		g.x = float64(nextMap.Width*tileSize) - pw - 2
		if g.y > float64(nextMap.Height*tileSize)-ph {
			g.y = float64(nextMap.Height*tileSize) - ph
		}
		return nil
	}
	if newX+pw > float64(m.Width*tileSize) && m.RightMap != -1 {
		g.currentMapIdx = m.RightMap
		nextMap := &maps[g.currentMapIdx]
		g.x = 2
		if g.y > float64(nextMap.Height*tileSize)-ph {
			g.y = float64(nextMap.Height*tileSize) - ph
		}
		return nil
	}
	if newY < 0 && m.UpMap != -1 {
		g.currentMapIdx = m.UpMap
		nextMap := &maps[g.currentMapIdx]
		g.y = float64(nextMap.Height*tileSize) - ph - 2
		if g.x > float64(nextMap.Width*tileSize)-pw {
			g.x = float64(nextMap.Width*tileSize) - pw
		}
		return nil
	}
	if newY+ph > float64(m.Height*tileSize) && m.DownMap != -1 {
		g.currentMapIdx = m.DownMap
		nextMap := &maps[g.currentMapIdx]
		g.y = 2
		if g.x > float64(nextMap.Width*tileSize)-pw {
			g.x = float64(nextMap.Width*tileSize) - pw
		}
		return nil
	}

	if newX != g.x && !isColliding(newX, g.y, g.currentMapIdx) {
		g.x = newX
	}
	if newY != g.y && !isColliding(g.x, newY, g.currentMapIdx) {
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
	maxCamX := float64(m.Width*tileSize - 320)
	if maxCamX < 0 {
		maxCamX = 0
	}
	if g.camX > maxCamX {
		g.camX = maxCamX
	}

	maxCamY := float64(m.Height*tileSize - 240)
	if maxCamY < 0 {
		maxCamY = 0
	}
	if g.camY > maxCamY {
		g.camY = maxCamY
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.state == StateTitle {
		screen.Fill(color.NRGBA{70, 130, 180, 255})
		ebitenutil.DebugPrintAt(screen, "==========================", 80, 80)
		ebitenutil.DebugPrintAt(screen, "   POKEMON MASTER: GO    ", 80, 100)
		ebitenutil.DebugPrintAt(screen, "==========================", 80, 120)

		if (g.tick/30)%2 == 0 {
			ebitenutil.DebugPrintAt(screen, "> Press ENTER to Start <", 90, 160)
		}
		return
	}

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

	if g.state == StateBattle {
		screen.Fill(color.NRGBA{240, 248, 255, 255}) // AliceBlue

		// Draw Enemy UI
		ebitenutil.DebugPrintAt(screen, "LV 5  " + g.enemyName, 20, 20)
		ebitenutil.DrawRect(screen, 20, 36, 120, 12, color.NRGBA{100, 100, 100, 255})
		hpRatioEnemy := float64(g.enemyHP) / float64(g.enemyMaxHP)
		hpColorE := color.NRGBA{0, 200, 0, 255}
		if hpRatioEnemy < 0.2 {
			hpColorE = color.NRGBA{200, 0, 0, 255}
		} else if hpRatioEnemy < 0.5 {
			hpColorE = color.NRGBA{200, 200, 0, 255}
		}
		ebitenutil.DrawRect(screen, 20, 36, 120*hpRatioEnemy, 12, hpColorE)
		ebitenutil.DebugPrintAt(screen, "Press SPACE to attack, X to run", 20, 200)

		// Draw enemy sprite
		img := g.getPokedexImage(g.enemyName)
		op := &ebiten.DrawImageOptions{}
		// Move enemy near top right
		op.GeoM.Translate(180, 20)
		screen.DrawImage(img, op)

		// Draw Player UI
		ebitenutil.DebugPrintAt(screen, "LV 5  PIKACHU", 160, 130)
		ebitenutil.DrawRect(screen, 160, 146, 140, 12, color.NRGBA{100, 100, 100, 255})
		hpRatioPlayer := float64(g.playerHP) / float64(g.playerMaxHP)
		hpColorP := color.NRGBA{0, 200, 0, 255}
		if hpRatioPlayer < 0.2 {
			hpColorP = color.NRGBA{200, 0, 0, 255}
		} else if hpRatioPlayer < 0.5 {
			hpColorP = color.NRGBA{200, 200, 0, 255}
		}
		ebitenutil.DrawRect(screen, 160, 146, 140*hpRatioPlayer, 12, hpColorP)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%d/%d", g.playerHP, g.playerMaxHP), 240, 162)

		// Draw player's back sprite at bottom left
		opP := &ebiten.DrawImageOptions{}
		opP.GeoM.Translate(40, 120)
		activeSprite := playerSpriteBoy
		if g.isGirl && playerSpriteGirl != nil {
			activeSprite = playerSpriteGirl
		}
		if activeSprite != nil {
			sx := 0
			// 3 * 48 = 144 (Up direction, mimicking back view)
			sy := 144
			sub := activeSprite.SubImage(image.Rect(sx, sy, sx+32, sy+48)).(*ebiten.Image)
			screen.DrawImage(sub, opP)
		}
		return
	}

	if g.state == StatePokedex {
		screen.Fill(color.NRGBA{200, 40, 40, 255})
		ebitenutil.DebugPrintAt(screen, "--- POKEDEX ---", 20, 20)
		ebitenutil.DebugPrintAt(screen, "Press P or ESC to close", 20, 36)
		ebitenutil.DebugPrintAt(screen, "Use UP/DOWN to scroll", 20, 52)

		for i := -3; i <= 3; i++ {
			idx := g.pokedexIdx + i
			if idx >= 0 && idx < len(pokedex) {
				y := 100 + i*20
				prefix := "  "
				if i == 0 {
					prefix = ">> "
				}
				line := fmt.Sprintf("%s%03d: %s", prefix, idx+1, pokedex[idx])
				ebitenutil.DebugPrintAt(screen, line, 20, y)
			}
		}

		selectedName := pokedex[g.pokedexIdx]
		pImg := g.getPokedexImage(selectedName)
		opSprite := &ebiten.DrawImageOptions{}
		opSprite.GeoM.Translate(180, 80)
		screen.DrawImage(pImg, opSprite)
		return
	}

	// StatePlaying rendering
	m := &maps[g.currentMapIdx]
	op := &ebiten.DrawImageOptions{}

	if tilesetImage != nil {
		for y := 0; y < m.Height; y++ {
			for x := 0; x < m.Width; x++ {
				tileID := m.BaseTile
				if y == m.Height/2 || x == m.Width/2 {
					tileID = m.PathTile
				} else {
					tileX := float64(x * tileSize) + 16
					tileY := float64(y * tileSize) + 16
					if inGrass(tileX, tileY, m) {
						tileID = m.TallGrassTile
					}
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

	for _, npc := range npcs {
		if npc.MapIdx == g.currentMapIdx && pikachuSprite != nil {
			opNpc := &ebiten.DrawImageOptions{}
			npcScreenX := npc.X - g.camX
			npcScreenY := npc.Y - g.camY - 16
			opNpc.GeoM.Translate(npcScreenX, npcScreenY)
			
			nsx := npc.Step * 32
			nsy := npc.Dir * 48
			rect := image.Rect(nsx, nsy, nsx+32, nsy+48)
			if rect.Max.X <= pikachuSprite.Bounds().Max.X && rect.Max.Y <= pikachuSprite.Bounds().Max.Y {
				nSub := pikachuSprite.SubImage(rect).(*ebiten.Image)
				screen.DrawImage(nSub, opNpc)
			}
		}
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
	
	ebitenutil.DebugPrintAt(screen, "Press 'P': Pokedex | Walk in darker patches for wild pokemon", 5, 220)
}

func (g *Game) Layout(w, h int) (int, int) {
	return 320, 240
}

func main() {
	game := &Game{
		state:         StateTitle,
		currentMapIdx: 0,
		x:             144,
		y:             104,
		dir:           0,
	}
	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Pokemon Master - Prototype")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
