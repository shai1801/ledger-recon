package source

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Package abstracts reading named files from either a plain directory or
// a .zip archive, so the rest of the program doesn't need to know which
// one --input pointed at.
type Package struct {
	dir       string
	zipReader *zip.Reader
	zipFile   *os.File
}

// Open inspects path and returns a Package backed by a directory or a
// zip archive, whichever path actually is.
func Open(path string) (*Package, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("opening input %q: %w", path, err)
	}

	if info.IsDir() {
		return &Package{dir: path}, nil
	}

	if !strings.EqualFold(filepath.Ext(path), ".zip") {
		return nil, fmt.Errorf("input %q is neither a directory nor a .zip file", path)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening zip %q: %w", path, err)
	}
	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("stat zip %q: %w", path, err)
	}
	zr, err := zip.NewReader(f, stat.Size())
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("reading zip %q: %w", path, err)
	}

	return &Package{zipReader: zr, zipFile: f}, nil
}

// Reader opens the named file, wherever it lives inside this package.
func (p *Package) Reader(name string) (io.ReadCloser, error) {
	if p.zipReader == nil {
		f, err := os.Open(filepath.Join(p.dir, name))
		if err != nil {
			return nil, fmt.Errorf("opening %q: %w", name, err)
		}
		return f, nil
	}

	for _, f := range p.zipReader.File {
		if f.Name == name || strings.HasSuffix(f.Name, "/"+name) {
			return f.Open()
		}
	}
	return nil, fmt.Errorf("file %q not found in zip", name)
}

// Bytes reads the named file fully, returning (nil, false) if it doesn't
// exist rather than an error -- used for optional files like
// configuration.json, which may legitimately be absent.
func (p *Package) Bytes(name string) ([]byte, bool) {
	r, err := p.Reader(name)
	if err != nil {
		return nil, false
	}
	defer r.Close()
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, false
	}
	return data, true
}

// Close releases the underlying zip file handle, if any. A no-op for
// directory-backed packages.
func (p *Package) Close() error {
	if p.zipFile != nil {
		return p.zipFile.Close()
	}
	return nil
}
