package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

// 空白挂件，不执行任何逻辑，不绘制任何像素
type BlankGame struct{}

func (g *BlankGame) Update() error             { return nil }
func (g *BlankGame) Draw(screen *ebiten.Image) {}
func (g *BlankGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 200, 200
}

func main() {
	ebiten.SetWindowDecorated(false)
	ebiten.SetScreenTransparent(true)
	ebiten.SetWindowFloating(true)

	// 强制降频，切断所有渲染和逻辑更新压力
	ebiten.SetTPS(5)

	if err := ebiten.RunGame(&BlankGame{}); err != nil {
		log.Fatal(err)
	}
}
