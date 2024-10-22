package routes

import (
	"bytes"
	"html/template"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/mleku/legit/git"
)

func (d *deps) Write404(w http.ResponseWriter) {
	var err error
	tpath := filepath.Join(d.c.Dirs.Templates, "*")
	t := template.Must(template.ParseGlob(tpath))
	w.WriteHeader(404)
	if err = t.ExecuteTemplate(w, "404", nil); chk.E(err) {
		log.E.F("404 template: %s", err)
	}
}

func (d *deps) Write500(w http.ResponseWriter) {
	var err error
	tpath := filepath.Join(d.c.Dirs.Templates, "*")
	t := template.Must(template.ParseGlob(tpath))
	w.WriteHeader(500)
	if err = t.ExecuteTemplate(w, "500", nil); chk.E(err) {
		log.E.F("500 template: %s", err)
	}
}

func (d *deps) listFiles(files []git.NiceTree, data map[string]any,
	w http.ResponseWriter) {
	var err error
	tpath := filepath.Join(d.c.Dirs.Templates, "*")
	t := template.Must(template.ParseGlob(tpath))
	data["files"] = files
	data["meta"] = d.c.Meta
	if err = t.ExecuteTemplate(w, "tree", data); chk.E(err) {
		return
	}
}

func countLines(r io.Reader) (int, error) {
	var err error
	var c int
	buf := make([]byte, 32*1024)
	bufLen := 0
	count := 0
	nl := []byte{'\n'}
	for {
		c, err = r.Read(buf)
		if c > 0 {
			bufLen += c
		}
		count += bytes.Count(buf[:c], nl)
		switch {
		case err == io.EOF:
			/* handle last line not having a newline at the end */
			if bufLen >= 1 && buf[(bufLen-1)%(32*1024)] != '\n' {
				count++
			}
			return count, nil
		case chk.E(err):
			return 0, err
		}
	}
}

func (d *deps) showFile(content string, data map[string]any,
	w http.ResponseWriter) {
	var err error
	tpath := filepath.Join(d.c.Dirs.Templates, "*")
	t := template.Must(template.ParseGlob(tpath))
	var lc int
	if lc, err = countLines(strings.NewReader(content)); chk.E(err) {
		// Non-fatal, we'll just skip showing line numbers in the template.
		log.E.F("counting lines: %s", err)
	}
	lines := make([]int, lc)
	if lc > 0 {
		for i := range lines {
			lines[i] = i + 1
		}
	}
	data["linecount"] = lines
	data["content"] = content
	data["meta"] = d.c.Meta
	if err = t.ExecuteTemplate(w, "file", data); err != nil {
		log.E.Ln(err)
		return
	}
}

func (d *deps) showRaw(content string, w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	if _, err := w.Write([]byte(content)); chk.E(err) {
	}
	return
}
