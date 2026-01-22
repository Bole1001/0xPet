package game

import (
	"fmt"
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
	"0xPet/internal/monitor"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"
)

type Manager struct {
	MyPet         *entity.Pet
	tick          float64 // 【新增】用于记录时间/帧数，计算正弦波
	ShowColor     bool    // true = 显示彩色，false = 显示经典绿
	ShowGlitch    bool    // 【新增】是否开启乱码故障
	ShowAnimation bool    // 【新增】是否开启上下浮动呼吸
	ShowMonitor   bool    // 【新增】是否显示 HUD (文字挂件)
	isDragging    bool    // 是否正在拖拽
	dragStartX    int     // 拖拽开始时，鼠标相对于窗口的X
	dragStartY    int     // 拖拽开始时，鼠标相对于窗口的Y
}

func (g *Manager) Init() {
	g.MyPet = &entity.Pet{}
	g.ShowColor = true
	g.ShowGlitch = true
	g.ShowAnimation = true
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
	// 1. 转字符
	charWidthCount := 50
	asciiLines, grid := ascii.Convert(img, charWidthCount)

	// 2. 计算尺寸
	fontW, fontH := 7, 13
	maxLineLen := 0
	for _, line := range asciiLines {
		if len(line) > maxLineLen {
			maxLineLen = len(line)
		}
	}
	winWidth := maxLineLen * fontW
	paddingTop := 20
	winHeight := len(asciiLines)*fontH + paddingTop

	// 3. 更新数据
	fullText := strings.Join(asciiLines, "\n")
	g.MyPet.OriginalContent = fullText
	g.MyPet.Content = fullText
	g.MyPet.Grid = grid
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

	// 【新增】按 'C' 键切换 彩色/纯色 模式
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		g.ShowColor = !g.ShowColor
	}
	// 【新增】按 G 切换乱码
	if inpututil.IsKeyJustPressed(ebiten.KeyG) {
		g.ShowGlitch = !g.ShowGlitch
	}
	// 【新增】按 A 切换浮动
	if inpututil.IsKeyJustPressed(ebiten.KeyA) {
		g.ShowAnimation = !g.ShowAnimation
	}
	// 【新增】TAB 键切换 监控文字显示
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		g.ShowMonitor = !g.ShowMonitor
	}

	g.tick++ // 每一帧加 1

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

	// --- 故障特效逻辑 ---
	// 1. 【必须执行】先重置：把上一帧的乱码全部还原
	// 无论开关是否开启，都要先清理上一帧的现场
	for r := range g.MyPet.Grid {
		for c := range g.MyPet.Grid[r] {
			g.MyPet.Grid[r][c].Char = g.MyPet.Grid[r][c].OriginalChar
		}
	}

	// 2. 【按需执行】再搞破坏
	// 只有当开关开启时，才生成新的乱码
	if g.ShowGlitch {
		if rand.Intn(100) < 10 { // 10% 概率触发
			rows := len(g.MyPet.Grid)
			glitchCount := 5 + rand.Intn(5)

			for i := 0; i < glitchCount; i++ {
				r := rand.Intn(rows)
				c := rand.Intn(len(g.MyPet.Grid[r]))
				chars := []string{"?", "#", "$", "&", "0", "1", "!"}
				g.MyPet.Grid[r][c].Char = chars[rand.Intn(len(chars))]
			}
		}
	}

	// 【新增】数据同步逻辑 (放在 return nil 之前)
	// 1. 从监控模块获取最新数据
	cpu, mem := monitor.GetStats()

	// 2. 存入宠物实体
	g.MyPet.CPUUsage = cpu
	g.MyPet.MemUsage = mem

	// 3. 状态映射 (定义阈值)
	// 如果 CPU 超过 80%，进入“高压状态”
	if cpu > 80.0 {
		g.MyPet.IsStressed = true
	} else {
		g.MyPet.IsStressed = false
	}
	return nil
}

func (g *Manager) Draw(screen *ebiten.Image) {
	// 1. 计算悬浮
	offsetY := 0.0 // 默认为 0 (静止)

	// 【新增】如果开关开启，才计算波浪
	if g.ShowAnimation {
		offsetY = math.Sin(g.tick*0.05) * 5
	}

	baseY := 30 + int(offsetY)

	fontW, fontH := 7, 13

	// 2. 统一遍历网格渲染
	for r, row := range g.MyPet.Grid {
		for c, charData := range row {
			x := c * fontW
			y := r*fontH + baseY

			// 决定颜色
			var drawColor color.Color
			// 优先级 1: 如果 CPU 高压，强制变红！
			if g.MyPet.IsStressed {
				drawColor = color.RGBA{255, 50, 50, 255} // 鲜艳的红色
			} else if g.ShowColor {
				// 优先级 2: 彩色模式
				drawColor = charData.Color
			} else {
				// 优先级 3: 默认纯色 (绿色)
				drawColor = color.RGBA{0, 255, 0, 255}
			}

			// 核心变化：画的是 charData.Char (它可能是正常的，也可能是故障乱码)
			text.Draw(screen, charData.Char, basicfont.Face7x13, x, y, drawColor)
		}
	}

	// 3. 【新增】绘制 HUD 监控文字
	if g.ShowMonitor {
		// 格式化字符串：保留0位小数 (例如 "CPU: 12% | MEM: 40%")
		msg := fmt.Sprintf("CPU: %.0f%% | MEM: %.0f%%", g.MyPet.CPUUsage, g.MyPet.MemUsage)

		// 为了让文字看清楚，我们画在背景上，用黄色高亮
		// 位置设在 (0, 10) 也就是第一行，可能会覆盖一点点头部，但最清晰
		// 如果想要文字不随宠物浮动，Y坐标就不要加 offsetY
		text.Draw(screen, msg, basicfont.Face7x13, 0, 10, color.RGBA{255, 255, 0, 255}) // 黄色
	}
}

func (g *Manager) Layout(outsideWidth, outsideHeight int) (int, int) {
	// 告诉 Ebiten 画布大小就是窗口大小
	return g.MyPet.Width, g.MyPet.Height
}
