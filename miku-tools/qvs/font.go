package qvs

import (
	"fmt"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
)

func getStringDimensions(fontPath string, fontSize float64, s string) (width, height int, err error) {
	// 读取字体文件
	fontBytes, err := os.ReadFile(fontPath)
	if err != nil {
		return 0, 0, fmt.Errorf("read font fail: %v", err)
	}

	// 解析字体
	fontTT, err := sfnt.Parse(fontBytes)
	if err != nil {
		return 0, 0, fmt.Errorf("parse font fail: %v", err)
	}

	// 创建字体Face（默认使用96 DPI）
	face, err := opentype.NewFace(fontTT, &opentype.FaceOptions{
		Size:    fontSize,
		DPI:     96,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return 0, 0, fmt.Errorf("create face fail: %v", err)
	}

	// 测量字符串宽度
	width = font.MeasureString(face, s).Ceil()

	// 计算字符串高度
	metrics := face.Metrics()
	ascent := metrics.Ascent.Ceil()
	descent := metrics.Descent.Ceil()
	height = ascent + descent

	return width, height, nil
}

func Font() {
	// 示例用法
	width, height, err := getStringDimensions("/System/Library/Fonts/Symbol.ttf", 12, "Hello World")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("Width: %dpx, Height: %dpx\n", width, height)
}
