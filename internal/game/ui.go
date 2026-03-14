package game

import (
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// 采用全局恒定的紧凑型菜单尺寸
const (
	MenuWidth = 160 // 宽度收紧，足够放下13px的纯文本
	RowHeight = 25  // 每行的高度，无额外 Gap
	StartY    = 20  // 顶部留白
	MinMenuH  = 180 // 菜单的最小安全高度 (20 + 6*25 + 10底部留白)
)

func (g *Manager) handleUIInput() {
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		g.ShowColor = !g.ShowColor
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyG) {
		g.ShowGlitch = !g.ShowGlitch
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyA) {
		g.ShowAnimation = !g.ShowAnimation
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		g.ShowMonitor = !g.ShowMonitor
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		g.ShowMenu = !g.ShowMenu
		g.velX, g.velY = 0, 0
	}

	if g.ShowMenu && !ebiten.IsFocused() {
		g.ShowMenu = false
		g.velX, g.velY = 0, 0
	}

	speed := 0.1
	if g.ShowMenu {
		if g.menuAnim < 1.0 {
			g.menuAnim += speed
		}
	} else {
		if g.menuAnim > 0.0 {
			g.menuAnim -= speed
		}
	}
	if g.menuAnim > 1.0 {
		g.menuAnim = 1.0
	}
	if g.menuAnim < 0.0 {
		g.menuAnim = 0.0
	}

	currentPetW := g.MyPet.Width
	currentPetH := g.MyPet.Height

	targetW := currentPetW + int(float64(MenuWidth)*g.menuAnim)
	targetH := currentPetH

	// 纵向撑高：哪怕是 Mini 模式，也确保有 180px 高度容纳菜单
	if g.menuAnim > 0 && MinMenuH > currentPetH {
		targetH = MinMenuH
	}

	w, h := ebiten.WindowSize()
	if w != targetW || h != targetH {
		ebiten.SetWindowSize(targetW, targetH)
	}

	if g.menuAnim > 0 {
		wx, wy := ebiten.WindowPosition()
		sw, sh := ebiten.ScreenSizeInFullscreen()
		needsMove := false

		if wx+targetW > sw {
			wx = sw - targetW
			needsMove = true
		}
		if wy+targetH > sh {
			wy = sh - targetH
			needsMove = true
		}
		if needsMove {
			ebiten.SetWindowPosition(wx, wy)
			g.lastWinX = wx
			g.lastWinY = wy
		}
	}
}

func (g *Manager) handleMenuClick() {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}

	mx, my := ebiten.CursorPosition()
	if mx < g.MyPet.Width {
		return
	}

	// 紧凑型碰撞检测
	clickedIdx := -1
	for i := 0; i < 6; i++ {
		top := StartY + i*RowHeight
		bot := top + RowHeight
		if my >= top && my <= bot {
			clickedIdx = i
			break
		}
	}

	switch clickedIdx {
	case 0:
		g.ShowColor = !g.ShowColor
		g.saveState()
	case 1:
		g.ShowGlitch = !g.ShowGlitch
		g.saveState()
	case 2:
		g.ShowAnimation = !g.ShowAnimation
		g.saveState()
	case 3:
		g.ShowMonitor = !g.ShowMonitor
		g.saveState()
	case 4:
		g.DisplayMode = (g.DisplayMode + 1) % 3
		g.saveState()
		g.LoadPetImage(g.currentImgPath)
	case 5:
		g.saveState()
		os.Exit(0)
	}
}
