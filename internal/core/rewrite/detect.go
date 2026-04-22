package rewrite

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gabriel-vasile/mimetype"
)

var textExtensions = map[string]bool{
	".json":  true,
	".jsonl": true,
	".md":    true,
	".txt":   true,
	".yaml":  true,
	".yml":   true,
	".toml":  true,
	".sh":    true,
	".zsh":   true,
	".fish":  true,
	".env":   true,
}

var binaryExtensions = map[string]bool{
	".db":      true,
	".sqlite":  true,
	".sqlite3": true,
	".zip":     true,
	".gz":      true,
	".jpg":     true,
	".jpeg":    true,
	".png":     true,
	".pdf":     true,
	".bin":     true,
	".dylib":   true,
	".so":      true,
}

func IsTextFile(path string) (bool, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if textExtensions[ext] {
		return true, nil
	}
	if binaryExtensions[ext] {
		return false, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()
	mtype, err := mimetype.DetectReader(file)
	if err != nil {
		return false, err
	}
	if strings.HasPrefix(mtype.String(), "text/") {
		return true, nil
	}
	return strings.Contains(mtype.String(), "json") || strings.Contains(mtype.String(), "xml"), nil
}
