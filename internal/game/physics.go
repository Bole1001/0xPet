package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// updatePhysics 处理拖拽、惯性滑行、边缘碰撞与 TPS 控制
func (g *Manager) updatePhysics() {
	// 1. 获取当前绝对坐标与尺寸
	mx, my := ebiten.CursorPosition()
	wx, wy := ebiten.WindowPosition()

	// 【关键修正】获取实时窗口尺寸 (包含可能已经展开的菜单宽度)
	ww, wh := ebiten.WindowSize()

	isClicking := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	isMoving := g.isDragging || math.Abs(g.velX) > 0.1 || math.Abs(g.velY) > 0.1

	// 【关键修正】悬停判定使用实时窗口尺寸 ww, wh
	isHover := mx >= 0 && mx <= ww && my >= 0 && my <= wh

	// 2. 动态调整 TPS：空闲时最低，鼠标悬停时提升，菜单或滑动时保持流畅
	targetTPS := 8
	if g.ShowMenu {
		targetTPS = 30 // 菜单时需要更流畅的响应
	} else if isMoving {
		targetTPS = 60 // 拖拽或惯性状态时保持流畅
	} else if isHover {
		targetTPS = 20 // 鼠标悬停时保持适度响应
	}
	if targetTPS != g.lastTPS {
		ebiten.SetTPS(targetTPS)
		g.lastTPS = targetTPS
	}

	// 3. 拖拽与滑行状态机
	if isClicking {
		// --- 状态 A: 正在被鼠标抓取 ---
		if !g.isDragging {
			g.isDragging = true
			g.dragStartX = mx
			g.dragStartY = my
		} else {
			newX := wx + mx - g.dragStartX
			newY := wy + my - g.dragStartY
			ebiten.SetWindowPosition(newX, newY)

			// 计算即时脱手速度
			g.velX = float64(newX - g.lastWinX)
			g.velY = float64(newY - g.lastWinY)
		}
	} else {
		// --- 状态 B: 松手后的自由滑行 ---
		g.isDragging = false

		if math.Abs(g.velX) > 0.1 || math.Abs(g.velY) > 0.1 {
			// 3.1 应用惯性
			wx += int(g.velX)
			wy += int(g.velY)
			ebiten.SetWindowPosition(wx, wy)

			// 3.2 应用摩擦力衰减
			friction := 0.95
			g.velX *= friction
			g.velY *= friction

			// 3.3 屏幕边缘碰撞检测 (反弹)
			sw, sh := ebiten.ScreenSizeInFullscreen()

			// 左墙
			if wx < 0 {
				wx = 0
				g.velX = -g.velX * 0.6
			}
			// 【关键修正】右墙：使用动态窗口宽度 ww
			if wx+ww > sw {
				wx = sw - ww
				g.velX = -g.velX * 0.6
			}
			// 上墙
			if wy < 0 {
				wy = 0
				g.velY = -g.velY * 0.6
			}
			// 下墙
			if wy+wh > sh {
				wy = sh - wh
				g.velY = -g.velY * 0.6
			}

			ebiten.SetWindowPosition(wx, wy)
		} else {
			// 速度阈值过低直接归零，防止微小抖动
			g.velX = 0
			g.velY = 0
		}
	}

	// 4. 记录最终位置，供下一帧计算速度增量
	finalX, finalY := ebiten.WindowPosition()
	g.lastWinX = finalX
	g.lastWinY = finalY
}
