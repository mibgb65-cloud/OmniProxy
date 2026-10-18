package agentconfig

import (
	"encoding/json"
	"errors"
	"path/filepath"
)

// 双文件备份只存固定顺序的字节与存在状态，不接受来自配置的目标路径。
type configBundle struct {
	Files []backup `json:"files"`
}

func bundlePaths(client, path string) []string {
	if client == "pi" {
		return []string{path, filepath.Join(filepath.Dir(path), "settings.json")}
	}
	return []string{path, filepath.Join(filepath.Dir(path), "custom_providers", "omniproxy.json")}
}

func decodeBundle(body []byte) (configBundle, error) {
	var bundle configBundle
	if len(body) > maxConfigBytes || json.Unmarshal(body, &bundle) != nil || len(bundle.Files) != 2 {
		bundle.clear()
		return configBundle{}, failure("invalid_config")
	}
	for _, file := range bundle.Files {
		if len(file.Body) > maxConfigBytes || !file.Exists && len(file.Body) != 0 {
			bundle.clear()
			return configBundle{}, failure("invalid_config")
		}
	}
	return bundle, nil
}

func (bundle configBundle) clear() {
	for _, file := range bundle.Files {
		clear(file.Body)
	}
}

func encodeBundle(bundle configBundle) ([]byte, error) {
	body, err := json.Marshal(bundle)
	if err != nil || len(body) > maxConfigBytes {
		clear(body)
		return nil, failure("unsafe_config")
	}
	return body, nil
}

func readBundle(client, path string) ([]byte, bool, error) {
	bundle := configBundle{}
	defer func() { bundle.clear() }()
	exists := false
	for _, file := range bundlePaths(client, path) {
		body, found, err := read(file)
		if err != nil {
			return nil, false, err
		}
		bundle.Files = append(bundle.Files, backup{body, found})
		exists = exists || found
	}
	body, err := encodeBundle(bundle)
	return body, exists, err
}

func writeBundle(client, path string, body []byte, revision string) error {
	target, err := decodeBundle(body)
	if err != nil {
		return err
	}
	defer target.clear()
	beforeBody, exists, err := readBundle(client, path)
	if err != nil {
		return err
	}
	defer clear(beforeBody)
	if digest(beforeBody, exists) != revision {
		return failure("config_changed")
	}
	before, err := decodeBundle(beforeBody)
	if err != nil {
		return err
	}
	defer before.clear()
	paths := bundlePaths(client, path)
	for index, next := range target.Files {
		old := before.Files[index]
		expected := digest(old.Body, old.Exists)
		if next.Exists {
			err = writeAtomic(paths[index], next.Body, expected)
		} else {
			err = removeOwned(paths[index], expected)
		}
		if err == nil {
			continue
		}
		// 部分写入失败只回滚本次已提交的版本；外部并发修改仍由逐文件摘要保护。
		for previous := index - 1; previous >= 0; previous-- {
			old, written := before.Files[previous], target.Files[previous]
			err = errors.Join(err, rollbackFile(paths[previous], digest(written.Body, written.Exists), fileSnapshot{body: old.Body, exists: old.Exists}))
		}
		return err
	}
	return nil
}
