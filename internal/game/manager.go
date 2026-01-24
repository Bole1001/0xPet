package game

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"io"
	"io/fs" // 【新增】处理文件系统接口
	"log"
	"math"      // 为了后面的动画
	"math/rand" // 为了后面的故障特效
	"os"
	"strings"

	_ "image/png" // 必加，否则 image: unknown format

	"0xPet/config"
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
	// 【新增】物理引擎相关
	velX           float64 // X轴速度
	velY           float64 // Y轴速度
	lastWinX       int     // 上一帧窗口的 X 坐标 (用于计算甩出去的速度)
	lastWinY       int     // 上一帧窗口的 Y 坐标
	currentImgPath string  // 【新增】记录当前图片的绝对路径
}

func (g *Manager) Init() {
	g.MyPet = &entity.Pet{}
	// 1. 读取配置
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Println("读取配置失败，使用默认值:", err)
	}

	// 2. 应用开关状态
	g.ShowColor = cfg.ShowColor
	g.ShowGlitch = cfg.ShowGlitch
	g.ShowAnimation = cfg.ShowAnimation
	g.ShowMonitor = cfg.ShowMonitor

	// 3. 智能加载图片
	// 如果配置文件里的图片路径存在，就用它；否则用默认图
	imageToLoad := "assets/idle.png"
	if cfg.ImagePath != "" {
		// 检查文件是否存在
		if _, err := os.Stat(cfg.ImagePath); err == nil {
			imageToLoad = cfg.ImagePath
		}
	}

	// 加载图片
	g.LoadPetImage(imageToLoad)
}

// 用于 Init 加载本地文件
func (g *Manager) LoadPetImage(path string) {
	file, err := os.Open(path)
	if err != nil {
		log.Println("本地图片加载失败:", err)
		return
	}
	defer file.Close()

	// 【新增】记录路径，以便下次保存
	g.currentImgPath = path

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

				// 【核心修改】
				// 1. 先把文件内容全部读出来，存到内存里
				// 因为流只能读一次，我们要用它做两件事(显示+保存)，所以必须先读出来
				fileBytes, err := io.ReadAll(f)
				if err != nil {
					log.Println("读取文件失败:", err)
				} else {
					// 2. 用读出来的字节进行解码，更新显示
					img, _, err := image.Decode(bytes.NewReader(fileBytes))
					if err == nil {
						log.Println("拖拽加载成功:", fileName)
						g.UpdatePetWithImage(img)

						// 3. 【新增】把这份数据写入本地硬盘，作为“存档”
						saveName := "assets/saved_pet.png"
						err = os.WriteFile(saveName, fileBytes, 0644)
						if err != nil {
							log.Println("图片缓存失败:", err)
						} else {
							// 4. 更新内存中的路径，并立即保存配置
							g.currentImgPath = saveName
							g.saveState() // 立即更新 config.json
							log.Println("图片已缓存并保存配置")
						}
					}
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
	// 只要鼠标抓着，或者速度还没降下来，就算是在动
	isMoving := g.isDragging || math.Abs(g.velX) > 0.1 || math.Abs(g.velY) > 0.1

	// 2. 动态调整 TPS
	// 如果正在交互（拖拽或鼠标指着它），开启 60 帧丝滑模式
	if isHover || isMoving || g.ShowAnimation {
		ebiten.SetTPS(60)
	} else {
		// 没人理它时，开启省电模式，每秒只动 5 下
		ebiten.SetTPS(5)
	}

	// 1. 实现 ESC 关闭程序
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		// 【新增】退出前保存状态
		g.saveState()
		return ebiten.Termination
	}

	// 2. 拖拽逻辑
	// 获取当前状态
	mx, my := ebiten.CursorPosition()
	isClicking := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)

	// 获取当前窗口位置 (绝对坐标)
	wx, wy := ebiten.WindowPosition()

	if isClicking {
		// === 状态 A: 正在被鼠标抓着 ===
		// 你的逻辑: 停止物理滑行，完全跟随鼠标

		if !g.isDragging {
			// 刚按下的瞬间：记录锚点
			g.isDragging = true
			g.dragStartX = mx
			g.dragStartY = my
		} else {
			// 拖拽中：移动窗口
			newX := wx + mx - g.dragStartX
			newY := wy + my - g.dragStartY
			ebiten.SetWindowPosition(newX, newY)

			// 【关键】计算即时速度 (Throwing Velocity)
			// 速度 = 当前位置 - 上一帧位置
			// 这样当你松手时，它就保留了最后这一瞬间的速度
			g.velX = float64(newX - g.lastWinX)
			g.velY = float64(newY - g.lastWinY)
		}
	} else {
		// === 状态 B: 松手了 (自由滑行) ===
		g.isDragging = false

		// 只有当速度足够大时才计算物理 (避免微小抖动)
		if math.Abs(g.velX) > 0.1 || math.Abs(g.velY) > 0.1 {

			// 1. 应用惯性 (移动窗口)
			wx += int(g.velX)
			wy += int(g.velY)
			ebiten.SetWindowPosition(wx, wy)

			// 2. 应用摩擦力 (慢慢停下)
			// 0.95 是摩擦系数，越小停得越快
			friction := 0.95
			g.velX *= friction
			g.velY *= friction

			// 3. 屏幕边缘碰撞检测 (反弹)
			sw, sh := ebiten.ScreenSizeInFullscreen()

			// 左墙
			if wx < 0 {
				wx = 0
				g.velX = -g.velX * 0.6 // 反弹并损耗 40% 能量
			}
			// 右墙
			if wx+g.MyPet.Width > sw {
				wx = sw - g.MyPet.Width
				g.velX = -g.velX * 0.6
			}
			// 上墙
			if wy < 0 {
				wy = 0
				g.velY = -g.velY * 0.6
			}
			// 下墙
			if wy+g.MyPet.Height > sh {
				wy = sh - g.MyPet.Height
				g.velY = -g.velY * 0.6
			}

			// 如果修正了位置，需要应用回去
			ebiten.SetWindowPosition(wx, wy)
		} else {
			// 速度太小，直接归零，省电
			g.velX = 0
			g.velY = 0
		}
	}

	// 【关键】记录这一帧的位置，留给下一帧算速度用
	// 注意：这里要重新获取一下最终位置，因为上面可能发生了碰撞修正
	finalX, finalY := ebiten.WindowPosition()
	g.lastWinX = finalX
	g.lastWinY = finalY

	// --- 故障特效逻辑 ---
	// 1. 【必须执行】先重置：把上一帧的乱码全部还原
	// 无论开关是否开启，都要先清理上一帧的现场
	for r := range g.MyPet.Grid {
		for c := range g.MyPet.Grid[r] {
			g.MyPet.Grid[r][c].Char = g.MyPet.Grid[r][c].OriginalChar
		}
	}

	// 2. 【按需执行】再搞破坏
	// 只有当 (开关开启) 且 (没在动) 时，才产生乱码
	if g.ShowGlitch && !isMoving {
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
	// 【新增】判断运动状态 (因为 Draw 不存状态，这里简单重算一下即可)
	isMoving := g.isDragging || math.Abs(g.velX) > 0.1 || math.Abs(g.velY) > 0.1

	// 1. 计算悬浮
	offsetY := 0.0 // 默认为 0 (静止)

	// 只有 (开关开启) 且 (没在动) 时，才计算波浪
	if g.ShowAnimation && !isMoving {
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
	if g.ShowMonitor && !isMoving {
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

// saveState 将当前状态写入 config.json
func (g *Manager) saveState() {
	cfg := &config.Config{
		ImagePath:     g.currentImgPath, // 保存当前图片路径
		ShowColor:     g.ShowColor,
		ShowGlitch:    g.ShowGlitch,
		ShowAnimation: g.ShowAnimation,
		ShowMonitor:   g.ShowMonitor,
	}

	// 调用我们在上一步写好的 Save 函数
	if err := config.Save(cfg, "config.json"); err != nil {
		// 如果保存失败（比如没权限），打印日志但不崩溃
		log.Println("保存配置失败:", err)
	} else {
		log.Println("配置已保存")
	}
}
