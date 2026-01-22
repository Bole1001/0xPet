package ascii

import (
	"0xPet/internal/entity"
	"image"
	"image/color"
	"strings"
)

// ASCII 字符集 (从黑到白，稍微调整了一下顺序以适应黑底或透明底)
// 你可以根据喜好调整这个字符串
const asciiChars = "@%#*+=-:. "

// Convert 将图片转换为 ASCII 字符串切片
// img: 原始图片对象
// targetWidth: 你希望生成的宠物宽度（字符数），比如 40 或 50
// 【修改】返回值变了！现在返回 (字符串切片, 颜色网格)
func Convert(img image.Image, targetWidth int) ([]string, [][]entity.CharData) {
	bounds := img.Bounds()
	width := bounds.Max.X
	height := bounds.Max.Y

	stepX := width / targetWidth
	if stepX < 1 {
		stepX = 1
	}
	stepY := stepX * 2

	var strResult []string
	// 【新增】初始化网格切片
	var gridResult [][]entity.CharData

	for y := 0; y < height; y += stepY {
		var lineBuilder strings.Builder
		// 【新增】这一行的字符数据切片
		var lineGrid []entity.CharData

		for x := 0; x < width; x += stepX {
			pixel := img.At(x, y)       // 获取原始颜色
			char := pixelToASCII(pixel) // 转成字符

			// 1. 拼接到字符串 (旧逻辑)
			lineBuilder.WriteString(char)

			// 2. 【新增】存入 Grid (新逻辑)
			lineGrid = append(lineGrid, entity.CharData{
				OriginalChar: char,  // 记下原本的
				Char:         char,  // 默认显示原本的
				Color:        pixel, // 记下原始颜色！
			})
		}
		strResult = append(strResult, lineBuilder.String())
		// 【新增】将这一行存入网格
		gridResult = append(gridResult, lineGrid)
	}

	return strResult, gridResult
}

// 你的原始转换逻辑，完全保留
func pixelToASCII(c color.Color) string {
	r, g, b, _ := c.RGBA()
	// Go 的 RGBA 返回 16bit (0-65535)，右移 8 位变成 0-255
	gray := 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)

	// 映射到字符集索引
	idx := int(gray / 255 * float64(len(asciiChars)-1))

	// 防御性编程：防止浮点数精度问题导致 idx 越界
	if idx >= len(asciiChars) {
		idx = len(asciiChars) - 1
	}

	return string(asciiChars[idx])
}
