package snapshot

import (
	"archive/tar"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	omlog "github.com/openmigrate/openmigrate/internal/core/log"
	"github.com/openmigrate/openmigrate/internal/core/pack"
	"github.com/openmigrate/openmigrate/internal/core/types"
)

type indexEntry struct {
	Target string `json:"target"`
	Rel    string `json:"rel"`
	Exists bool   `json:"exists"`
}

func CreateSnapshot(targetPaths []string, passphrase string, logger *omlog.Logger) (types.SnapshotMeta, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return types.SnapshotMeta{}, err
	}
	if err := ensureDisk(targetPaths, home); err != nil {
		return types.SnapshotMeta{}, err
	}
	id := time.Now().Format("20060102-150405")
	snapshotDir := filepath.Join(home, "Library", "Application Support", "OpenMigrate", "snapshots", id)
	if err := os.MkdirAll(snapshotDir, 0o755); err != nil {
		return types.SnapshotMeta{}, err
	}
	stageDir, err := os.MkdirTemp("", "openmigrate-snapshot-*")
	if err != nil {
		return types.SnapshotMeta{}, err
	}
	defer os.RemoveAll(stageDir)

	index := make([]indexEntry, 0, len(targetPaths))
	for _, target := range targetPaths {
		rel := filepath.ToSlash(strings.TrimPrefix(filepath.Clean(target), "/"))
		entry := indexEntry{Target: target, Rel: rel}
		if _, err := os.Lstat(target); err == nil {
			entry.Exists = true
			if err := copyPath(target, filepath.Join(stageDir, "payload", filepath.FromSlash(rel))); err != nil {
				return types.SnapshotMeta{}, err
			}
		}
		index = append(index, entry)
	}
	indexBytes, err := json.Marshal(index)
	if err != nil {
		return types.SnapshotMeta{}, err
	}
	if err := os.MkdirAll(filepath.Join(stageDir, ".openmigrate"), 0o755); err != nil {
		return types.SnapshotMeta{}, err
	}
	if err := ioutil.WriteFile(filepath.Join(stageDir, ".openmigrate", "snapshot-index.json"), indexBytes, 0o644); err != nil {
		return types.SnapshotMeta{}, err
	}

	tarPath := filepath.Join(snapshotDir, "snapshot.tar")
	if err := tarDirectory(stageDir, tarPath); err != nil {
		return types.SnapshotMeta{}, err
	}
	defer os.Remove(tarPath)

	src, err := os.Open(tarPath)
	if err != nil {
		return types.SnapshotMeta{}, err
	}
	defer src.Close()
	dstPath := filepath.Join(snapshotDir, "snapshot.age")
	dst, err := os.Create(dstPath)
	if err != nil {
		return types.SnapshotMeta{}, err
	}
	defer dst.Close()
	if err := pack.Encrypt(src, dst, passphrase); err != nil {
		return types.SnapshotMeta{}, err
	}
	if logger != nil {
		logger.Info("snapshot created", map[string]interface{}{"archive": dstPath, "targets": len(targetPaths)})
	}
	return types.SnapshotMeta{
		ID:          id,
		CreatedAt:   time.Now(),
		ArchivePath: dstPath,
		Targets:     targetPaths,
	}, nil
}

func Rollback(meta types.SnapshotMeta, passphrase string, logger *omlog.Logger) error {
	workDir, err := os.MkdirTemp("", "openmigrate-rollback-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workDir)

	decryptedPath := filepath.Join(workDir, "snapshot.tar")
	src, err := os.Open(meta.ArchivePath)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.Create(decryptedPath)
	if err != nil {
		return err
	}
	if err := pack.Decrypt(src, dst, passphrase); err != nil {
		_ = dst.Close()
		return err
	}
	_ = dst.Close()
	restoreDir := filepath.Join(workDir, "restore")
	if err := untarDirectory(decryptedPath, restoreDir); err != nil {
		return err
	}
	indexBytes, err := ioutil.ReadFile(filepath.Join(restoreDir, ".openmigrate", "snapshot-index.json"))
	if err != nil {
		return err
	}
	var index []indexEntry
	if err := json.Unmarshal(indexBytes, &index); err != nil {
		return err
	}

	for _, item := range index {
		target := item.Target
		if !item.Exists {
			_ = os.RemoveAll(target)
			continue
		}
		source := filepath.Join(restoreDir, "payload", filepath.FromSlash(item.Rel))
		backup := target + ".rollback-bak"
		_ = os.RemoveAll(backup)
		if _, err := os.Lstat(target); err == nil {
			if err := os.Rename(target, backup); err != nil {
				return err
			}
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.Rename(source, target); err != nil {
			if _, statErr := os.Lstat(backup); statErr == nil {
				_ = os.Rename(backup, target)
			}
			return err
		}
		_ = os.RemoveAll(backup)
	}

	if logger != nil {
		logger.Info("rollback finished", map[string]interface{}{"archive": meta.ArchivePath})
	}
	return nil
}

func ResolveSnapshot(snapshotID string) (types.SnapshotMeta, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return types.SnapshotMeta{}, err
	}
	root := filepath.Join(home, "Library", "Application Support", "OpenMigrate", "snapshots")
	if snapshotID == "" || snapshotID == "latest" {
		entries, err := ioutil.ReadDir(root)
		if err != nil {
			if os.IsNotExist(err) {
				return types.SnapshotMeta{}, types.ErrSnapshotNotFound
			}
			return types.SnapshotMeta{}, err
		}
		for i := len(entries) - 1; i >= 0; i-- {
			if entries[i].IsDir() {
				snapshotID = entries[i].Name()
				break
			}
		}
	}
	if snapshotID == "" {
		return types.SnapshotMeta{}, types.ErrSnapshotNotFound
	}
	archivePath := filepath.Join(root, snapshotID, "snapshot.age")
	info, err := os.Stat(archivePath)
	if err != nil {
		if os.IsNotExist(err) {
			return types.SnapshotMeta{}, types.ErrSnapshotNotFound
		}
		return types.SnapshotMeta{}, err
	}
	return types.SnapshotMeta{
		ID:          snapshotID,
		CreatedAt:   info.ModTime(),
		ArchivePath: archivePath,
	}, nil
}

func ensureDisk(paths []string, home string) error {
	var total int64
	for _, item := range paths {
		size, err := pathSize(item)
		if err != nil {
			continue
		}
		total += size
	}
	var stat syscall.Statfs_t
	if err := syscall.Statfs(home, &stat); err != nil {
		return err
	}
	free := int64(stat.Bavail) * int64(stat.Bsize)
	if free < total+10*1024*1024 {
		return types.ErrDiskFull
	}
	return nil
}

func pathSize(path string) (int64, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return 0, err
	}
	if !info.IsDir() {
		return info.Size(), nil
	}
	var total int64
	err = filepath.Walk(path, func(current string, currentInfo os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !currentInfo.IsDir() {
			total += currentInfo.Size()
		}
		return nil
	})
	return total, err
}

func copyPath(source, target string) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return copyFile(source, target, info.Mode())
	}
	return filepath.Walk(source, func(current string, currentInfo os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, current)
		if err != nil {
			return err
		}
		dest := target
		if rel != "." {
			dest = filepath.Join(target, rel)
		}
		if currentInfo.IsDir() {
			return os.MkdirAll(dest, currentInfo.Mode())
		}
		return copyFile(current, dest, currentInfo.Mode())
	})
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

func tarDirectory(sourceDir, targetTar string) error {
	file, err := os.Create(targetTar)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := tar.NewWriter(file)
	defer writer.Close()

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(rel)
		if err := writer.WriteHeader(header); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()
		_, err = io.Copy(writer, src)
		return err
	})
}

func untarDirectory(tarPath, dest string) error {
	file, err := os.Open(tarPath)
	if err != nil {
		return err
	}
	defer file.Close()
	reader := tar.NewReader(file)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		target := filepath.Join(dest, filepath.FromSlash(header.Name))
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}
		default:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(file, reader); err != nil {
				_ = file.Close()
				return err
			}
			_ = file.Close()
		}
	}
}
