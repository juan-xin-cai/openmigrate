package rewrite

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	omlog "github.com/openmigrate/openmigrate/internal/core/log"
	"github.com/openmigrate/openmigrate/internal/core/types"
)

func RewriteTree(root string, mapping types.PathMapping, scan types.PathScanResult, logger *omlog.Logger) (types.RewriteReport, error) {
	report := types.RewriteReport{
		ExternalPaths: append([]string(nil), scan.ExternalPaths...),
		ProjectRoots:  mapProjectRoots(scan.ProjectRoots, mapping),
	}
	dirs := make([]string, 0, 16)

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
		if info.IsDir() {
			dirs = append(dirs, path)
			return nil
		}
		isText, err := IsTextFile(path)
		if err != nil {
			return err
		}
		if !isText {
			report.SkippedBinary = append(report.SkippedBinary, rel)
			report.Warnings = append(report.Warnings, types.RewriteWarning{Path: rel, Message: "非文本文件，跳过路径重写"})
			if logger != nil {
				logger.Warn("skip non-text file during rewrite", map[string]interface{}{"path": rel})
			}
			return nil
		}
		if err := rewriteFile(path, mapping); err != nil {
			return err
		}
		report.RewrittenFiles++
		return nil
	})
	if err != nil {
		return report, err
	}

	sort.Slice(dirs, func(i, j int) bool { return len(dirs[i]) > len(dirs[j]) })
	for _, dir := range dirs {
		base := filepath.Base(dir)
		newBase := rewriteEncodedName(base, mapping)
		if base == newBase {
			continue
		}
		newPath := filepath.Join(filepath.Dir(dir), newBase)
		if err := os.Rename(dir, newPath); err != nil {
			return report, err
		}
	}

	return report, nil
}

func RewriteLinkRelations(links []types.LinkRelation, mapping types.PathMapping) []types.LinkRelation {
	out := make([]types.LinkRelation, 0, len(links))
	for _, link := range links {
		link.LinkRelativePath = filepath.ToSlash(rewriteEncodedRelativePath(link.LinkRelativePath, mapping))
		link.TargetRelativePath = filepath.ToSlash(rewriteEncodedRelativePath(link.TargetRelativePath, mapping))
		out = append(out, link)
	}
	return out
}

func MapAbsolutePath(value string, mapping types.PathMapping) string {
	replaced := value
	projectMappings := append([]types.PathPair(nil), mapping.ProjectMappings...)
	sort.Slice(projectMappings, func(i, j int) bool { return len(projectMappings[i].From) > len(projectMappings[j].From) })
	for _, pair := range projectMappings {
		replaced = strings.ReplaceAll(replaced, pair.From, pair.To)
	}
	if mapping.SourceHome != "" && mapping.TargetHome != "" {
		replaced = strings.ReplaceAll(replaced, mapping.SourceHome, mapping.TargetHome)
	}
	return replaced
}

func rewriteFile(path string, mapping types.PathMapping) error {
	ext := strings.ToLower(filepath.Ext(path))
	modeInfo, err := os.Stat(path)
	if err != nil {
		return err
	}
	if ext == ".jsonl" {
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		buffer := make([]byte, 0, 64*1024)
		scanner.Buffer(buffer, 2*1024*1024)
		var out bytes.Buffer
		for scanner.Scan() {
			line := MapAbsolutePath(scanner.Text(), mapping)
			out.WriteString(line)
			out.WriteByte('\n')
		}
		if err := scanner.Err(); err != nil {
			return err
		}
		return ioutil.WriteFile(path, out.Bytes(), modeInfo.Mode())
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	rewritten := MapAbsolutePath(string(data), mapping)
	return ioutil.WriteFile(path, []byte(rewritten), modeInfo.Mode())
}

func mapProjectRoots(roots []string, mapping types.PathMapping) []string {
	out := make([]string, 0, len(roots))
	for _, root := range roots {
		out = append(out, MapAbsolutePath(root, mapping))
	}
	return out
}

func rewriteEncodedRelativePath(relPath string, mapping types.PathMapping) string {
	parts := strings.Split(filepath.ToSlash(relPath), "/")
	for i, part := range parts {
		parts[i] = rewriteEncodedName(part, mapping)
	}
	return strings.Join(parts, "/")
}

func rewriteEncodedName(name string, mapping types.PathMapping) string {
	replaced := name
	projectMappings := append([]types.PathPair(nil), mapping.ProjectMappings...)
	sort.Slice(projectMappings, func(i, j int) bool { return len(projectMappings[i].From) > len(projectMappings[j].From) })
	for _, pair := range projectMappings {
		replaced = strings.ReplaceAll(replaced, encodePath(pair.From), encodePath(pair.To))
	}
	if mapping.SourceHome != "" && mapping.TargetHome != "" {
		replaced = strings.ReplaceAll(replaced, encodePath(mapping.SourceHome), encodePath(mapping.TargetHome))
	}
	return replaced
}

func encodePath(pathValue string) string {
	return strings.ReplaceAll(pathValue, "/", "-")
}
