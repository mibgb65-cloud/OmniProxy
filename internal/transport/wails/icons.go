package wails

import _ "embed"

// 将图标编入桌面适配层，保证托盘与窗口回退图标不依赖运行目录中的外部文件。
//
//go:embed assets/app-icon.png
var appIcon []byte
