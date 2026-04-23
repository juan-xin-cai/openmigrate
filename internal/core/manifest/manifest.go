package manifest

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/openmigrate/openmigrate/internal/core/types"
	"github.com/openmigrate/openmigrate/internal/core/whitelist"
)

func Build(params types.ManifestParams, cfgs ...types.AgentConfig) (types.Manifest, error) {
	if len(params.OnlyScopes) > 0 && len(params.ExcludeScopes) > 0 {
		return types.Manifest{}, types.ErrConflictingScopeFilter
	}

	selected := make(map[string]types.FileEntry)
	for _, cfg := range cfgs {
		if err := collectEntries(params.SourceHome, cfg, selected); err != nil {
			return types.Manifest{}, err
		}
	}

	entries := make([]types.FileEntry, 0, len(selected))
	for _, entry := range selected {
		if !includeByScope(entry, params) {
			continue
		}
		if params.NoHistory && isHistoryJSONL(entry.RelativePath) {
			continue
		}
		entries = append(entries, entry)
	}

	entries, err := addParentDirs(params.SourceHome, entries)
	if err != nil {
		return types.Manifest{}, err
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].RelativePath < entries[j].RelativePath })

	var totalSize int64
	for _, entry := range entries {
		totalSize += entry.Size
	}
	return types.Manifest{SourceHome: params.SourceHome, Entries: entries, TotalSize: totalSize}, nil
}

func collectEntries(sourceHome string, cfg types.AgentConfig, selected map[string]types.FileEntry) error {
	for _, root := range cfg.Roots {
		rootPath := filepath.Join(sourceHome, filepath.FromSlash(root))
		info, err := os.Lstat(rootPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if !info.IsDir() {
			if err := maybeAddEntry(sourceHome, rootPath, filepath.ToSlash(root), false, cfg, selected); err != nil {
				return err
			}
			continue
		}

		err = filepath.Walk(rootPath, func(current string, walkInfo os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			rel, err := filepath.Rel(sourceHome, current)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			spec, matched := bestEntry(rel, cfg)
			if matched && spec.Strategy == types.StrategyExclude {
				if walkInfo.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if walkInfo.IsDir() {
				return nil
			}
			if !matched {
				return nil
			}
			return addEntry(current, rel, spec, selected)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func maybeAddEntry(sourceHome, absPath, relPath string, isDir bool, cfg types.AgentConfig, selected map[string]types.FileEntry) error {
	spec, matched := bestEntry(relPath, cfg)
	if !matched || spec.Strategy == types.StrategyExclude {
		return nil
	}
	if isDir {
		return nil
	}
	return addEntry(absPath, relPath, spec, selected)
}

func addEntry(absPath, relPath string, spec types.WhitelistEntry, selected map[string]types.FileEntry) error {
	if _, ok := selected[relPath]; ok {
		return nil
	}
	entry, err := newEntry(absPath, relPath, spec)
	if err != nil {
		return err
	}
	selected[relPath] = entry
	return nil
}

func addParentDirs(sourceHome string, entries []types.FileEntry) ([]types.FileEntry, error) {
	out := make([]types.FileEntry, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		out = append(out, entry)
		seen[entry.RelativePath] = struct{}{}
		for parent := path.Dir(entry.RelativePath); parent != "." && parent != "/"; parent = path.Dir(parent) {
			if _, ok := seen[parent]; ok {
				continue
			}
			dirPath := filepath.Join(sourceHome, filepath.FromSlash(parent))
			dirEntry, err := newEntry(dirPath, parent, types.WhitelistEntry{Strategy: types.StrategyInclude})
			if err != nil {
				return nil, err
			}
			out = append(out, dirEntry)
			seen[parent] = struct{}{}
		}
	}
	return out, nil
}

func includeByScope(entry types.FileEntry, params types.ManifestParams) bool {
	if len(params.OnlyScopes) == 0 && len(params.ExcludeScopes) == 0 {
		return true
	}
	scopeSet := make(map[string]struct{}, len(entry.Scopes))
	for _, scope := range entry.Scopes {
		scopeSet[scope] = struct{}{}
	}
	if len(params.OnlyScopes) > 0 {
		for _, scope := range params.OnlyScopes {
			if _, ok := scopeSet[scope]; ok {
				goto excludeCheck
			}
		}
		return false
	}
excludeCheck:
	for _, scope := range params.ExcludeScopes {
		if _, ok := scopeSet[scope]; ok {
			return false
		}
	}
	return true
}

func isHistoryJSONL(relPath string) bool {
	relPath = filepath.ToSlash(relPath)
	return strings.HasSuffix(relPath, ".jsonl") && strings.Contains(relPath, "/projects/")
}

func bestEntry(relPath string, cfg types.AgentConfig) (types.WhitelistEntry, bool) {
	best := types.WhitelistEntry{}
	bestLen := -1
	for _, entry := range cfg.Entries {
		if !whitelist.Match(relPath, entry.Path) {
			continue
		}
		if len(entry.Path) < bestLen {
			continue
		}
		best = entry
		bestLen = len(entry.Path)
	}
	return best, bestLen >= 0
}

func newEntry(absPath, relPath string, spec types.WhitelistEntry) (types.FileEntry, error) {
	info, err := os.Lstat(absPath)
	if err != nil {
		return types.FileEntry{}, fmt.Errorf("stat %s: %w", absPath, err)
	}
	entry := types.FileEntry{
		SourcePath:      absPath,
		RelativePath:    relPath,
		Mode:            info.Mode(),
		Size:            info.Size(),
		Strategy:        spec.Strategy,
		IsDir:           info.IsDir(),
		IsSymlink:       info.Mode()&os.ModeSymlink != 0,
		GroupKey:        types.GroupKey(relPath),
		FieldStripRules: append([]types.FieldStripRule(nil), spec.FieldStripRules...),
		Scopes:          append([]string(nil), spec.Scopes...),
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
