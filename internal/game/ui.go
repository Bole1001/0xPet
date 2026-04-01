package game

import (
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	MenuWidth = 160 // 宽度收紧，足够放下13px的纯文本
	RowHeight = 25  // 每行的高度，无额外 Gap
	StartY    = 20  // 顶部留白
	MinMenuH  = 180 // 菜单的最小安全高度 (20 + 6*25 + 10底部留白)
)

func (g *Manager) handleUIInput() {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		g.ShowMenu = !g.ShowMenu
		g.velX, g.velY = 0, 0
		g.menuDirty = true
	}
}

func (g *Manager) handleMenuClick() {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}

	mx, my := ebiten.CursorPosition()
	menuW := float64(MenuWidth) * g.menuAnim
	sw, _ := ebiten.WindowSize()
	menuX := float64(sw) - menuW
	if float64(mx) < menuX {
		return
	}

	clickedIdx := -1
	for i := 0; i < 4; i++ {
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
		g.isDirty = true
		g.menuDirty = true
		g.saveState()
	case 1:
		g.ShowMonitor = !g.ShowMonitor
		g.menuDirty = true
		g.saveState()
	case 2:
		g.DisplayMode = (g.DisplayMode + 1) % 3
		g.menuDirty = true
		g.saveState()
		g.LoadPetImage(g.currentImgPath)
	case 3:
		g.saveState()
		os.Exit(0)
	}
}
