package game

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font"
)

// updatePetCanvas 核心渲染引擎：仅在状态脏化时执行高昂的逐字绘制
func (g *Manager) updatePetCanvas() {
	if g.MyPet.Width <= 0 || g.MyPet.Height <= 0 {
		return
	}

	// 1. 动态重建或清空物理画布
	if g.petCanvas == nil || g.petCanvas.Bounds().Dx() != g.MyPet.Width || g.petCanvas.Bounds().Dy() != g.MyPet.Height {
		g.petCanvas = ebiten.NewImage(g.MyPet.Width, g.MyPet.Height)
	}
	g.petCanvas.Clear()

	// 2. 解析字体基准
	var currentFont font.Face
	var fontW, fontH float64
	if g.DisplayMode == 0 {
		currentFont = g.FontNormal
		fontW, fontH = 8.0, 16.0
	} else {
		currentFont = g.FontSmall
		fontW, fontH = 4.0, 8.0
	}

	// 3. 将所有字符烤制到 petCanvas 上 (注意：取消了 baseY 偏移，直接从 0,0 开始画)
	for r, row := range g.MyPet.Grid {
		for c, charData := range row {
			x := float64(c) * fontW
			y := float64(r) * fontH

			var drawColor color.Color
			if g.MyPet.IsStressed {
				drawColor = color.RGBA{255, 50, 50, 255}
			} else if g.ShowColor {
				rc, gc, bc, ac := charData.Color.RGBA()
				boost := func(v uint32) uint8 {
					val := float64(v>>8) * 1.3
					if val > 255 {
						return 255
					}
					return uint8(val)
				}
				drawColor = color.RGBA{boost(rc), boost(gc), boost(bc), uint8(ac >> 8)}
			} else {
				drawColor = color.RGBA{0, 255, 0, 255}
			}

			r32, g32, b32, _ := drawColor.RGBA()
			r8, g8, b8 := r32>>8, g32>>8, b32>>8
			luminance := (r8*299 + g8*587 + b8*114) / 1000

			if luminance > 70 && charData.Char != " " {
				shadowColor := color.RGBA{0, 0, 0, 140}
				text.Draw(g.petCanvas, charData.Char, currentFont, int(x)+1, int(y)+1, shadowColor)
			}
			text.Draw(g.petCanvas, charData.Char, currentFont, int(x), int(y), drawColor)
		}
	}

	// 4. 解除脏标记
	g.isDirty = false
}

// drawPet 极速渲染通道：静态底图 O(1) 绘制 + 乱码增量 O(N) 覆写
func (g *Manager) drawPet(screen *ebiten.Image) {
	// 【关键修正 1】将 ShowGlitch 从全量重绘触发器中剥离！
	// 只有在切换模式、改颜色、或初始化时才允许重绘 30 万次
	if g.isDirty || g.petCanvas == nil {
		g.updatePetCanvas()
	}

	isMoving := g.isDragging || math.Abs(g.velX) > 0.1 || math.Abs(g.velY) > 0.1

	// 1. 极致性能：单次 API 调用，把烤好的整张静态宠物贴图拍在屏幕上
	if g.petCanvas != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(0, 30.0)
		screen.DrawImage(g.petCanvas, op)
	}

	// 2. 独立 HUD 渲染
	if g.ShowMonitor && !isMoving {
		msg := fmt.Sprintf("CPU: %.0f%% | MEM: %.0f%%", g.MyPet.CPUUsage, g.MyPet.MemUsage)
		text.Draw(screen, msg, g.FontNormal, 0, 15, color.RGBA{255, 255, 0, 255})
	}
}

func (g *Manager) buildMenuCanvas(height int) {
	if height <= 0 {
		return
	}

	if g.menuCanvas == nil || g.menuCanvas.Bounds().Dy() != height {
		g.menuCanvas = ebiten.NewImage(MenuWidth, height)
	}
	g.menuCanvas.Clear()

	bgColor := color.RGBA{8, 8, 12, 230}
	vector.DrawFilledRect(g.menuCanvas, 0, 0, float32(MenuWidth), float32(height), bgColor, false)
	vector.DrawFilledRect(g.menuCanvas, 0, 0, 2, float32(height), color.RGBA{0, 255, 255, 255}, false)

	type menuItem struct {
		label string
		state bool
	}
	items := []menuItem{
		{"COLOR", g.ShowColor},
		{"HUD", g.ShowMonitor},
		{"MODE", false},
		{"EXIT", false},
	}

	baseTextX := 15
	menuFont := g.FontNormal

	for i, item := range items {
		textY := StartY + i*RowHeight + 18
		var symbol string
		var drawCol color.Color = color.RGBA{180, 180, 190, 255}

		if item.state {
			symbol = "[*]"
			drawCol = color.RGBA{0, 255, 255, 255}
		} else {
			symbol = "[ ]"
		}

		if item.label == "MODE" {
			modes := []string{"NORMAL", "HI-RES", "MINI"}
			item.label = item.label + ": " + modes[g.DisplayMode]
			symbol = "[~]"
			drawCol = color.RGBA{220, 220, 80, 255}
		} else if item.label == "EXIT" {
			symbol = "[!]"
			drawCol = color.RGBA{255, 100, 100, 255} // 警示红
		} else {
			if item.state {
				symbol = "[*]"
				drawCol = color.RGBA{0, 255, 255, 255} // 高亮青
			} else {
				symbol = "[ ]"
			}
		}

		fullText := fmt.Sprintf("%s %s", symbol, item.label)
		text.Draw(g.menuCanvas, fullText, menuFont, baseTextX, textY, drawCol)
	}

	g.menuDirty = false
}

func (g *Manager) drawMenu(screen *ebiten.Image) {
	w, h := screen.Size()
	menuW := int(float64(MenuWidth) * g.menuAnim)
	if menuW <= 0 {
		return
	}

	if g.menuCanvas == nil || g.menuCanvas.Bounds().Dy() != h || g.menuDirty {
		g.buildMenuCanvas(h)
	}
	if g.menuCanvas == nil {
		return
	}

	subMenu, ok := g.menuCanvas.SubImage(image.Rect(0, 0, menuW, h)).(*ebiten.Image)
	if !ok {
		return
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(w-menuW), 0)
	screen.DrawImage(subMenu, op)
}
