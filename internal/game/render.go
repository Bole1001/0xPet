package game

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"0xPet/internal/monitor"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

// updateEffects 处理每一帧的数据同步与故障特效计算
func (g *Manager) updateEffects() {
	isMoving := g.isDragging || math.Abs(g.velX) > 0.1 || math.Abs(g.velY) > 0.1

	// 1. 还原字符网格状态
	for r := range g.MyPet.Grid {
		for c := range g.MyPet.Grid[r] {
			g.MyPet.Grid[r][c].Char = g.MyPet.Grid[r][c].OriginalChar
		}
	}

	// 2. 触发故障特效 (仅在静止且开关开启时)
	if g.ShowGlitch && !isMoving {
		if rand.Intn(100) < 10 { // 10% 概率触发
			rows := len(g.MyPet.Grid)
			glitchCount := 5 + rand.Intn(5)
			for i := 0; i < glitchCount; i++ {
				r := rand.Intn(rows)
				if len(g.MyPet.Grid[r]) > 0 {
					c := rand.Intn(len(g.MyPet.Grid[r]))
					chars := []string{"?", "#", "$", "&", "0", "1", "!"}
					g.MyPet.Grid[r][c].Char = chars[rand.Intn(len(chars))]
				}
			}
		}
	}

	// 3. 数据同步 (CPU/内存)
	cpu, mem := monitor.GetStats()
	g.MyPet.CPUUsage = cpu
	g.MyPet.MemUsage = mem
	g.MyPet.IsStressed = cpu > 80.0
}

// drawPet 渲染宠物主体与监控 HUD
func (g *Manager) drawPet(screen *ebiten.Image) {
	isMoving := g.isDragging || math.Abs(g.velX) > 0.1 || math.Abs(g.velY) > 0.1
	offsetY := 0.0

	if g.ShowAnimation && !isMoving {
		offsetY = math.Sin(g.tick*0.05) * 5
	}

	baseY := 30.0 + offsetY

	// 【新增】根据模式选择正确的字库与间距
	var currentFont font.Face
	var fontW, fontH float64
	if g.DisplayMode == 0 {
		currentFont = g.FontNormal
		fontW, fontH = 7.0, 13.0
	} else {
		currentFont = g.FontSmall
		fontW, fontH = 3.5, 6.5
	}

	for r, row := range g.MyPet.Grid {
		for c, charData := range row {
			x := float64(c) * fontW
			y := float64(r)*fontH + baseY

			var drawColor color.Color
			if g.MyPet.IsStressed {
				drawColor = color.RGBA{255, 50, 50, 255}
			} else if g.ShowColor {
				drawColor = charData.Color
			} else {
				drawColor = color.RGBA{0, 255, 0, 255}
			}

			// 【关键】废弃 basicfont，使用当前模式的高清矢量字库
			text.Draw(screen, charData.Char, currentFont, int(x), int(y), drawColor)
		}
	}

	if g.ShowMonitor && !isMoving {
		msg := fmt.Sprintf("CPU: %.0f%% | MEM: %.0f%%", g.MyPet.CPUUsage, g.MyPet.MemUsage)
		// HUD 永远使用正常大小的字体，保证可读性
		text.Draw(screen, msg, g.FontNormal, 0, 15, color.RGBA{255, 255, 0, 255})
	}
}

// drawMenu 渲染侧滑抽屉菜单
func (g *Manager) drawMenu(screen *ebiten.Image) {
	w, h := screen.Size()

	// 【核心逻辑】菜单始终吸附在当前窗口的最右侧边缘
	// 因为 w 在 ui.go 中被动态拉宽，这会产生完美的抽出动画
	menuX := float32(w) - MenuWidth

	// 1. 画抽屉背景面板
	bgColor := color.RGBA{5, 5, 10, 245}
	vector.DrawFilledRect(screen, menuX, 0, MenuWidth, float32(h), bgColor, false)

	// 顶部能量条
	accentColor := color.RGBA{0, 255, 255, 255}
	vector.DrawFilledRect(screen, menuX, 0, MenuWidth, 4, accentColor, false)

	// 2. 绘制按钮组 (严格映射 ui.go 中的物理碰撞坐标)
	btnW := float32(MenuWidth - 40)
	btnH := float32(30)
	baseBtnX := menuX + 20
	startBtnY := float32(50)
	gap := float32(15)

	textCol := color.RGBA{200, 200, 200, 255}
	btnBg := color.RGBA{30, 30, 40, 255}
	exitBg := color.RGBA{180, 40, 40, 255}

	drawCenteredText := func(txt string, y int, col color.Color) {
		textW := len(txt) * 7
		textX := int(baseBtnX) + (int(btnW)-textW)/2
		text.Draw(screen, txt, basicfont.Face7x13, textX, y+20, col)
	}

	// Color Mode
	vector.DrawFilledRect(screen, baseBtnX, startBtnY, btnW, btnH, btnBg, false)
	statusText, statusCol := "COLOR: OFF", textCol
	if g.ShowColor {
		statusText, statusCol = "COLOR: ON", accentColor
		vector.DrawFilledRect(screen, baseBtnX, startBtnY, 4, btnH, accentColor, false)
	}
	drawCenteredText(statusText, int(startBtnY), statusCol)

	// Glitch Effect
	y2 := startBtnY + btnH + gap
	vector.DrawFilledRect(screen, baseBtnX, y2, btnW, btnH, btnBg, false)
	statusText, statusCol = "GLITCH: OFF", textCol
	if g.ShowGlitch {
		statusText, statusCol = "GLITCH: ON", accentColor
		vector.DrawFilledRect(screen, baseBtnX, y2, 4, btnH, accentColor, false)
	}
	drawCenteredText(statusText, int(y2), statusCol)

	// Float Animation
	y3 := y2 + btnH + gap
	vector.DrawFilledRect(screen, baseBtnX, y3, btnW, btnH, btnBg, false)
	statusText, statusCol = "FLOAT: OFF", textCol
	if g.ShowAnimation {
		statusText, statusCol = "FLOAT: ON", accentColor
		vector.DrawFilledRect(screen, baseBtnX, y3, 4, btnH, accentColor, false)
	}
	drawCenteredText(statusText, int(y3), statusCol)

	// Monitor HUD
	y4 := y3 + btnH + gap
	vector.DrawFilledRect(screen, baseBtnX, y4, btnW, btnH, btnBg, false)
	statusText, statusCol = "HUD: OFF", textCol
	if g.ShowMonitor {
		statusText, statusCol = "HUD: ON", accentColor
		vector.DrawFilledRect(screen, baseBtnX, y4, 4, btnH, accentColor, false)
	}
	drawCenteredText(statusText, int(y4), statusCol)

	// DISPLAY MODE
	y5 := y4 + btnH + gap
	vector.DrawFilledRect(screen, baseBtnX, y5, btnW, btnH, btnBg, false)
	modeText := "MODE: NORMAL"
	if g.DisplayMode == 1 {
		modeText = "MODE: HI-RES"
	} else if g.DisplayMode == 2 {
		modeText = "MODE: MINI"
	}
	drawCenteredText(modeText, int(y5), accentColor)
	vector.DrawFilledRect(screen, baseBtnX, y5, 4, btnH, accentColor, false)

	// EXIT
	yExit := float32(h) - 50
	vector.DrawFilledRect(screen, baseBtnX, yExit, btnW, btnH, exitBg, false)
	drawCenteredText(">> SYSTEM EXIT <<", int(yExit), color.White)
}
