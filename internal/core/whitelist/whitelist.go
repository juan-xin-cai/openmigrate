package whitelist

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/openmigrate/openmigrate/internal/core/types"
)

//go:embed claude-code-v2.json
var embeddedFiles embed.FS

func Load(agent, version string) (types.AgentConfig, error) {
	name := strings.ToLower(agent) + "-" + strings.ToLower(version) + ".json"
	for _, dir := range searchDirs() {
		if dir == "" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err == nil {
			return decodeConfig(name, data)
		}
		if err != nil && !os.IsNotExist(err) {
			return types.AgentConfig{}, fmt.Errorf("load whitelist %s from %s: %w", name, dir, err)
		}
	}
	data, err := embeddedFiles.ReadFile(name)
	if err != nil {
		return types.AgentConfig{}, fmt.Errorf("load whitelist %s: %w", name, err)
	}
	return decodeConfig(name, data)
}

func Match(relPath, pattern string) bool {
	relPath = filepath.ToSlash(strings.TrimPrefix(relPath, "/"))
	pattern = filepath.ToSlash(strings.TrimPrefix(pattern, "/"))
	if strings.HasSuffix(pattern, "/**") {
		base := strings.TrimSuffix(pattern, "/**")
		return relPath == base || strings.HasPrefix(relPath, base+"/")
	}
	matched, _ := filepath.Match(pattern, relPath)
	return matched
}

func decodeConfig(name string, data []byte) (types.AgentConfig, error) {
	var cfg types.AgentConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return types.AgentConfig{}, fmt.Errorf("decode whitelist %s: %w", name, err)
	}
	return cfg, nil
}

func searchDirs() []string {
	dirs := make([]string, 0, 5)
	if envDir := os.Getenv("OPENMIGRATE_WHITELIST_DIR"); envDir != "" {
		dirs = append(dirs, envDir)
	}
	if cwd, err := os.Getwd(); err == nil {
		dirs = append(dirs, filepath.Join(cwd, "internal", "core", "whitelist"))
		dirs = append(dirs, filepath.Join(cwd, "whitelist"))
	}
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		dirs = append(dirs, filepath.Join(exeDir, "whitelist"))
		dirs = append(dirs, filepath.Join(exeDir, "..", "share", "openmigrate", "whitelist"))
	}
	return dirs
}
