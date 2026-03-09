// Package game provides functions and types for managing game logic.
package game

import (
	"log"
	"os"

	"0xPet/config"
	"0xPet/internal/entity"

	"github.com/hajimehoshi/ebiten/v2"
)

// Manager 游戏全局状态管理器
type Manager struct {
	MyPet *entity.Pet
	tick  float64

	ShowColor     bool
	ShowGlitch    bool
	ShowAnimation bool
	ShowMonitor   bool

	// 物理相关
	isDragging bool
	dragStartX int
	dragStartY int
	velX       float64
	velY       float64
	lastWinX   int
	lastWinY   int

	currentImgPath string

	// 菜单相关
	ShowMenu   bool
	menuAnim   float64
	menuHeight int
}

// Init 初始化管理器
func (g *Manager) Init() {
	g.MyPet = &entity.Pet{}

	cfg, err := config.Load("config.json")
	if err != nil {
		log.Println("读取配置失败，使用默认值:", err)
	}

	g.ShowColor = cfg.ShowColor
	g.ShowGlitch = cfg.ShowGlitch
	g.ShowAnimation = cfg.ShowAnimation
	g.ShowMonitor = cfg.ShowMonitor

	imageToLoad := "assets/idle.png"
	if cfg.ImagePath != "" {
		if _, err := os.Stat(cfg.ImagePath); err == nil {
			imageToLoad = cfg.ImagePath
		}
	}

	g.LoadPetImage(imageToLoad)
	g.menuHeight = 200 // 待重构移除
}

// Layout 定义逻辑屏幕尺寸
func (g *Manager) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

// Update 游戏主循环 (逻辑层)
func (g *Manager) Update() error {
	// 1. 系统级输入 (ESC退出、文件拖拽)
	if err := g.handleSystemInput(); err != nil {
		return err
	}

	// 2. UI交互输入 (快捷键、右键菜单)
	g.handleUIInput()

	// 3. 时间步进
	g.tick++

	// 先更新特效状态，确保即使在菜单打开时，背后的宠物依然有特效
	g.updateEffects()

	// 4. 拦截器：如果菜单已完全打开，则挂起物理引擎
	if g.ShowMenu && g.menuAnim > 0.9 {
		g.handleMenuClick()
		return nil
	}

	// 5. 核心逻辑更新 (物理滑行被挂起，不影响静止时的特效)
	g.updatePhysics()

	return nil
}

// Draw 游戏渲染循环 (表现层)
func (g *Manager) Draw(screen *ebiten.Image) {
	if g.menuAnim > 0 {
		g.drawMenu(screen)
	}
	g.drawPet(screen)
}
