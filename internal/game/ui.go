package game

import (
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	MenuWidth  = 200 // 抽屉菜单的固定物理宽度
	MenuHeight = 250 // 抽屉菜单的最低高度 (确保能容纳 5 个按钮)
)

// handleUIInput 处理快捷键、右键状态以及菜单展开的动态窗口伸缩
func (g *Manager) handleUIInput() {
	// 1. 快捷键监听
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

	// 2. 右键菜单开关
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		g.ShowMenu = !g.ShowMenu
		g.velX, g.velY = 0, 0 // 打开/关闭菜单时强行消除物理惯性
	}

	// 3. 失去焦点自动收起
	if g.ShowMenu && !ebiten.IsFocused() {
		g.ShowMenu = false
		g.velX, g.velY = 0, 0
	}

	// 4. 计算动画进度 (平滑插值)
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

	// 5. 动态调整窗口尺寸 (侧滑抽屉核心逻辑)
	currentPetW := g.MyPet.Width
	currentPetH := g.MyPet.Height

	// 横向拉长：宠物宽度 + 菜单宽度*进度
	targetW := currentPetW + int(float64(MenuWidth)*g.menuAnim)
	targetH := currentPetH

	// 纵向撑高：如果菜单处于展开状态，且宠物高度不足以放下菜单，则撑高窗口
	if g.menuAnim > 0 && MenuHeight > currentPetH {
		targetH = MenuHeight
	}

	w, h := ebiten.WindowSize()
	if w != targetW || h != targetH {
		ebiten.SetWindowSize(targetW, targetH)
	}
}

// handleMenuClick 处理菜单界面的点击坐标命中检测 (Hit-Testing)
func (g *Manager) handleMenuClick() {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}

	mx, my := ebiten.CursorPosition()

	// 【新增拦截】X轴阻断
	// 因为菜单是从宠物右侧拉出来的，如果点击的 X 坐标小于宠物宽度，说明点在宠物身上，直接忽略
	if mx < g.MyPet.Width {
		return
	}

	// 定义布局参数 (必须与 Draw 层严格一致)
	startBtnY := 50
	btnH := 30
	gap := 15

	btn1Top := startBtnY
	btn1Bot := btn1Top + btnH

	btn2Top := btn1Bot + gap
	btn2Bot := btn2Top + btnH

	btn3Top := btn2Bot + gap
	btn3Bot := btn3Top + btnH

	btn4Top := btn3Bot + gap
	btn4Bot := btn4Top + btnH

	_, h := ebiten.WindowSize()
	btnExitTop := h - 50
	btnExitBot := btnExitTop + btnH

	// 命中执行逻辑
	if my >= btn1Top && my <= btn1Bot {
		g.ShowColor = !g.ShowColor
		g.saveState()
		return
	}
	if my >= btn2Top && my <= btn2Bot {
		g.ShowGlitch = !g.ShowGlitch
		g.saveState()
		return
	}
	if my >= btn3Top && my <= btn3Bot {
		g.ShowAnimation = !g.ShowAnimation
		g.saveState()
		return
	}
	if my >= btn4Top && my <= btn4Bot {
		g.ShowMonitor = !g.ShowMonitor
		g.saveState()
		return
	}
	if my >= btnExitTop && my <= btnExitBot {
		g.saveState()
		os.Exit(0)
	}
}
