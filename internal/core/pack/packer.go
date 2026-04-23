package pack

import (
	"io"
	"os"
	"path/filepath"

	"github.com/openmigrate/openmigrate/internal/core/fieldstrip"
	omlog "github.com/openmigrate/openmigrate/internal/core/log"
	"github.com/openmigrate/openmigrate/internal/core/types"
)

func CreatePackage(manifest types.Manifest, meta types.PackageMeta, outputPath, metaPath, passphrase string, logger *omlog.Logger) error {
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
		if err := copyEntry(entry, dest, logger); err != nil {
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

func copyEntry(entry types.FileEntry, target string, logger *omlog.Logger) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if len(entry.FieldStripRules) > 0 {
		return copyStrippedJSON(entry, target, logger)
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

func copyStrippedJSON(entry types.FileEntry, target string, logger *omlog.Logger) error {
	data, err := os.ReadFile(entry.SourcePath)
	if err != nil {
		return err
	}
	clean, err := fieldstrip.Strip(data, entry.FieldStripRules)
	if err != nil {
		if err == types.ErrNotJSON {
			if logger != nil {
				logger.Warn("field strip skipped for non-json entry", map[string]interface{}{"path": entry.RelativePath})
			}
			return os.WriteFile(target, data, entry.Mode)
		}
		return err
	}
	return os.WriteFile(target, clean, entry.Mode)
}
