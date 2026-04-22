package pack

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/openmigrate/openmigrate/internal/core/types"
)

func CreatePackage(manifest types.Manifest, meta types.PackageMeta, outputPath, metaPath, passphrase string) error {
	stageDir, err := os.MkdirTemp("", "openmigrate-pack-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stageDir)

	for _, entry := range manifest.Entries {
		dest := filepath.Join(stageDir, filepath.FromSlash(entry.RelativePath))
		if entry.IsDir {
			if err := os.MkdirAll(dest, entry.Mode); err != nil {
				return err
			}
			continue
		}
		if err := copyEntry(entry, dest); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Join(stageDir, ".openmigrate"), 0o755); err != nil {
		return err
	}
	if err := WriteMeta(filepath.Join(stageDir, ".openmigrate", "package-meta.json"), meta); err != nil {
		return err
	}

	compressedPath := outputPath + ".tmp.zst"
	defer os.Remove(compressedPath)
	compressedFile, err := os.Create(compressedPath)
	if err != nil {
		return err
	}
	if err := CompressDirectory(stageDir, compressedFile); err != nil {
		_ = compressedFile.Close()
		return err
	}
	_ = compressedFile.Close()

	src, err := os.Open(compressedPath)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer dst.Close()
	if err := Encrypt(src, dst, passphrase); err != nil {
		return err
	}
	return WriteMeta(metaPath, meta)
}

func copyEntry(entry types.FileEntry, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if entry.Strategy == types.StrategyFieldStrip {
		return copyStrippedJSON(entry, target)
	}
	src, err := os.Open(entry.SourcePath)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, entry.Mode)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}

func copyStrippedJSON(entry types.FileEntry, target string) error {
	data, err := ioutil.ReadFile(entry.SourcePath)
	if err != nil {
		return err
	}
	var payload interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return ioutil.WriteFile(target, data, entry.Mode)
	}
	payload = stripSensitiveFields(payload)
	clean, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(target, clean, entry.Mode)
}

func stripSensitiveFields(node interface{}) interface{} {
	switch typed := node.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(typed))
		for key, value := range typed {
			if shouldStripField(key) {
				continue
			}
			out[key] = stripSensitiveFields(value)
		}
		return out
	case []interface{}:
		out := make([]interface{}, 0, len(typed))
		for _, value := range typed {
			out = append(out, stripSensitiveFields(value))
		}
		return out
	default:
		return node
	}
}

func shouldStripField(key string) bool {
	lower := strings.ToLower(key)
	return lower == "secrets" || strings.HasPrefix(lower, "oauth:") || strings.HasPrefix(lower, "token") || lower == "device_id_salt"
}
