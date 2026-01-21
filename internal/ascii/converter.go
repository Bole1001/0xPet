package ascii

import (
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
func Convert(img image.Image, targetWidth int) []string {
	bounds := img.Bounds()
	width := bounds.Max.X
	height := bounds.Max.Y

	// 1. 计算缩放步长 (你的核心逻辑)
	stepX := width / targetWidth
	if stepX < 1 {
		stepX = 1
	}

	// [关键点] 矫正纵横比
	// 终端字符的高通常是宽的 2 倍，所以 Y 轴采样步长要翻倍
	stepY := stepX * 2

	var result []string

	// 2. 遍历像素 (采样)
	for y := 0; y < height; y += stepY {
		var line strings.Builder // 使用 Builder 拼接字符串更高效
		for x := 0; x < width; x += stepX {
			// 获取像素颜色
			pixel := img.At(x, y)
			// 转换并追加到当前行
			line.WriteString(pixelToASCII(pixel))
		}
		// 将这一行加入结果集
		result = append(result, line.String())
	}

	return result
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
