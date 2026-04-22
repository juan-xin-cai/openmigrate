package conflict

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/openmigrate/openmigrate/internal/core/types"
)

func Detect(packageRoot, targetHome string) (types.ConflictReport, error) {
	report := types.ConflictReport{Buckets: make(map[string]types.ConflictBucket)}

	if err := compareSettings(packageRoot, targetHome, &report); err != nil {
		return report, err
	}

	packageGroups, err := collectGroups(packageRoot)
	if err != nil {
		return report, err
	}

	targetGroups, err := collectTargetGroups(packageGroups, targetHome)
	if err != nil {
		return report, err
	}

	keys := make(map[string]struct{})
	for key := range packageGroups {
		if key == ".claude/settings.json" {
			continue
		}
		keys[key] = struct{}{}
	}
	for key := range targetGroups {
		if key == ".claude/settings.json" {
			continue
		}
		keys[key] = struct{}{}
	}

	ordered := make([]string, 0, len(keys))
	for key := range keys {
		ordered = append(ordered, key)
	}
	sort.Strings(ordered)

	for _, key := range ordered {
		bucketName := classifyGroup(key)
		pkgPath, inPackage := packageGroups[key]
		targetPath, inTarget := targetGroups[key]
		item := types.ConflictItem{Type: bucketName, Key: key, PackagePath: pkgPath, TargetPath: targetPath}
		switch {
		case inPackage && !inTarget:
			appendBucket(&report, bucketName, "addition", item)
		case !inPackage && inTarget:
			appendBucket(&report, bucketName, "target", item)
		default:
			same, err := sameTree(pkgPath, targetPath)
			if err != nil {
				return report, err
			}
			if !same {
				item.Reason = "内容不同"
				appendBucket(&report, bucketName, "conflict", item)
			}
		}
	}

	return report, nil
}

func compareSettings(packageRoot, targetHome string, report *types.ConflictReport) error {
	packageSettings := filepath.Join(packageRoot, ".claude", "settings.json")
	targetSettings := filepath.Join(targetHome, ".claude", "settings.json")
	pkgMap, _ := readJSONMap(packageSettings)
	targetMap, _ := readJSONMap(targetSettings)
	keys := make(map[string]struct{})
	for key := range pkgMap {
		keys[key] = struct{}{}
	}
	for key := range targetMap {
		keys[key] = struct{}{}
	}
	for key := range keys {
		item := types.ConflictItem{
			Type:        "settings",
			Key:         "settings:" + key,
			PackagePath: packageSettings,
			TargetPath:  targetSettings,
		}
		pkgValue, okPkg := pkgMap[key]
		targetValue, okTarget := targetMap[key]
		switch {
		case okPkg && !okTarget:
			appendBucket(report, "settings", "addition", item)
		case !okPkg && okTarget:
			appendBucket(report, "settings", "target", item)
		case okPkg && okTarget && !reflect.DeepEqual(pkgValue, targetValue):
			item.Reason = "settings key 冲突"
			appendBucket(report, "settings", "conflict", item)
		}
	}
	return nil
}

func collectGroups(root string) (map[string]string, error) {
	out := make(map[string]string)
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if strings.HasPrefix(rel, ".openmigrate/") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		group := types.GroupKey(rel)
		if group == "" {
			return nil
		}
		if _, ok := out[group]; !ok {
			out[group] = filepath.Join(root, filepath.FromSlash(group))
		}
		return nil
	})
	return out, err
}

func collectTargetGroups(packageGroups map[string]string, targetHome string) (map[string]string, error) {
	out := make(map[string]string)
	parents := map[string]struct{}{}
	for key := range packageGroups {
		parent := parentBucketRoot(key)
		if parent != "" {
			parents[parent] = struct{}{}
		}
		targetPath := filepath.Join(targetHome, filepath.FromSlash(key))
		if _, err := os.Lstat(targetPath); err == nil {
			out[key] = targetPath
		}
	}
	for parent := range parents {
		parentPath := filepath.Join(targetHome, filepath.FromSlash(parent))
		entries, err := ioutil.ReadDir(parentPath)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			key := filepath.ToSlash(filepath.Join(parent, entry.Name()))
			if _, ok := out[key]; ok {
				continue
			}
			out[key] = filepath.Join(parentPath, entry.Name())
		}
	}
	return out, nil
}

func sameTree(left, right string) (bool, error) {
	leftHash, err := hashPath(left)
	if err != nil {
		return false, err
	}
	rightHash, err := hashPath(right)
	if err != nil {
		return false, err
	}
	return leftHash == rightHash, nil
}

func hashPath(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	hasher := sha256.New()
	if !info.IsDir() {
		file, err := os.Open(path)
		if err != nil {
			return "", err
		}
		defer file.Close()
		if _, err := io.Copy(hasher, file); err != nil {
			return "", err
		}
		return hex.EncodeToString(hasher.Sum(nil)), nil
	}
	err = filepath.Walk(path, func(current string, currentInfo os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(path, current)
		if err != nil {
			return err
		}
		_, _ = hasher.Write([]byte(rel))
		_, _ = hasher.Write([]byte(currentInfo.Mode().String()))
		if currentInfo.IsDir() {
			return nil
		}
		file, err := os.Open(current)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(hasher, file)
		return err
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func appendBucket(report *types.ConflictReport, bucketName, kind string, item types.ConflictItem) {
	bucket := report.Buckets[bucketName]
	switch kind {
	case "addition":
		bucket.Additions = append(bucket.Additions, item)
	case "conflict":
		bucket.Conflicts = append(bucket.Conflicts, item)
	case "target":
		bucket.TargetOnly = append(bucket.TargetOnly, item)
	}
	report.Buckets[bucketName] = bucket
}

func classifyGroup(key string) string {
	switch {
	case strings.HasPrefix(key, ".claude/skills/"):
		return "skills"
	case strings.HasPrefix(key, ".claude/projects/"):
		return "projects"
	case strings.HasPrefix(key, ".claude/plugins/"):
		return "plugins"
	case strings.HasPrefix(key, ".claude/history.jsonl"):
		return "history"
	case strings.HasPrefix(key, ".claude/settings.json"):
		return "settings"
	default:
		return "paths"
	}
}

func parentBucketRoot(key string) string {
	switch {
	case strings.HasPrefix(key, ".claude/skills/"):
		return ".claude/skills"
	case strings.HasPrefix(key, ".claude/projects/"):
		return ".claude/projects"
	case strings.HasPrefix(key, ".claude/plugins/"):
		return ".claude/plugins"
	}
	return ""
}

func readJSONMap(path string) (map[string]interface{}, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return map[string]interface{}{}, err
	}
	out := make(map[string]interface{})
	if len(data) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return map[string]interface{}{}, err
	}
	return out, nil
}
