package game

import (
	"bytes"
	"image"
	"image/draw"
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
	croppedImg := autoCropImage(img)

	var charWidthCount int
	var fontW, fontH float64

	// 【核心逻辑】根据当前模式设定渲染参数
	switch g.DisplayMode {
	case 0: // 正常模式
		charWidthCount = 50
		fontW, fontH = 8.0, 16.0
	case 1: // 高清模式
		charWidthCount = 100
		fontW, fontH = 4.0, 8.0
	case 2: // 迷你模式
		charWidthCount = 50
		fontW, fontH = 4.0, 8.0
	}

	asciiLines, grid := ascii.Convert(croppedImg, charWidthCount)

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

	g.isDirty = true
}

// saveState 将当前状态写入 config.json
func (g *Manager) saveState() {
	cfg := &config.Config{
		ImagePath:   g.currentImgPath,
		ShowColor:   g.ShowColor,
		ShowMonitor: g.ShowMonitor,
	}

	if err := config.Save(cfg, "config.json"); err != nil {
		log.Println("保存配置失败:", err)
	} else {
		log.Println("配置已保存")
	}
}

// autoCropImage 智能预处理：切除图片四周所有的透明像素，提取绝对主体的最小包围盒
func autoCropImage(img image.Image) image.Image {
	bounds := img.Bounds()
	minX, minY := bounds.Max.X, bounds.Max.Y
	maxX, maxY := bounds.Min.X, bounds.Min.Y

	// 1. 扫描寻找包含非透明像素的极值坐标
	hasContent := false
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a > 0 { // 只要不是绝对透明
				hasContent = true
				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}

	// 如果全图都是透明的，或者计算出的包围盒无效，直接返回原图，防止崩溃
	if !hasContent || minX > maxX || minY > maxY {
		return img
	}

	// 2. 增加一点安全边距 (Padding)，防止边缘字符紧贴窗口被系统裁剪
	padding := 2
	minX -= padding
	minY -= padding
	maxX += padding
	maxY += padding

	// 确保不越界
	if minX < bounds.Min.X {
		minX = bounds.Min.X
	}
	if minY < bounds.Min.Y {
		minY = bounds.Min.Y
	}
	if maxX > bounds.Max.X {
		maxX = bounds.Max.X
	}
	if maxY > bounds.Max.Y {
		maxY = bounds.Max.Y
	}

	// 3. 截取核心图像
	cropRect := image.Rect(minX, minY, maxX, maxY)
	croppedImg := image.NewRGBA(image.Rect(0, 0, cropRect.Dx(), cropRect.Dy()))
	draw.Draw(croppedImg, croppedImg.Bounds(), img, cropRect.Min, draw.Src)

	return croppedImg
}
