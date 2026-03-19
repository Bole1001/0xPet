// Package ascii provides functions and types for converting images to ASCII art
package ascii

import (
	"0xPet/internal/entity"
	"image"
	"image/color"
	"strings"
)

// ASCII 字符集从密集到稀疏，越靠前的字符表示越暗的像素
const asciiChars = "@W#80Oocv:,.. "

//"@80QOo:,. "

// Convert 将图片转换为 ASCII 字符串切片 (高精度区块均值采样版)
func Convert(img image.Image, targetWidth int) ([]string, [][]entity.CharData) {
	bounds := img.Bounds()
	width := bounds.Max.X
	height := bounds.Max.Y

	stepX := width / targetWidth
	if stepX < 1 {
		stepX = 1
	}
	stepY := stepX * 2 // 终端字符通常是 1:2 的长宽比

	var strResult []string
	var gridResult [][]entity.CharData

	for y := 0; y < height; y += stepY {
		var lineBuilder strings.Builder
		var lineGrid []entity.CharData

		// 计算当前区块的 Y 轴物理边界 (防止越界)
		endY := y + stepY
		if endY > height {
			endY = height
		}

		for x := 0; x < width; x += stepX {
			// 计算当前区块的 X 轴物理边界
			endX := x + stepX
			if endX > width {
				endX = width
			}

			// 1. 初始化能量积分器
			var rSum, gSum, bSum, aSum uint64
			var count uint64

			// 2. 遍历该物理区块内的所有真实像素
			for by := y; by < endY; by++ {
				for bx := x; bx < endX; bx++ {
					r, g, b, a := img.At(bx, by).RGBA() // 提取 16-bit 原始颜色
					rSum += uint64(r)
					gSum += uint64(g)
					bSum += uint64(b)
					aSum += uint64(a)
					count++
				}
			}

			// 防御性除零保护
			if count == 0 {
				count = 1
			}

			// 3. 计算物理区块的绝对平均颜色 (降级回 16-bit 以适配 Go 的 Color 接口)
			avgColor := color.RGBA64{
				R: uint16(rSum / count),
				G: uint16(gSum / count),
				B: uint16(bSum / count),
				A: uint16(aSum / count),
			}

			// 4. 使用平均颜色进行字符映射
			char := pixelToASCII(avgColor)

			lineBuilder.WriteString(char)

			lineGrid = append(lineGrid, entity.CharData{
				OriginalChar: char,
				Char:         char,
				Color:        avgColor, // 【关键】将计算出的平均色彩存入数据层，供后续渲染
			})
		}
		strResult = append(strResult, lineBuilder.String())
		gridResult = append(gridResult, lineGrid)
	}

	return strResult, gridResult
}

// pixelToASCII 将单个像素颜色转换为 ASCII 字符
func pixelToASCII(c color.Color) string {
	r, g, b, a := c.RGBA() // 提取全部 4 个通道

	if a < 6553 {
		return " "
	}

	// 计算有效像素的灰度值
	gray := 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)

	idx := int(gray / 255 * float64(len(asciiChars)-1))

	if idx >= len(asciiChars) {
		idx = len(asciiChars) - 1
	}

	return string(asciiChars[idx])
}
