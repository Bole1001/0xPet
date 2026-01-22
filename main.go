package main

import (
	"0xPet/internal/game"
	"0xPet/internal/monitor"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	// 1. 基础窗口设置
	ebiten.SetWindowDecorated(false)  // 无边框
	ebiten.SetScreenTransparent(true) // 透明背景
	ebiten.SetWindowFloating(true)    // 始终置顶
	ebiten.SetWindowTitle("0xPet")

	ebiten.SetWindowSize(100, 100)

	// 【新增】启动系统监控 (后台线程开始每秒采集数据)
	monitor.Start()

	// 2. 初始化逻辑
	mgr := &game.Manager{}
	mgr.Init() // 这里面会计算并重新 SetWindowSize

	// 3. 启动
	if err := ebiten.RunGame(mgr); err != nil {
		log.Fatal(err)
	}
}
