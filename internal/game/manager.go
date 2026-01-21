package game

import (
	"image"
	"image/color"
	"io/fs" // 【新增】处理文件系统接口
	"log"
	"math"      // 为了后面的动画
	"math/rand" // 为了后面的故障特效
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
	tick       float64 // 【新增】用于记录时间/帧数，计算正弦波
	isDragging bool    // 是否正在拖拽
	dragStartX int     // 拖拽开始时，鼠标相对于窗口的X
	dragStartY int     // 拖拽开始时，鼠标相对于窗口的Y
}

func (g *Manager) Init() {
	g.MyPet = &entity.Pet{}
	// 加载默认图片
	g.LoadPetImage("assets/idle.png")
}

// 用于 Init 加载本地文件
func (g *Manager) LoadPetImage(path string) {
	file, err := os.Open(path)
	if err != nil {
		log.Println("本地图片加载失败:", err)
		return
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		log.Println("解码失败:", err)
		return
	}

	// 调用上面那个通用函数
	g.UpdatePetWithImage(img)
}

// 【新增】核心逻辑：只负责把图片对象转成字符画，不关心图片从哪来的
func (g *Manager) UpdatePetWithImage(img image.Image) {
	// --- 从这里开始粘贴你原来 LoadPetImage 里解码之后的代码 ---

	// 1. 转字符
	charWidthCount := 50
	asciiLines := ascii.Convert(img, charWidthCount)

	// 2. 计算尺寸
	fontW, fontH := 7, 13
	maxLineLen := 0
	for _, line := range asciiLines {
		if len(line) > maxLineLen {
			maxLineLen = len(line)
		}
	}
	winWidth := maxLineLen * fontW
	winHeight := len(asciiLines) * fontH

	// 3. 更新数据
	fullText := strings.Join(asciiLines, "\n")
	g.MyPet.OriginalContent = fullText
	g.MyPet.Content = fullText
	g.MyPet.Width = winWidth
	g.MyPet.Height = winHeight

	// 4. 重设窗口
	ebiten.SetWindowSize(winWidth, winHeight)
}

func (g *Manager) Update() error {
	// 【新增】文件拖拽监听
	// dr 是一个文件系统接口 (fs.FS)
	if dr := ebiten.DroppedFiles(); dr != nil {
		// 读取这个虚拟文件系统根目录下的所有文件
		entries, err := fs.ReadDir(dr, ".")
		if err == nil && len(entries) > 0 {
			// 获取第一个文件的文件名
			fileName := entries[0].Name()

			// 打开文件 (得到一个类似流的对象)
			f, err := dr.Open(fileName)
			if err == nil {
				defer f.Close()

				// 解码图片 (image.Decode 支持直接读流)
				img, _, err := image.Decode(f)
				if err == nil {
					// 成功！调用通用函数更新宠物
					log.Println("拖拽加载成功:", fileName)
					g.UpdatePetWithImage(img)
				} else {
					log.Println("拖拽图片解码失败:", err)
				}
			}
		}
	}

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

	g.tick++ // 每一帧加 1

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

	// 【新增】故障特效逻辑
	// 1% 的概率触发故障 (你可以调整这个概率，比如 50 = 2% 左右)
	if rand.Intn(100) < 5 {
		// 将字符串转为 rune 切片方便修改
		runes := []rune(g.MyPet.OriginalContent)

		// 随机挑 5 个字符改成乱码
		for i := 0; i < 5; i++ {
			idx := rand.Intn(len(runes))
			// 只有当这个位置不是换行符时才改
			if runes[idx] != '\n' {
				// 变成随机字符，比如 '?' 或 '#' 或 乱码
				chars := []rune("@#?&01")
				runes[idx] = chars[rand.Intn(len(chars))]
			}
		}
		g.MyPet.Content = string(runes)
	} else {
		// 95% 的时间恢复正常 (从 OriginalContent 还原)
		// 这样故障只会闪一下
		g.MyPet.Content = g.MyPet.OriginalContent
	}
	return nil
}

func (g *Manager) Draw(screen *ebiten.Image) {
	// 【新增】计算悬浮偏移量
	// math.Sin 返回 -1 到 1
	// g.tick * 0.05 控制速度 (数值越小越慢)
	// * 5 控制幅度 (上下浮动 5 像素)
	offsetY := math.Sin(g.tick*0.05) * 5

	// 原来的 Y=11，现在加上动态偏移
	drawY := 11 + int(offsetY)

	text.Draw(screen, g.MyPet.Content, basicfont.Face7x13, 0, drawY, color.RGBA{0, 255, 0, 255})
}

func (g *Manager) Layout(outsideWidth, outsideHeight int) (int, int) {
	// 告诉 Ebiten 画布大小就是窗口大小
	return g.MyPet.Width, g.MyPet.Height
}
