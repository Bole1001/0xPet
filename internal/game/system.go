package game

import (
	"bytes"
	"image"
	_ "image/png"
	"io"
	"io/fs"
	"log"
	"os"
	"strings"

	"0xPet/config"
	"0xPet/internal/ascii"

	"github.com/hajimehoshi/ebiten/v2"
)

// handleSystemInput 处理系统级输入 (拖拽文件、ESC退出)
func (g *Manager) handleSystemInput() error {
	// 1. ESC 退出程序并保存
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		g.saveState()
		return ebiten.Termination
	}

	// 2. 拖拽文件解析
	if dr := ebiten.DroppedFiles(); dr != nil {
		entries, err := fs.ReadDir(dr, ".")
		if err == nil && len(entries) > 0 {
			fileName := entries[0].Name()
			f, err := dr.Open(fileName)
			if err == nil {
				defer f.Close()
				fileBytes, err := io.ReadAll(f)
				if err != nil {
					log.Println("读取文件失败:", err)
				} else {
					img, _, err := image.Decode(bytes.NewReader(fileBytes))
					if err == nil {
						log.Println("拖拽加载成功:", fileName)
						g.UpdatePetWithImage(img)

						saveName := "assets/saved_pet.png"
						err = os.WriteFile(saveName, fileBytes, 0644)
						if err != nil {
							log.Println("图片缓存失败:", err)
						} else {
							g.currentImgPath = saveName
							g.saveState()
							log.Println("图片已缓存并保存配置")
						}
					}
				}
			}
		}
	}
	return nil
}

// LoadPetImage 读取本地图片文件并触发转换
func (g *Manager) LoadPetImage(path string) {
	file, err := os.Open(path)
	if err != nil {
		log.Println("本地图片加载失败:", err)
		return
	}
	defer file.Close()

	g.currentImgPath = path

	img, _, err := image.Decode(file)
	if err != nil {
		log.Println("解码失败:", err)
		return
	}

	g.UpdatePetWithImage(img)
}

// UpdatePetWithImage 核心逻辑：图片对象转字符画，计算实体尺寸
func (g *Manager) UpdatePetWithImage(img image.Image) {
	var charWidthCount int
	var fontW, fontH float64

	// 【核心逻辑】根据当前模式设定渲染参数
	switch g.DisplayMode {
	case 0: // 正常模式 (50宽，大字)
		charWidthCount = 50
		fontW, fontH = 7.0, 13.0
	case 1: // 高清模式 (100宽，小字，窗口大小不变)
		charWidthCount = 100
		fontW, fontH = 3.5, 6.5
	case 2: // 迷你模式 (50宽，小字，窗口缩小一半)
		charWidthCount = 50
		fontW, fontH = 3.5, 6.5
	}

	asciiLines, grid := ascii.Convert(img, charWidthCount)

	maxLineLen := 0
	for _, line := range asciiLines {
		if len(line) > maxLineLen {
			maxLineLen = len(line)
		}
	}

	// 计算物理窗口大小
	winWidth := int(float64(maxLineLen) * fontW)
	paddingTop := 20
	winHeight := int(float64(len(asciiLines))*fontH) + paddingTop

	fullText := strings.Join(asciiLines, "\n")
	g.MyPet.OriginalContent = fullText
	g.MyPet.Content = fullText
	g.MyPet.Grid = grid
	g.MyPet.Width = winWidth
	g.MyPet.Height = winHeight

	ebiten.SetWindowSize(winWidth, winHeight)
}

// saveState 将当前状态写入 config.json
func (g *Manager) saveState() {
	cfg := &config.Config{
		ImagePath:     g.currentImgPath,
		ShowColor:     g.ShowColor,
		ShowGlitch:    g.ShowGlitch,
		ShowAnimation: g.ShowAnimation,
		ShowMonitor:   g.ShowMonitor,
	}

	if err := config.Save(cfg, "config.json"); err != nil {
		log.Println("保存配置失败:", err)
	} else {
		log.Println("配置已保存")
	}
}
