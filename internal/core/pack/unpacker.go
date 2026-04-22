package pack

import (
	"os"
	"path/filepath"

	"github.com/openmigrate/openmigrate/internal/core/types"
)

func UnpackPackage(packagePath, passphrase string) (string, types.PackageMeta, error) {
	workDir, err := os.MkdirTemp("", "openmigrate-unpack-*")
	if err != nil {
		return "", types.PackageMeta{}, err
	}
	decryptedPath := filepath.Join(workDir, "package.tar.zst")
	src, err := os.Open(packagePath)
	if err != nil {
		return "", types.PackageMeta{}, err
	}
	defer src.Close()
	dst, err := os.Create(decryptedPath)
	if err != nil {
		return "", types.PackageMeta{}, err
	}
	if err := Decrypt(src, dst, passphrase); err != nil {
		_ = dst.Close()
		return "", types.PackageMeta{}, err
	}
	_ = dst.Close()
	root := filepath.Join(workDir, "payload")
	file, err := os.Open(decryptedPath)
	if err != nil {
		return "", types.PackageMeta{}, err
	}
	defer file.Close()
	if err := DecompressArchive(file, root); err != nil {
		return "", types.PackageMeta{}, err
	}
	meta, err := ReadMeta(filepath.Join(root, ".openmigrate", "package-meta.json"))
	if err != nil {
		return "", types.PackageMeta{}, err
	}
	return root, meta, nil
}
