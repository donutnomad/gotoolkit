package format

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/swaggo/swag"
)

func Format(path string) error {
	formatter := swag.NewFormatter()
	original, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	contents := make([]byte, len(original))
	copy(contents, original)
	formatted, err := formatter.Format(path, contents)
	if err != nil {
		return err
	}
	if bytes.Equal(original, formatted) {
		// Skip write if no change
		return nil
	}
	return write(path, formatted)
}

func write(path string, contents []byte) error {
	originalFileInfo, err := os.Stat(path)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path))
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(contents); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Chmod(f.Name(), originalFileInfo.Mode()); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
