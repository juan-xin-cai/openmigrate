package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/openmigrate/openmigrate/internal/core/types"
	"github.com/openmigrate/openmigrate/internal/core/whitelist"
)

func Build(sourceHome string, cfg types.AgentConfig) (types.Manifest, error) {
	entries := make([]types.FileEntry, 0, 64)
	seen := make(map[string]struct{})
	var totalSize int64

	for _, root := range cfg.Roots {
		rootPath := filepath.Join(sourceHome, filepath.FromSlash(root))
		info, err := os.Lstat(rootPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return types.Manifest{}, err
		}
		if !info.IsDir() {
			rel := filepath.ToSlash(root)
			strategy := strategyFor(rel, false, cfg)
			if strategy == types.StrategyExclude {
				continue
			}
			entry, err := newEntry(rootPath, rel, strategy)
			if err != nil {
				return types.Manifest{}, err
			}
			if _, ok := seen[rel]; !ok {
				entries = append(entries, entry)
				seen[rel] = struct{}{}
				totalSize += entry.Size
			}
			continue
		}

		err = filepath.Walk(rootPath, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			rel, err := filepath.Rel(sourceHome, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			strategy := strategyFor(rel, info.IsDir(), cfg)
			if strategy == types.StrategyExclude {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if _, ok := seen[rel]; ok {
				return nil
			}
			entry, err := newEntry(path, rel, strategy)
			if err != nil {
				return err
			}
			entries = append(entries, entry)
			seen[rel] = struct{}{}
			totalSize += entry.Size
			return nil
		})
		if err != nil {
			return types.Manifest{}, err
		}
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].RelativePath < entries[j].RelativePath })
	return types.Manifest{SourceHome: sourceHome, Entries: entries, TotalSize: totalSize}, nil
}

func strategyFor(relPath string, isDir bool, cfg types.AgentConfig) types.Strategy {
	matched := types.StrategyExclude
	bestLen := -1
	for _, entry := range cfg.Entries {
		if !whitelist.Match(relPath, entry.Path) {
			continue
		}
		if len(entry.Path) >= bestLen {
			bestLen = len(entry.Path)
			matched = entry.Strategy
		}
	}
	if matched != types.StrategyExclude || !isDir {
		return matched
	}
	clean := filepath.ToSlash(filepath.Clean(relPath))
	for _, entry := range cfg.Entries {
		if entry.Strategy == types.StrategyExclude {
			continue
		}
		pattern := filepath.ToSlash(strings.TrimSuffix(entry.Path, "/**"))
		if pattern == clean || strings.HasPrefix(pattern, clean+"/") {
			return types.StrategyInclude
		}
	}
	return matched
}

func newEntry(absPath, relPath string, strategy types.Strategy) (types.FileEntry, error) {
	info, err := os.Lstat(absPath)
	if err != nil {
		return types.FileEntry{}, err
	}
	entry := types.FileEntry{
		SourcePath:   absPath,
		RelativePath: relPath,
		Mode:         info.Mode(),
		Size:         info.Size(),
		Strategy:     strategy,
		IsDir:        info.IsDir(),
		IsSymlink:    info.Mode()&os.ModeSymlink != 0,
		GroupKey:     types.GroupKey(relPath),
	}
	if entry.IsSymlink {
		target, err := os.Readlink(absPath)
		if err != nil {
			return types.FileEntry{}, fmt.Errorf("readlink %s: %w", absPath, err)
		}
		entry.SymlinkTarget = target
	}
	return entry, nil
}
