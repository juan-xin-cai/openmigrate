package writer

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	omlog "github.com/openmigrate/openmigrate/internal/core/log"
	"github.com/openmigrate/openmigrate/internal/core/types"
)

func CollectFiles(root string) ([]types.FileEntry, error) {
	out := make([]types.FileEntry, 0, 32)
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." || strings.HasPrefix(rel, ".openmigrate/") {
			if info.IsDir() && strings.HasPrefix(rel, ".openmigrate/") {
				return filepath.SkipDir
			}
			return nil
		}
		out = append(out, types.FileEntry{
			SourcePath:   path,
			RelativePath: rel,
			Mode:         info.Mode(),
			Size:         info.Size(),
			IsDir:        info.IsDir(),
			GroupKey:     types.GroupKey(rel),
		})
		return nil
	})
	return out, err
}

func Write(files []types.FileEntry, targetHome string, decisions types.ConflictDecision, conflicts types.ConflictReport, logger *omlog.Logger) ([]string, []string, []string, error) {
	stageRoot, err := os.MkdirTemp("", "openmigrate-write-*")
	if err != nil {
		return nil, nil, nil, err
	}
	defer os.RemoveAll(stageRoot)

	groupEntries := make(map[string][]types.FileEntry)
	for _, entry := range files {
		groupEntries[entry.GroupKey] = append(groupEntries[entry.GroupKey], entry)
	}
	groups := make([]string, 0, len(groupEntries))
	for group := range groupEntries {
		groups = append(groups, group)
	}
	sort.Strings(groups)
	if err := validateConflictDecisions(conflicts, decisions); err != nil {
		return nil, nil, nil, err
	}

	var written []string
	var updated []string
	var skipped []string

	for _, group := range groups {
		action := decisions.Actions[group]
		if action == types.DecisionKeepTarget || action == types.DecisionSkip {
			skipped = append(skipped, group)
			continue
		}
		for _, entry := range groupEntries[group] {
			dest := filepath.Join(stageRoot, filepath.FromSlash(entry.RelativePath))
			if entry.IsDir {
				if err := os.MkdirAll(dest, entry.Mode); err != nil {
					return nil, nil, nil, err
				}
				continue
			}
			if err := copyFile(entry.SourcePath, dest, entry.Mode); err != nil {
				return nil, nil, nil, err
			}
		}
		if group == ".claude/settings.json" {
			if err := mergeSettings(filepath.Join(stageRoot, ".claude", "settings.json"), filepath.Join(targetHome, ".claude", "settings.json"), decisions.Actions); err != nil {
				return nil, nil, nil, err
			}
		}
	}

	type backupItem struct {
		Target string
		Backup string
	}
	backups := make([]backupItem, 0, len(groups))
	backupRoot := filepath.Join(stageRoot, ".backups")

	restore := func() {
		for i := len(backups) - 1; i >= 0; i-- {
			item := backups[i]
			_ = os.RemoveAll(item.Target)
			if _, err := os.Lstat(item.Backup); err == nil {
				_ = os.MkdirAll(filepath.Dir(item.Target), 0o755)
				_ = os.Rename(item.Backup, item.Target)
			}
		}
	}

	for _, group := range groups {
		action := decisions.Actions[group]
		if action == types.DecisionKeepTarget || action == types.DecisionSkip {
			continue
		}
		staged := filepath.Join(stageRoot, filepath.FromSlash(group))
		if _, err := os.Lstat(staged); err != nil {
			continue
		}
		target := filepath.Join(targetHome, filepath.FromSlash(group))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			restore()
			return nil, nil, nil, err
		}
		backup := filepath.Join(backupRoot, filepath.FromSlash(group))
		if _, err := os.Lstat(target); err == nil {
			if err := os.MkdirAll(filepath.Dir(backup), 0o755); err != nil {
				restore()
				return nil, nil, nil, err
			}
			if err := os.Rename(target, backup); err != nil {
				restore()
				return nil, nil, nil, err
			}
			backups = append(backups, backupItem{Target: target, Backup: backup})
			updated = append(updated, group)
		} else {
			written = append(written, group)
		}
		if err := os.Rename(staged, target); err != nil {
			restore()
			return nil, nil, nil, err
		}
	}

	if logger != nil {
		logger.Info("writer commit finished", map[string]interface{}{"written": len(written), "updated": len(updated), "skipped": len(skipped)})
	}
	return written, updated, skipped, nil
}

func validateConflictDecisions(conflicts types.ConflictReport, decisions types.ConflictDecision) error {
	for bucketName, bucket := range conflicts.Buckets {
		for _, item := range bucket.Conflicts {
			if _, ok := decisions.Actions[item.Key]; ok {
				continue
			}
			if bucketName == "settings" {
				return fmt.Errorf("%w: %s", types.ErrConflictDecisionRequired, item.Key)
			}
			return fmt.Errorf("%w: %s", types.ErrConflictDecisionRequired, item.Key)
		}
	}
	return nil
}

func mergeSettings(stagedPath, targetPath string, actions map[string]types.DecisionAction) error {
	hasSettingsAction := false
	for key := range actions {
		if strings.HasPrefix(key, "settings:") {
			hasSettingsAction = true
			break
		}
	}
	if !hasSettingsAction {
		return nil
	}
	staged, err := readMap(stagedPath)
	if err != nil {
		return err
	}
	target, _ := readMap(targetPath)
	for key, action := range actions {
		if !strings.HasPrefix(key, "settings:") || (action != types.DecisionKeepTarget && action != types.DecisionSkip) {
			continue
		}
		field := strings.TrimPrefix(key, "settings:")
		if value, ok := target[field]; ok {
			staged[field] = value
		} else {
			delete(staged, field)
		}
	}
	data, err := json.MarshalIndent(staged, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(stagedPath, data, 0o644)
}

func readMap(path string) (map[string]interface{}, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return map[string]interface{}{}, err
	}
	out := make(map[string]interface{})
	if len(data) == 0 {
		return out, nil
	}
	return out, json.Unmarshal(data, &out)
}

func copyFile(source, target string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}
