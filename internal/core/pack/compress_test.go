package pack

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestCompressDecompressRoundTrip(t *testing.T) {
	source := t.TempDir()
	if err := os.MkdirAll(filepath.Join(source, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := ioutil.WriteFile(filepath.Join(source, "nested", "a.txt"), []byte("alpha"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	var archive bytes.Buffer
	if err := CompressDirectory(source, &archive); err != nil {
		t.Fatalf("compress: %v", err)
	}

	dest := t.TempDir()
	if err := DecompressArchive(bytes.NewReader(archive.Bytes()), dest); err != nil {
		t.Fatalf("decompress: %v", err)
	}
	data, err := ioutil.ReadFile(filepath.Join(dest, "nested", "a.txt"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) != "alpha" {
		t.Fatalf("unexpected content: %q", string(data))
	}
}
