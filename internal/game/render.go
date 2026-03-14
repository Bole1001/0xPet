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
				// 解析原图色彩，并强行提升 30% 的亮度，防止深色在透明背景下隐形
				r, gg, b, a := charData.Color.RGBA()
				boost := func(v uint32) uint8 {
					val := float64(v>>8) * 1.3
					if val > 255 {
						return 255
					}
					return uint8(val)
				}
				drawColor = color.RGBA{boost(r), boost(gg), boost(b), uint8(a >> 8)}
			} else {
				drawColor = color.RGBA{0, 255, 0, 255} // 默认纯绿
			}

			// 1. 提取最终要绘制颜色的 8位 RGB 值
			r32, g32, b32, _ := drawColor.RGBA()
			r8, g8, b8 := r32>>8, g32>>8, b32>>8

			// 2. 运用标准 sRGB 亮度公式计算人眼感知亮度 (范围: 0~255)
			luminance := (r8*299 + g8*587 + b8*114) / 1000

			// 3. 拦截过滤：
			// 只有当字符亮度 > 70（不是深暗色），并且它不是一个不可见的空格时，才允许绘制保护性阴影
			if luminance > 70 && charData.Char != " " {
				// 将阴影的 Alpha 值稍微调柔和（180 -> 140），减少生硬的切割感
				shadowColor := color.RGBA{0, 0, 0, 140}
				text.Draw(screen, charData.Char, currentFont, int(x)+1, int(y)+1, shadowColor)
			}

			// 4. 无论如何，叠加绘制主体字符本身
			text.Draw(screen, charData.Char, currentFont, int(x), int(y), drawColor)
		}
	}

	if g.ShowMonitor && !isMoving {
		msg := fmt.Sprintf("CPU: %.0f%% | MEM: %.0f%%", g.MyPet.CPUUsage, g.MyPet.MemUsage)
		// HUD 永远使用正常大小的字体，保证可读性
		text.Draw(screen, msg, g.FontNormal, 0, 15, color.RGBA{255, 255, 0, 255})
	}
}

// drawMenu 渲染复古控制台风格的紧凑型菜单
func (g *Manager) drawMenu(screen *ebiten.Image) {
	w, h := screen.Size()
	menuX := float32(w - MenuWidth)

	// 1. 绘制极简半透明背景
	bgColor := color.RGBA{8, 8, 12, 230}
	vector.DrawFilledRect(screen, menuX, 0, float32(MenuWidth), float32(h), bgColor, false)

	// 左侧高亮分割线
	accentColor := color.RGBA{0, 255, 255, 255} // 青色
	vector.DrawFilledRect(screen, menuX, 0, 2, float32(h), accentColor, false)

	// 2. 定义渲染结构
	type menuItem struct {
		label string
		state bool
		mType int // 0=普通开关, 1=模式切换, 2=退出
	}
	items := []menuItem{
		{"COLOR", g.ShowColor, 0},
		{"GLITCH", g.ShowGlitch, 0},
		{"FLOAT", g.ShowAnimation, 0},
		{"HUD", g.ShowMonitor, 0},
		{"MODE", false, 1},
		{"EXIT", false, 2},
	}

	baseTextX := int(menuX) + 15
	// UI 永远使用大字号，确保可读性
	menuFont := g.FontNormal

	// 3. 逐行渲染
	for i, item := range items {
		// 行的垂直中心偏下对齐
		textY := StartY + i*RowHeight + 18

		var symbol string
		var drawCol color.Color = color.RGBA{180, 180, 190, 255} // 默认暗白

		if item.mType == 2 {
			// 退出按钮
			symbol = "[!]"
			drawCol = color.RGBA{255, 100, 100, 255} // 红色
		} else if item.mType == 1 {
			// 模式切换按钮
			modes := []string{"NORMAL", "HI-RES", "MINI"}
			symbol = "[~]"
			item.label = item.label + ": " + modes[g.DisplayMode]
			drawCol = color.RGBA{220, 220, 80, 255} // 黄色
		} else {
			// 普通开关
			if item.state {
				symbol = "[*]"
				drawCol = accentColor // 开启时高亮青色
			} else {
				symbol = "[ ]"
			}
		}

		// 格式化输出，例如：[*] COLOR   或   [~] MODE: MINI
		fullText := fmt.Sprintf("%s %s", symbol, item.label)
		text.Draw(screen, fullText, menuFont, baseTextX, textY, drawCol)
	}
}
