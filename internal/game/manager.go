// Package game provides functions and types for managing game logic.
package game

import (
	"log"
	"os"

	"0xPet/config"
	"0xPet/internal/entity"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

type Manager struct {
	MyPet *entity.Pet
	tick  float64

	ShowColor     bool
	ShowGlitch    bool
	ShowAnimation bool
	ShowMonitor   bool

	// 【新增】显示模式：0=正常, 1=高分辨率, 2=迷你模式
	DisplayMode int

	// 【新增】字体实例
	FontNormal font.Face
	FontSmall  font.Face

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
	ShowMenu bool
	menuAnim float64
}

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

	// 【新增】加载 TTF 字体并生成一大一小两个字库实例
	fontBytes, err := os.ReadFile("assets/PixelOperatorMono.ttf")
	if err != nil {
		log.Fatal("无法加载字体文件:", err)
	}
	tt, err := opentype.Parse(fontBytes)
	if err != nil {
		log.Fatal("解析字体失败:", err)
	}
	// DPI 72 是一倍缩放标准。13 和 6.5 是绝对像素高度
	g.FontNormal, _ = opentype.NewFace(tt, &opentype.FaceOptions{Size: 13, DPI: 72})
	g.FontSmall, _ = opentype.NewFace(tt, &opentype.FaceOptions{Size: 6.5, DPI: 72})

	imageToLoad := "assets/idle.png"
	if cfg.ImagePath != "" {
		if _, err := os.Stat(cfg.ImagePath); err == nil {
			imageToLoad = cfg.ImagePath
		}
	}

	g.LoadPetImage(imageToLoad)
}

func (g *Manager) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func (g *Manager) Update() error {
	if err := g.handleSystemInput(); err != nil {
		return err
	}
	g.handleUIInput()
	g.tick++
	g.updateEffects()

	if g.ShowMenu && g.menuAnim > 0.9 {
		g.handleMenuClick()
		return nil
	}

	g.updatePhysics()
	return nil
}

func (g *Manager) Draw(screen *ebiten.Image) {
	if g.menuAnim > 0 {
		g.drawMenu(screen)
	}
	g.drawPet(screen)
}
