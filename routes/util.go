package routes

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"mleku.dev/git/slog"
	"mleku.net/legit/git"
)

var log, chk = slog.New(os.Stderr)

func isGoModule(gr *git.GitRepo) bool {
	_, err := gr.FileContent("go.mod")
	return err == nil
}

func getDescription(path string) (desc string) {
	descFile := filepath.Join(path, "description")
	db, err := os.ReadFile(descFile)
	if err == nil {
		desc = string(db)
	} else {
		desc = ""
	}
	return
}

func (d *deps) isIgnored(name string) bool {
	for _, i := range d.c.Repo.Ignore {
		if name == i {
			return true
		}
	}
	return false
}

type repoInfo struct {
	Git      *git.GitRepo
	Path     string
	Category string
}

func (d *deps) getAllRepos() (repos []repoInfo, err error) {
	maximum := strings.Count(d.c.Repo.ScanPath, string(os.PathSeparator)) + 2
	err = filepath.WalkDir(d.c.Repo.ScanPath, func(path string, de fs.DirEntry, e error) (err error) {
		if chk.E(e) {
			return
		}
		if de.IsDir() {
			// Check if we've exceeded our recursion depth
			if strings.Count(path, string(os.PathSeparator)) > maximum {
				return fs.SkipDir
			}
			if d.isIgnored(path) {
				log.I.Ln(path, "is ignored")
				return fs.SkipDir
			}
			// A bare repo should always have at least a HEAD file, if it
			// doesn't we can continue recursing
			if _, err = os.Lstat(filepath.Join(path, "HEAD")); !chk.E(err) {
				var gr *git.GitRepo
				if gr, err = git.Open(path, ""); !chk.E(err) {
					relpath, _ := filepath.Rel(d.c.Repo.ScanPath, path)
					repos = append(repos, repoInfo{
						Git:      gr,
						Path:     relpath,
						Category: d.category(path),
					})
					// Since we found a Git repo, we don't want to recurse
					// further
					return fs.SkipDir
				}
			}
		}
		return
	})
	return
}

func (d *deps) category(path string) string {
	return strings.TrimPrefix(filepath.Dir(strings.TrimPrefix(path,
		d.c.Repo.ScanPath)), string(os.PathSeparator))
}
