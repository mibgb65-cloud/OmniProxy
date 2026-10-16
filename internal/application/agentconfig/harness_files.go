package agentconfig

import (
	"encoding/json"
	"errors"
)

// 备份只包含当前 profile 的配置和独立凭据项，另一 profile 的变更不使本项失效。
type harnessBundle struct {
	Config backup  `json:"config"`
	Key    *string `json:"key"`
}

func (s *Service) readClient(client, path string) ([]byte, bool, error) {
	if !isHarness(client) {
		return read(path)
	}
	config, err := snapshotFile(path)
	if err != nil {
		return nil, false, err
	}
	defer clear(config.body)
	credentials, err := snapshotFile(s.paths.HarnessCredentials)
	if err != nil {
		return nil, false, err
	}
	defer clear(credentials.body)
	key, err := harnessCredential(credentials.body, harnessRef(client))
	if err != nil {
		return nil, false, err
	}
	encoded, err := json.Marshal(harnessBundle{backup{config.body, config.exists}, key})
	if err != nil || len(encoded) > maxConfigBytes {
		return nil, false, failure("invalid_config")
	}
	return encoded, config.exists || key != nil, nil
}

func (s *Service) snapshotClient(client, path string) (fileSnapshot, error) {
	body, exists, err := s.readClient(client, path)
	if err != nil {
		return fileSnapshot{}, err
	}
	return fileSnapshot{body, exists, digest(body, exists)}, nil
}

func (s *Service) writeClient(client, path string, body []byte, revision string) error {
	if !isHarness(client) {
		return writeAtomic(path, body, revision)
	}
	var target harnessBundle
	if json.Unmarshal(body, &target) != nil {
		return failure("invalid_config")
	}
	defer clear(target.Config.Body)
	before, err := s.snapshotClient(client, path)
	if err != nil {
		return err
	}
	defer clear(before.body)
	if before.revision != revision {
		return failure("config_changed")
	}
	currentConfig, err := snapshotFile(path)
	if err != nil {
		return err
	}
	defer clear(currentConfig.body)
	credentials, err := snapshotFile(s.paths.HarnessCredentials)
	if err != nil {
		return err
	}
	defer clear(credentials.body)
	var previous harnessBundle
	if json.Unmarshal(before.body, &previous) != nil {
		return failure("invalid_config")
	}
	defer clear(previous.Config.Body)
	if currentConfig.revision != digest(previous.Config.Body, previous.Config.Exists) {
		return failure("config_changed")
	}
	currentKey, keyError := harnessCredential(credentials.body, harnessRef(client))
	if keyError != nil {
		return keyError
	}
	if (currentKey == nil) != (previous.Key == nil) || currentKey != nil && *currentKey != *previous.Key {
		return failure("config_changed")
	}
	next, err := patchHarnessCredential(credentials.body, harnessRef(client), target.Key)
	if err != nil {
		return err
	}
	defer clear(next)
	// 先保存独立凭据，再写引用；第二步失败只回滚本次已写版本，外部变更不得被覆盖。
	if err := writeAtomic(s.paths.HarnessCredentials, next, credentials.revision); err != nil {
		return err
	}
	if target.Config.Exists {
		err = writeAtomic(path, target.Config.Body, currentConfig.revision)
	} else {
		err = removeOwned(path, currentConfig.revision)
	}
	if err != nil {
		return errors.Join(err, rollbackFile(s.paths.HarnessCredentials, digest(next, true), credentials))
	}
	return nil
}

func (s *Service) rollbackClient(client, path, revision string, before fileSnapshot) error {
	if isHarness(client) || before.exists {
		return s.writeClient(client, path, before.body, revision)
	}
	return removeOwned(path, revision)
}
