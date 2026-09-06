// Package packs embeds the rule packs and writes the matching subset for a
// detected stack into <project>/.devbase/rules/.
package packs

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

//go:generate sh -c "rm -rf data && cp -R ../../packs data"
//go:embed all:data
var files embed.FS

// Write copies every file under packs/<stack>/ for each stack into
// dir/.devbase/rules/<stack>/. Missing pack dirs are skipped silently.
// Returns the written destination files.
func Write(dir string, stacks []string) ([]string, error) {
	var written []string
	for _, stack := range stacks {
		src := path.Join("data", stack)
		err := fs.WalkDir(files, src, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				if strings.Contains(err.Error(), "not found") {
					return fs.SkipDir
				}
				return err
			}
			if d.IsDir() {
				return nil
			}
			data, err := files.ReadFile(p)
			if err != nil {
				return err
			}
			rel := strings.TrimPrefix(p, "data/")
			dst := filepath.Join(dir, ".devbase", "rules", filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(dst, data, 0o644); err != nil {
				return err
			}
			written = append(written, dst)
			return nil
		})
		if err != nil {
			return written, fmt.Errorf("pack %s: %w", stack, err)
		}
	}
	return written, nil
}
