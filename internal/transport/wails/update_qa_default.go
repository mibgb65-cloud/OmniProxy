//go:build !updateqa

package wails

import (
	"omniproxy/internal/application/updater"
	"omniproxy/internal/platform/profile"
)

func configureUpdateQA(*updater.Options, *profile.Profile) {}
