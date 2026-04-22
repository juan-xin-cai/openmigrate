package symlink

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	omlog "github.com/openmigrate/openmigrate/internal/core/log"
	"github.com/openmigrate/openmigrate/internal/core/types"
)

func Resolve(manifest types.Manifest, sourceHome string, logger *omlog.Logger) (types.Manifest, error) {
	seen := make(map[string]types.FileEntry)
	var links []types.LinkRelation
	var totalSize int64

	addEntry := func(entry types.FileEntry) {
		if existing, ok := seen[entry.RelativePath]; ok && existing.SourcePath != "" {
			return
		}
		seen[entry.RelativePath] = entry
		totalSize += entry.Size
	}

	for _, entry := range manifest.Entries {
		if !entry.IsSymlink {
			addEntry(entry)
			continue
		}

		finalPath, warning, cloneable, err := resolveTarget(entry.SourcePath)
		if err != nil {
			if logger != nil {
				logger.Warn("resolve symlink failed", map[string]interface{}{"path": entry.RelativePath, "error": err.Error()})
			}
			continue
		}
		external := cloneable && !sameOrChild(sourceHome, finalPath)
		link := types.LinkRelation{
			LinkRelativePath: entry.RelativePath,
			OriginalTarget:   entry.SymlinkTarget,
			External:         external,
			Warning:          warning,
		}
		if cloneable && !external {
			rel, relErr := filepath.Rel(sourceHome, finalPath)
			if relErr == nil {
				link.TargetRelativePath = filepath.ToSlash(rel)
			}
		} else if cloneable && link.Warning == "" {
			link.Warning = "软链目标在家目录外，导入后保落实体"
		}
		links = append(links, link)
		if logger != nil && link.Warning != "" {
			logger.Warn("symlink warning", map[string]interface{}{"path": entry.RelativePath, "warning": link.Warning})
		}

		if !cloneable {
			continue
		}
		clone, err := cloneResolvedPath(finalPath, entry.RelativePath)
		if err != nil {
			return types.Manifest{}, err
		}
		for _, item := range clone {
			addEntry(item)
		}
		if !external && link.TargetRelativePath != "" && link.TargetRelativePath != entry.RelativePath {
			targetClone, err := cloneResolvedPath(finalPath, link.TargetRelativePath)
			if err != nil {
				return types.Manifest{}, err
			}
			for _, item := range targetClone {
				addEntry(item)
			}
		}
	}

	entries := make([]types.FileEntry, 0, len(seen))
	for _, entry := range seen {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].RelativePath < entries[j].RelativePath })
	return types.Manifest{SourceHome: sourceHome, Entries: entries, Links: links, TotalSize: totalSize}, nil
}

func Restore(targetHome string, links []types.LinkRelation, logger *omlog.Logger) error {
	for _, link := range links {
		if link.External || link.TargetRelativePath == "" {
			continue
		}
		linkPath := filepath.Join(targetHome, filepath.FromSlash(link.LinkRelativePath))
		targetPath := filepath.Join(targetHome, filepath.FromSlash(link.TargetRelativePath))
		relTarget, err := filepath.Rel(filepath.Dir(linkPath), targetPath)
		if err != nil {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
			return err
		}
		tempLink := linkPath + ".tmp-link"
		_ = os.Remove(tempLink)
		if err := os.Symlink(relTarget, tempLink); err != nil {
			if logger != nil {
				logger.Warn("restore symlink failed, keep entity", map[string]interface{}{"path": link.LinkRelativePath, "error": err.Error()})
			}
			continue
		}
		backupPath := linkPath + ".entity-bak"
		_ = os.RemoveAll(backupPath)
		entityExists := false
		if _, err := os.Lstat(linkPath); err == nil {
			entityExists = true
			if err := os.Rename(linkPath, backupPath); err != nil {
				_ = os.Remove(tempLink)
				if logger != nil {
					logger.Warn("backup entity before symlink restore failed", map[string]interface{}{"path": link.LinkRelativePath, "error": err.Error()})
				}
				continue
			}
		}
		if err := os.Rename(tempLink, linkPath); err != nil {
			if entityExists {
				_ = os.Rename(backupPath, linkPath)
			}
			_ = os.Remove(tempLink)
			if logger != nil {
				logger.Warn("restore symlink failed, keep entity", map[string]interface{}{"path": link.LinkRelativePath, "error": err.Error()})
			}
			continue
		}
		if entityExists {
			_ = os.RemoveAll(backupPath)
		}
	}
	return nil
}

func resolveTarget(linkPath string) (string, string, bool, error) {
	visited := make(map[string]struct{})
	current := linkPath
	for depth := 0; ; depth++ {
		if _, ok := visited[current]; ok {
			return current, "检测到循环软链，停止跟随", false, nil
		}
		visited[current] = struct{}{}
		info, err := os.Lstat(current)
		if err != nil {
			return "", "", false, err
		}
		if info.Mode()&os.ModeSymlink == 0 {
			return current, "", true, nil
		}
		if depth >= 2 {
			return current, "软链深度超过 2，已截断跟随", false, nil
		}
		target, err := os.Readlink(current)
		if err != nil {
			return "", "", false, err
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(current), target)
		}
		current = filepath.Clean(target)
	}
}

func cloneResolvedPath(sourcePath, baseRel string) ([]types.FileEntry, error) {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return nil, err
	}
	out := make([]types.FileEntry, 0, 8)
	if !info.IsDir() {
		out = append(out, types.FileEntry{
			SourcePath:   sourcePath,
			RelativePath: filepath.ToSlash(baseRel),
			Mode:         info.Mode(),
			Size:         info.Size(),
			GroupKey:     types.GroupKey(filepath.ToSlash(baseRel)),
		})
		return out, nil
	}
	err = filepath.Walk(sourcePath, func(path string, current os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(sourcePath, path)
		if err != nil {
			return err
		}
		targetRel := baseRel
		if rel != "." {
			targetRel = filepath.Join(baseRel, rel)
		}
		statInfo := current
		if current.Mode()&os.ModeSymlink != 0 {
			statInfo, err = os.Stat(path)
			if err != nil {
				return nil
			}
		}
		out = append(out, types.FileEntry{
			SourcePath:   path,
			RelativePath: filepath.ToSlash(targetRel),
			Mode:         statInfo.Mode(),
			Size:         statInfo.Size(),
			IsDir:        statInfo.IsDir(),
			GroupKey:     types.GroupKey(filepath.ToSlash(targetRel)),
		})
		return nil
	})
	return out, err
}

func sameOrChild(root, candidate string) bool {
	root = filepath.Clean(root)
	candidate = filepath.Clean(candidate)
	return candidate == root || strings.HasPrefix(candidate, root+string(os.PathSeparator))
}
