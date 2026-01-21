package game

import (
	"image"
	"image/color"
	"log"
	"os"
	"strings"

	_ "image/png" // 必加，否则 image: unknown format

	"0xPet/internal/ascii"
	"0xPet/internal/entity"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"
)

type Manager struct {
	MyPet      *entity.Pet
	isDragging bool // 是否正在拖拽
	dragStartX int  // 拖拽开始时，鼠标相对于窗口的X
	dragStartY int  // 拖拽开始时，鼠标相对于窗口的Y
}

func (g *Manager) Init() {
	g.MyPet = &entity.Pet{}

	// 1. 读取图片
	file, err := os.Open("assets/idle.png")
	if err != nil {
		log.Fatal("找不到 assets/idle.png")
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		log.Fatal("解码失败:", err)
	}

	// 2. 转字符 (宽度设为 50，你可以调大调小)
	charWidthCount := 50
	asciiLines := ascii.Convert(img, charWidthCount)

	// 3. 【关键】计算尺寸 (量体裁衣)
	// basicfont.Face7x13 的特性：每个字宽 7 像素，高 13 像素
	fontW, fontH := 7, 13

	// 找最长的一行
	maxLineLen := 0
	for _, line := range asciiLines {
		if len(line) > maxLineLen {
			maxLineLen = len(line)
		}
	}

	// 算出像素宽高
	winWidth := maxLineLen * fontW
	winHeight := len(asciiLines) * fontH

	// 4. 保存数据
	g.MyPet.Content = strings.Join(asciiLines, "\n")
	g.MyPet.Width = winWidth
	g.MyPet.Height = winHeight

	// 5. 设置窗口
	// 让窗口大小 = 字符画大小
	ebiten.SetWindowSize(winWidth, winHeight)
	// 把窗口挪到屏幕中间 (或者随便一个位置，比如 200, 200)
	ebiten.SetWindowPosition(200, 200)
}

func (g *Manager) Update() error {
	// 1. 获取鼠标状态
	x, y := ebiten.CursorPosition()
	isHover := x >= 0 && x <= g.MyPet.Width && y >= 0 && y <= g.MyPet.Height
	isDragging := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)

	// 2. 动态调整 TPS
	// 如果正在交互（拖拽或鼠标指着它），开启 60 帧丝滑模式
	if isHover || isDragging {
		ebiten.SetTPS(60)
	} else {
		// 没人理它时，开启省电模式，每秒只动 5 下
		ebiten.SetTPS(5)
	}

	// 1. 实现 ESC 关闭程序
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}

	// 2. 拖拽逻辑
	// 获取鼠标相对于窗口左上角的坐标
	mx, my := ebiten.CursorPosition()

	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		if !g.isDragging {
			// 刚按下的瞬间，记录鼠标相对于窗口的偏移量
			g.isDragging = true
			g.dragStartX = mx
			g.dragStartY = my
		} else {
			// 正在拖拽中：计算新的窗口位置
			// 新窗口位置 = 鼠标屏幕绝对位置 - 初始偏移量
			// 但 Ebiten 只给相对坐标，所以需要结合 WindowPosition
			wx, wy := ebiten.WindowPosition()
			// 这里的数学逻辑：
			// 当前鼠标在屏幕的绝对位置 = wx + mx
			// 我们希望保持 (wx_new + dragStartX) = (wx + mx)
			// 所以 wx_new = wx + mx - dragStartX

			ebiten.SetWindowPosition(wx+mx-g.dragStartX, wy+my-g.dragStartY)
		}
	} else {
		g.isDragging = false
	}

	return nil
}

func (g *Manager) Draw(screen *ebiten.Image) {
	// 画在 (0, 11) 的位置
	// 为什么要 Y=11？因为文字是从脚底（基线）开始画的，往下挪一点防止头被切掉
	text.Draw(screen, g.MyPet.Content, basicfont.Face7x13, 0, 11, color.RGBA{0, 255, 0, 255})
}

func (g *Manager) Layout(outsideWidth, outsideHeight int) (int, int) {
	// 告诉 Ebiten 画布大小就是窗口大小
	return g.MyPet.Width, g.MyPet.Height
}
