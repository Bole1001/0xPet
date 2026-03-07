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
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"
)

type Manager struct {
	MyPet *entity.Pet
	tick  float64 // 【新增】用于记录时间/帧数，计算正弦波

	ShowColor     bool // true = 显示彩色，false = 显示经典绿
	ShowGlitch    bool // 【新增】是否开启乱码故障
	ShowAnimation bool // 【新增】是否开启上下浮动呼吸
	ShowMonitor   bool // 【新增】是否显示 HUD (文字挂件)

	isDragging bool // 是否正在拖拽
	dragStartX int  // 拖拽开始时，鼠标相对于窗口的X
	dragStartY int  // 拖拽开始时，鼠标相对于窗口的Y
	// 【新增】物理引擎相关
	velX     float64 // X轴速度
	velY     float64 // Y轴速度
	lastWinX int     // 上一帧窗口的 X 坐标 (用于计算甩出去的速度)
	lastWinY int     // 上一帧窗口的 Y 坐标

	currentImgPath string // 【新增】记录当前图片的绝对路径

	// 【新增】菜单动画相关系统
	ShowMenu   bool    // 目标状态：true=显示菜单，false=显示宠物
	menuAnim   float64 // 动画进度：0.0(纯宠物) -> 1.0(纯菜单)
	menuHeight int     // 菜单界面的高度 (固定值，比如 200)
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

	g.menuHeight = 200 // 【新增】设定菜单高度
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

	// 新增】检测右键点击 -> 切换菜单开关
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		g.ShowMenu = !g.ShowMenu
		g.velX = 0 // 打开菜单时，强制停止物理滑行
		g.velY = 0
	}

	// 2. 【新增】失去焦点自动归位
	if g.ShowMenu && !ebiten.IsFocused() {
		g.ShowMenu = false
		// 同样清空速度，防止状态切换时的物理漂移
		g.velX = 0
		g.velY = 0
	}

	// === 【新增】菜单点击逻辑 ===
	if g.ShowMenu && g.menuAnim > 0.9 {
		// 检测左键点击
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			_, my := ebiten.CursorPosition()
			// 调用刚才写好的处理函数
			g.handleMenuClick(my)
		}

		// ！！重要！！
		// 如果菜单开着，直接 return，不执行后面的物理引擎和拖拽
		// 这样防止你点按钮的时候，宠物在后面乱跑
		return nil
	}

	//【新增】计算动画进度 (平滑过渡)
	// 每一帧移动 0.1 (也就是 10 帧完成切换，约 0.16秒，非常快且跟手)
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
	// 修正数值，防止溢出 (保持在 0.0 ~ 1.0 之间)
	if g.menuAnim > 1.0 {
		g.menuAnim = 1.0
	}
	if g.menuAnim < 0.0 {
		g.menuAnim = 0.0
	}

	//【新增】动态调整窗口大小
	// 如果菜单出来了（进度 > 0），窗口高度就要变大，否则菜单显示不全
	// 我们取 "宠物当前高度" 和 "菜单高度" 里的最大值
	currentPetH := g.MyPet.Height
	targetH := currentPetH
	if g.menuAnim > 0 {
		if g.menuHeight > currentPetH {
			targetH = g.menuHeight
		}
	}
	// 只有当计算出的高度和当前不一样时，才去设置窗口，避免闪烁
	w, h := ebiten.WindowSize()
	if h != targetH {
		ebiten.SetWindowSize(w, targetH)
	}

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

	if isClicking && g.menuAnim < 0.1 {
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
	// 1. 计算这一帧的位移量
	// 屏幕宽度 * 进度。进度越大，向左移得越远
	w, h := screen.Size()
	slideOffset := float64(w) * g.menuAnim

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
			// 【关键修改】在计算 x 时，减去 slideOffset
			x := float64(c*fontW) - slideOffset
			y := float64(r*fontH + baseY)

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
			text.Draw(screen, charData.Char, basicfont.Face7x13, int(x), int(y), drawColor)
		}
	}

	// 只有当动画开始后才绘制菜单
	if g.menuAnim > 0 {
		// 1. 计算菜单层的位置
		menuX := float64(w) * (1.0 - g.menuAnim)

		// 2. 画背景 (更深邃的黑色，增加对比度)
		// 使用 RGBA{5, 5, 10, 245} 接近纯黑，防透视干扰
		vector.DrawFilledRect(screen, float32(menuX), 0, float32(w), float32(h), color.RGBA{5, 5, 10, 245}, false)

		// --- 装饰元素 ---
		// 在顶部画一条青色的“能量条”，增加科技感
		accentColor := color.RGBA{0, 255, 255, 255} // 青色
		vector.DrawFilledRect(screen, float32(menuX), 0, float32(w), 4, accentColor, false)

		// --- 3. 绘制按钮布局 ---
		// 我们定义一个简单的布局系统
		btnW := float32(w - 40) // 按钮宽度 (左右各留20px)
		btnH := float32(30)     // 按钮高度
		baseBtnX := float32(menuX) + 20
		startBtnY := float32(50)
		gap := float32(15) // 按钮之间的间距

		// 定义颜色
		textCol := color.RGBA{200, 200, 200, 255} // 浅灰文字
		//highlightCol := color.RGBA{0, 255, 0, 255} // 亮绿文字 (用于开启状态)
		btnBg := color.RGBA{30, 30, 40, 255}   // 按钮深灰底色
		exitBg := color.RGBA{180, 40, 40, 255} // 退出按钮红底

		// --- 辅助函数：画居中文字 (因为没有 text.Measure，我们手动算) ---
		drawCenteredText := func(txt string, y int, col color.Color) {
			// basicfont 每个字符宽 7px
			textW := len(txt) * 7
			textX := int(baseBtnX) + (int(btnW)-textW)/2
			text.Draw(screen, txt, basicfont.Face7x13, textX, y+20, col)
		}

		// === 按钮 1: Color Mode ===
		// 画底框
		vector.DrawFilledRect(screen, baseBtnX, startBtnY, btnW, btnH, btnBg, false)
		// 画左边的装饰竖条 (状态指示器)
		statusCol := textCol
		statusText := "COLOR: OFF"
		if g.ShowColor {
			statusCol = accentColor // 开启时变亮
			statusText = "COLOR: ON"
			// 开启时左边亮条
			vector.DrawFilledRect(screen, baseBtnX, startBtnY, 4, btnH, accentColor, false)
		}
		drawCenteredText(statusText, int(startBtnY), statusCol)

		// === 按钮 2: Glitch Effect ===
		y2 := startBtnY + btnH + gap
		vector.DrawFilledRect(screen, baseBtnX, y2, btnW, btnH, btnBg, false)
		statusText = "GLITCH: OFF"
		statusCol = textCol
		if g.ShowGlitch {
			statusCol = accentColor
			statusText = "GLITCH: ON"
			vector.DrawFilledRect(screen, baseBtnX, y2, 4, btnH, accentColor, false)
		}
		drawCenteredText(statusText, int(y2), statusCol)

		// === 【新增】按钮 3: Float Animation (浮动) ===
		y3 := y2 + btnH + gap // 算在 Glitch 下面
		vector.DrawFilledRect(screen, baseBtnX, y3, btnW, btnH, btnBg, false)
		statusText = "FLOAT: OFF"
		statusCol = textCol
		if g.ShowAnimation {
			statusCol = accentColor
			statusText = "FLOAT: ON"
			// 亮灯
			vector.DrawFilledRect(screen, baseBtnX, y3, 4, btnH, accentColor, false)
		}
		drawCenteredText(statusText, int(y3), statusCol)

		// === 【修改】按钮 4: Monitor HUD (被挤下来了) ===
		y4 := y3 + btnH + gap // 注意：这里变成了 y4，基于 y3 计算
		vector.DrawFilledRect(screen, baseBtnX, y4, btnW, btnH, btnBg, false)
		statusText = "HUD: OFF"
		statusCol = textCol
		if g.ShowMonitor {
			statusCol = accentColor
			statusText = "HUD: ON"
			vector.DrawFilledRect(screen, baseBtnX, y4, 4, btnH, accentColor, false)
		}
		drawCenteredText(statusText, int(y4), statusCol)

		// === 按钮 4: EXIT (放在最底下) ===
		// 把它放远一点，防止误触
		yExit := float32(h) - 50
		vector.DrawFilledRect(screen, baseBtnX, yExit, btnW, btnH, exitBg, false)
		drawCenteredText(">> SYSTEM EXIT <<", int(yExit), color.White)
	}

	// 3. 【新增】绘制 HUD 监控文字
	if g.ShowMonitor && !isMoving {
		// 格式化字符串：保留0位小数 (例如 "CPU: 12% | MEM: 40%")
		msg := fmt.Sprintf("CPU: %.0f%% | MEM: %.0f%%", g.MyPet.CPUUsage, g.MyPet.MemUsage)

		// 为了让文字看清楚，我们画在背景上，用黄色高亮
		// 位置设在 (0, 10) 也就是第一行，可能会覆盖一点点头部，但最清晰
		// 如果想要文字不随宠物浮动，Y坐标就不要加 offsetY
		text.Draw(screen, msg, basicfont.Face7x13, 0-int(slideOffset), 10, color.RGBA{255, 255, 0, 255}) // 黄色
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

// handleMenuClick 处理菜单界面的点击事件
func (g *Manager) handleMenuClick(my int) {
	// --- 1. 定义布局参数 (必须和 Draw 里的一模一样！) ---
	// 如果以后改了 Draw 里的位置，这里也要改
	startBtnY := 50
	btnH := 30
	gap := 15

	// 计算每个按钮的 Y 轴范围
	// 按钮 1: Color
	btn1Top := startBtnY
	btn1Bot := btn1Top + btnH

	// 按钮 2: Glitch
	btn2Top := btn1Bot + gap
	btn2Bot := btn2Top + btnH

	// 【新增】按钮 3: Float
	btn3Top := btn2Bot + gap
	btn3Bot := btn3Top + btnH

	// 【修改】按钮 4: Monitor (往下顺延)
	btn4Top := btn3Bot + gap
	btn4Bot := btn4Top + btnH

	// 按钮 4: Exit (在最底下)
	// 注意：我们在 Draw 里用的是 float32(h) - 50
	// 这里我们需要获取屏幕高度来计算
	_, h := ebiten.WindowSize()
	btnExitTop := h - 50
	btnExitBot := btnExitTop + btnH

	// --- 2. 判定点击 ---

	// 检查点击了 [COLOR]
	if my >= btn1Top && my <= btn1Bot {
		g.ShowColor = !g.ShowColor
		return
	}

	// 检查点击了 [GLITCH]
	if my >= btn2Top && my <= btn2Bot {
		g.ShowGlitch = !g.ShowGlitch
		return
	}

	// 【新增】点击 [FLOAT]
	if my >= btn3Top && my <= btn3Bot {
		g.ShowAnimation = !g.ShowAnimation
		return
	}

	// 【修改】点击 [MONITOR] (注意这里变成了 btn4Top)
	if my >= btn4Top && my <= btn4Bot {
		g.ShowMonitor = !g.ShowMonitor
		return
	}

	// 检查点击了 [EXIT]
	if my >= btnExitTop && my <= btnExitBot {
		g.saveState() // 临死前存个档
		os.Exit(0)    // 彻底关闭程序
	}
}
