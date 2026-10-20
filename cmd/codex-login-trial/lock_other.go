//go:build !windows

package main

import "errors"

func lockTrial(string) (func(), error) {
	return nil, errors.New("旧版登录试验目前仅支持 Windows")
}
