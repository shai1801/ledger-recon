package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WriteRecords atomically writes records to <dir>/<filename>.<ext>, where
// ext is derived from format. It writes to a temp file in the SAME
// directory first, then renames into place -- so a crash mid-write never
// leaves a partially-written or corrupt output file. A prior successful
// output file remains fully intact right up until the rename succeeds.
func WriteRecords[T any](dir, filename, format string, records []T) error {

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating output directory %q: %w", dir, err)
	}

	ext, err := extensionFor(format)
	if err != nil {
		return err
	}

	targetPath := filepath.Join(dir, filename+ext)

	tmp, err := os.CreateTemp(dir, "."+filename+"-*.tmp")
	if err != nil {
		return fmt.Errorf("Creating temp file for %s:%w", filename, err)
	}

	tmpPath := tmp.Name()

	// If anything below fails before the rename, clean up the temp file
	// so it doesn't linger in the output directory.

	defer os.Remove(tmpPath)

	if err := writeFormatted(tmp, format, records); err != nil {
		tmp.Close()
		return fmt.Errorf("writing %s: %w", filename, err)
	}

	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("syncing %s: %w", filename, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file for %s: %w", filename, err)
	}

	if err := os.Rename(tmpPath, targetPath); err != nil {
		return fmt.Errorf("renaming into place for %s: %w", filename, err)
	}

	return nil

}

func extensionFor(format string) (string, error) {
	switch format {
	case "jsonl":
		return ".jsonl", nil
	case "json":
		return ".json", nil
	default:
		return "", fmt.Errorf("unsupported output_format %q (supported: json, jsonl)", format)
	}
}

func writeFormatted[T any](f *os.File, format string, records []T) error {
	switch format {
	case "jsonl":
		enc := json.NewEncoder(f)
		for _, r := range records {
			if err := enc.Encode(r); err != nil {
				return err
			}
		}
		return nil
	case "json":
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		return enc.Encode(records)
	default:
		return fmt.Errorf("unsupported output_format %q", format)
	}
}
