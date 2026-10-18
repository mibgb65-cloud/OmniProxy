// Package appinstall 提供受限的 Windows 安装辅助模式，不依赖桌面框架。
package appinstall

// Request 的路径及摘要仅由更新服务根据已验证资产生成。
type Request struct {
	Package, Digest, Receipt, Version string
	Size                              int64
}

// Receipt 由原用户辅助进程在安装结束后写入，再启动新应用读取。
type Receipt struct {
	Version string `json:"version"`
	Success bool   `json:"success"`
	Code    string `json:"code"`
}
