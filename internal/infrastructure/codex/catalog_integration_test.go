package codex

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	catalog "omniproxy/internal/core/modelcatalog"
)

// 显式启用时核对模型目录、默认模型和会话覆盖；授权、配置和缓存均位于临时目录。
func TestInstalledCodexLocalModelCatalog(t *testing.T) {
	if os.Getenv("OMNIPROXY_CODEX_CATALOG_QA") != "1" {
		t.Skip("仅在本机隔离验收时启用")
	}
	path := os.Getenv("OMNIPROXY_CODEX_TEST_EXECUTABLE")
	if path == "" {
		var err error
		path, err = exec.LookPath("codex")
		if err != nil {
			t.Fatal("Codex core unavailable")
		}
	}
	if !filepath.IsAbs(path) {
		t.Fatal("Codex core path must be absolute")
	}
	home := t.TempDir()
	body, err := catalog.CodexJSON([]catalog.Entry{{ID: "qa-first", Name: "First QA Model", ProviderKind: "openai", Metadata: catalog.Metadata{InputModalities: []string{"text", "image"}}}, {ID: "qa-second", Name: "Second QA Model"}})
	if err != nil {
		t.Fatal(err)
	}
	catalogPath := filepath.Join(home, "models.json")
	if err := os.WriteFile(catalogPath, body, 0600); err != nil {
		t.Fatal(err)
	}
	config := "model_provider='qa-provider'\nmodel='qa-second'\nmodel_catalog_json=" + strconv.Quote(filepath.ToSlash(catalogPath)) + "\n[model_providers.qa-provider]\nname='Catalog QA'\nbase_url='http://127.0.0.1:1/v1'\nwire_api='responses'\nenv_key='OMNIPROXY_QA_MODEL_KEY'\n[features]\napi_key_model_discovery=false\n"
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	work := filepath.Join(home, "work")
	if err := os.Mkdir(work, 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, path, "app-server", "--listen", "stdio://", "-c", `cli_auth_credentials_store="file"`)
	hideWindow(command)
	command.Env = append(isolatedEnvironment(home), "OMNIPROXY_QA_MODEL_KEY=qa-only-not-a-real-credential")
	command.Dir = work
	command.Env = append(command.Env, "USERPROFILE="+home, "HOME="+home, "APPDATA="+home, "LOCALAPPDATA="+home)
	command.Stderr = io.Discard
	input, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	output, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	client := &rpcClient{cmd: command, input: input, pending: map[uint64]chan wireResponse{}, notices: make(chan notification, 32), done: make(chan struct{})}
	go client.read(output)
	defer client.Close()
	var initialized struct{}
	if err := client.call(ctx, "initialize", map[string]any{"clientInfo": map[string]string{"name": "omniproxy_catalog_qa", "version": "0.1.0"}, "capabilities": map[string]bool{"experimentalApi": true}}, &initialized); err != nil {
		t.Fatal(err)
	}
	if err := client.write(map[string]any{"method": "initialized"}); err != nil {
		t.Fatal(err)
	}
	var response struct {
		Data []struct {
			Model           string   `json:"model"`
			Name            string   `json:"displayName"`
			InputModalities []string `json:"inputModalities"`
		} `json:"data"`
	}
	if err := client.call(ctx, "model/list", map[string]any{"includeHidden": false}, &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Data) != 2 || response.Data[0].Model != "qa-first" || response.Data[1].Model != "qa-second" || response.Data[0].Name != "First QA Model" {
		t.Fatal("installed core did not consume the generated catalog")
	}
	if len(response.Data[0].InputModalities) != 2 || response.Data[0].InputModalities[1] != "image" || len(response.Data[1].InputModalities) != 1 || response.Data[1].InputModalities[0] != "text" {
		t.Fatal("installed core did not preserve model image capability")
	}
	var effective struct {
		Config struct {
			Model string `json:"model"`
		} `json:"config"`
	}
	if err := client.call(ctx, "config/read", map[string]any{"includeLayers": false}, &effective); err != nil || effective.Config.Model != "qa-second" {
		t.Fatal("installed core did not read the configured default model", err)
	}
	for _, explicit := range []bool{false, true} {
		params := map[string]any{"cwd": work, "ephemeral": true, "approvalPolicy": "never", "sandbox": "read-only"}
		want := "qa-second"
		if explicit {
			params["model"], want = "qa-first", "qa-first"
		}
		var started struct {
			Model string `json:"model"`
		}
		if err := client.call(ctx, "thread/start", params, &started); err != nil || started.Model != want {
			t.Fatal("installed core did not respect default or explicit thread model", err)
		}
	}

}
