// Package game provides functions and types for managing game logic.
package game

import (
	"log"
	"os"
	"time"

	"0xPet/config"
	"0xPet/internal/entity"
	"0xPet/internal/monitor"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

type Manager struct {
	MyPet *entity.Pet

	ShowColor   bool
	ShowMonitor bool
	ShowMenu    bool
	menuAnim    float64

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

	petCanvas *ebiten.Image
	isDirty   bool
}

func (g *Manager) Init() {
	g.MyPet = &entity.Pet{}

	cfg, err := config.Load("config.json")
	if err != nil {
		log.Println("读取配置失败，使用默认值:", err)
	}

	g.ShowColor = cfg.ShowColor
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
	g.FontNormal, _ = opentype.NewFace(tt, &opentype.FaceOptions{Size: 16, DPI: 72})
	g.FontSmall, _ = opentype.NewFace(tt, &opentype.FaceOptions{Size: 8, DPI: 72})

	imageToLoad := "assets/idle.png"
	if cfg.ImagePath != "" {
		if _, err := os.Stat(cfg.ImagePath); err == nil {
			imageToLoad = cfg.ImagePath
		}
	}

	g.LoadPetImage(imageToLoad)

	// 【新增：异步硬件监控协程】
	// 与主渲染线程完全物理隔离，每 2 秒更新一次数据即可，彻底释放系统 CPU
	go func() {
		for {
			// 如果由于某种原因 MyPet 未初始化，进行防御性挂起
			if g.MyPet == nil {
				time.Sleep(1 * time.Second)
				continue
			}

			// 低频调用系统 API
			cpu, mem := monitor.GetStats()

			// 写入内存，供渲染层 (drawPet) 直接以 O(1) 复杂度读取
			g.MyPet.CPUUsage = cpu
			g.MyPet.MemUsage = mem
			g.MyPet.IsStressed = cpu > 80.0

			// 强制休眠 2 秒 (人类查看 HUD 数据的合理刷新率)
			time.Sleep(2 * time.Second)
		}
	}()
}

func (g *Manager) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func (g *Manager) Update() error {
	if err := g.handleSystemInput(); err != nil {
		return err
	}
	g.handleUIInput()
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
