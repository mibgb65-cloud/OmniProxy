// Package legacy 只读解析 v1 数据；不修改源文件，也不将凭据返回给前端。
package legacy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Token 只读取迁移需要的旧字段，忽略历史用量和错误正文。
type Token struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Provider       string `json:"provider"`
	CredentialType string `json:"credentialType"`
	Region         string `json:"region"`
	BaseURL        string `json:"baseUrl"`
	Value          string `json:"tokenValue"`
	Disabled       bool   `json:"disabled"`
}

// Source 的密钥数据仅留在后端，Snapshot 用于检测预览后的文件变化。
type Source struct {
	Directory string
	Snapshot  string
	Tokens    []Token
	Endpoints map[string]string
}

// Read 对固定文件名做有界读取，不跟随文件符号链接。
func Read(directory string) (Source, error) {
	var source Source
	if !filepath.IsAbs(directory) || len(directory) > 4096 || strings.ContainsAny(directory, "\x00\r\n") {
		return source, errors.New("legacy: invalid_directory")
	}
	directory = filepath.Clean(directory)
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return source, errors.New("legacy: invalid_directory")
	}
	data, err := readFile(filepath.Join(directory, "tokens.json"), 16<<20)
	if err != nil {
		return source, errors.New("legacy: unreadable_tokens")
	}
	defer clear(data)
	if json.Unmarshal(data, &source.Tokens) != nil || source.Tokens == nil || len(source.Tokens) > 1000 {
		return Source{}, errors.New("legacy: invalid_tokens")
	}
	config, err := readFile(filepath.Join(directory, "config.json"), 2<<20)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Source{}, errors.New("legacy: unreadable_config")
	}
	defer clear(config)
	var values map[string]json.RawMessage
	if len(config) > 0 && json.Unmarshal(config, &values) != nil {
		return Source{}, errors.New("legacy: invalid_config")
	}
	source.Endpoints = make(map[string]string)
	for field, value := range values {
		if strings.HasSuffix(field, "BaseUrl") {
			var endpoint string
			if json.Unmarshal(value, &endpoint) != nil || len(endpoint) > 2048 {
				return Source{}, errors.New("legacy: invalid_config")
			}
			source.Endpoints[field] = endpoint
		}
	}
	digest := sha256.New()
	digest.Write(data)
	digest.Write([]byte{0})
	digest.Write(config)
	source.Directory, source.Snapshot = directory, hex.EncodeToString(digest.Sum(nil))
	return source, nil
}

func readFile(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, errors.New("legacy: invalid_file")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, errors.New("legacy: changed_file")
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || int64(len(data)) > limit {
		clear(data)
		return nil, errors.New("legacy: invalid_file")
	}
	return data, nil
}

// DefaultDirectory 识别 v1 自定义目录，不混用开发与生产数据。
func DefaultDirectory(development bool) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", errors.New("legacy: home_unavailable")
	}
	name, configName := ".omniproxy", "OmniProxy"
	if development {
		name, configName = ".omniproxy-dev", "OmniProxyDev"
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", errors.New("legacy: config_directory_unavailable")
	}
	data, err := readFile(filepath.Join(base, configName, "bootstrap.json"), 16384)
	if errors.Is(err, os.ErrNotExist) {
		return filepath.Join(home, name), nil
	}
	if err != nil {
		return "", errors.New("legacy: invalid_bootstrap")
	}
	var bootstrap struct {
		DataDir string `json:"dataDir"`
	}
	if json.Unmarshal(data, &bootstrap) != nil || !filepath.IsAbs(bootstrap.DataDir) || len(bootstrap.DataDir) > 4096 {
		return "", errors.New("legacy: invalid_bootstrap")
	}
	return filepath.Clean(bootstrap.DataDir), nil
}
