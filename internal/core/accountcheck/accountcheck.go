package accountcheck

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	omlog "github.com/openmigrate/openmigrate/internal/core/log"
	"github.com/openmigrate/openmigrate/internal/core/types"
)

const desktopRelativeRoot = "Library/Application Support/Claude"

type ownerAccountFile struct {
	OwnerAccountID string `json:"ownerAccountId"`
}

func ExtractSourceAccount(sourceHome string) (string, error) {
	return readOwnerAccount(filepath.Join(sourceHome, filepath.FromSlash(desktopRelativeRoot), "cowork-enabled-cli-ops.json"))
}

func Check(meta types.PackageMeta, targetHome string, skip bool, logger *omlog.Logger) error {
	if meta.OwnerAccountID == "" {
		return nil
	}

	targetPath := filepath.Join(targetHome, filepath.FromSlash(desktopRelativeRoot), "cowork-enabled-cli-ops.json")
	targetID, err := readOwnerAccount(targetPath)
	if err == nil && targetID == meta.OwnerAccountID {
		return nil
	}

	if skip {
		if logger != nil {
			logger.Warn("skip claude desktop session account check", map[string]interface{}{
				"target_path": targetPath,
				"warning":     "sessions directory will produce orphan entries",
			})
		}
		return nil
	}

	return fmt.Errorf("%w: 请先打开 Claude Desktop 并登录同一 Anthropic 账号", types.ErrAccountMismatch)
}

func readOwnerAccount(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var payload ownerAccountFile
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", err
	}
	if payload.OwnerAccountID == "" {
		return "", os.ErrNotExist
	}
	return payload.OwnerAccountID, nil
}
