package profile

import "fmt"

// Close behavior values persisted in the profile config.
const (
	CloseExit   = "exit"
	CloseToTray = "tray"
)

// ParseCloseBehavior validates a stored close behavior. Older configs without
// the field keep the previous behavior: closing the window exits the app.
func ParseCloseBehavior(value string) (string, error) {
	switch value {
	case "", CloseExit:
		return CloseExit, nil
	case CloseToTray:
		return CloseToTray, nil
	default:
		return "", fmt.Errorf("profile: invalid close behavior %q", value)
	}
}

// SetCloseBehavior validates and persists the close behavior.
func SetCloseBehavior(kind Kind, value string) error {
	if value != CloseExit && value != CloseToTray {
		return fmt.Errorf("profile: invalid close behavior %q", value)
	}
	cfg, err := LoadConfig(kind)
	if err != nil {
		return err
	}
	cfg.CloseBehavior = value
	return SaveConfig(kind, cfg)
}
