//go:build windows

package appinstall

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

type handshake struct {
	Ready bool   `json:"ready"`
	Error string `json:"error"`
}

// Launch 把辅助程序复制到缓存后启动，避免自身占用待替换的程序文件。
func Launch(request Request) error {
	target, err := os.Executable()
	if err != nil {
		return errors.New("update: install_unavailable")
	}
	scope, err := installation(target)
	if err != nil {
		return err
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return errors.New("update: helper_failed")
	}
	helper := filepath.Join(filepath.Dir(request.Package), "runner-"+hex.EncodeToString(nonce[:])+".exe")
	if err := copyExecutable(target, helper); err != nil {
		return errors.New("update: helper_failed")
	}
	cmd := exec.Command(helper, "--omniproxy-update-helper", request.Package, request.Digest, strconv.FormatInt(request.Size, 10), target, request.Receipt, request.Version, scope)
	cmd.Dir = filepath.Dir(helper)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return errors.New("update: helper_failed")
	}
	if err := cmd.Start(); err != nil {
		return errors.New("update: helper_failed")
	}
	var response handshake
	err = json.NewDecoder(bufio.NewReader(io.LimitReader(stdout, 1024))).Decode(&response)
	stdout.Close()
	if err != nil || !response.Ready {
		cmd.Wait()
		if response.Error != "" {
			return errors.New(response.Error)
		}
		return errors.New("update: helper_failed")
	}
	return cmd.Process.Release()
}

func copyExecutable(source, target string) (err error) {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := out.Close()
		if err == nil {
			err = closeErr
		}
	}()
	_, err = io.Copy(out, in)
	return err
}
