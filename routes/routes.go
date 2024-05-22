package routes

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday/v2"
	"mleku.net/legit/config"
	"mleku.net/legit/git"
)

type deps struct {
	c *config.Config
}

type info struct {
	Name, Desc, Idle string
	d                time.Time
}

func (d *deps) Index(w http.ResponseWriter, r *http.Request) {
	var err error
	var dirs []os.DirEntry
	if dirs, err = os.ReadDir(d.c.Repo.ScanPath); chk.E(err) {
		d.Write500(w)
		log.E.F("reading scan path: %s", err)
		return
	}
	var infos []info
	for _, dir := range dirs {
		if d.isIgnored(dir.Name()) {
			continue
		}
		path := filepath.Join(d.c.Repo.ScanPath, dir.Name())
		var gr *git.GitRepo
		if gr, err = git.Open(path, ""); chk.E(err) {
			continue
		}
		var c *object.Commit
		if c, err = gr.LastCommit(); chk.E(err) {
			d.Write500(w)
			return
		}
		desc := getDescription(path)
		infos = append(infos, info{
			Name: dir.Name(),
			Desc: desc,
			Idle: humanize.Time(c.Author.When),
			d:    c.Author.When,
		})
	}
	sort.Slice(infos, func(i, j int) bool {
		return infos[j].d.Before(infos[i].d)
	})
	tpath := filepath.Join(d.c.Dirs.Templates, "*")
	t := template.Must(template.ParseGlob(tpath))
	data := map[string]any{
		"meta": d.c.Meta,
		"info": infos,
	}
	if err = t.ExecuteTemplate(w, "index", data); chk.E(err) {
		return
	}
}

func (d *deps) RepoIndex(w http.ResponseWriter, r *http.Request) {
	var err error
	name := r.PathValue("name")
	if d.isIgnored(name) {
		d.Write404(w)
		return
	}
	name = filepath.Clean(name)
	path := filepath.Join(d.c.Repo.ScanPath, name)
	var gr *git.GitRepo
	if gr, err = git.Open(path, ""); chk.E(err) {
		d.Write404(w)
		return
	}
	var commits []*object.Commit
	if commits, err = gr.Commits(); chk.E(err) {
		d.Write500(w)
		return
	}
	var readmeContent template.HTML
	for _, readme := range d.c.Repo.Readme {
		ext := filepath.Ext(readme)
		var content string
		content, err = gr.FileContent(readme)
		if len(content) > 0 {
			switch ext {
			case ".md", ".mkd", ".markdown":
				unsafe := blackfriday.Run(
					[]byte(content),
					blackfriday.WithExtensions(blackfriday.CommonExtensions),
				)
				html := bluemonday.UGCPolicy().SanitizeBytes(unsafe)
				readmeContent = template.HTML(html)
			default:
				readmeContent = template.HTML(
					fmt.Sprintf(`<pre>%s</pre>`, content),
				)
			}
			break
		}
	}
	if readmeContent == "" {
		log.E.Ln("no readme found for %s", name)
	}
	var mainBranch string
	if mainBranch, err = gr.FindMainBranch(d.c.Repo.MainBranch); chk.E(err) {
		d.Write500(w)
		return
	}
	tpath := filepath.Join(d.c.Dirs.Templates, "*")
	t := template.Must(template.ParseGlob(tpath))
	if len(commits) >= 3 {
		commits = commits[:3]
	}
	data := map[string]any{
		"name":       name,
		"ref":        mainBranch,
		"readme":     readmeContent,
		"commits":    commits,
		"desc":       getDescription(path),
		"servername": d.c.Server.Name,
		"meta":       d.c.Meta,
		"gomod":      isGoModule(gr),
	}
	if err = t.ExecuteTemplate(w, "repo", data); chk.E(err) {
		return
	}
	return
}

func (d *deps) RepoTree(w http.ResponseWriter, r *http.Request) {
	var err error
	name := r.PathValue("name")
	if d.isIgnored(name) {
		log.E.Ln(name, "is ignored")
		d.Write404(w)
		return
	}
	treePath := r.PathValue("rest")
	ref := r.PathValue("ref")
	name = filepath.Clean(name)
	path := filepath.Join(d.c.Repo.ScanPath, name)
	var gr *git.GitRepo
	if gr, err = git.Open(path, ref); chk.E(err) {
		d.Write404(w)
		return
	}
	var files []git.NiceTree
	if files, err = gr.FileTree(treePath); chk.E(err) {
		d.Write500(w)
		return
	}
	data := map[string]any{
		"name":   name,
		"ref":    ref,
		"parent": treePath,
		"desc":   getDescription(path),
		"dotdot": filepath.Dir(treePath),
	}
	d.listFiles(files, data, w)
	return
}

func (d *deps) FileContent(w http.ResponseWriter, r *http.Request) {
	var err error
	var raw bool
	if rawParam, err := strconv.ParseBool(r.URL.Query().Get("raw")); err == nil {
		raw = rawParam
	}
	name := r.PathValue("name")
	if d.isIgnored(name) {
		d.Write404(w)
		return
	}
	treePath := r.PathValue("rest")
	ref := r.PathValue("ref")
	name = filepath.Clean(name)
	path := filepath.Join(d.c.Repo.ScanPath, name)
	gr, err := git.Open(path, ref)
	if err != nil {
		d.Write404(w)
		return
	}
	var contents string
	if contents, err = gr.FileContent(treePath); chk.E(err) {
	}
	data := map[string]any{
		"name": name,
		"ref":  ref,
		"desc": getDescription(path),
		"path": treePath,
	}
	if raw {
		d.showRaw(contents, w)
	} else {
		d.showFile(contents, data, w)
	}
	return
}

func (d *deps) Log(w http.ResponseWriter, r *http.Request) {
	var err error
	name := r.PathValue("name")
	if d.isIgnored(name) {
		log.E.Ln(name, "is ignored")
		d.Write404(w)
		return
	}
	ref := r.PathValue("ref")
	path := filepath.Join(d.c.Repo.ScanPath, name)
	var gr *git.GitRepo
	if gr, err = git.Open(path, ref); chk.E(err) {
		d.Write404(w)
		return
	}
	var commits []*object.Commit
	if commits, err = gr.Commits(); chk.E(err) {
		d.Write500(w)
		return
	}
	tpath := filepath.Join(d.c.Dirs.Templates, "*")
	t := template.Must(template.ParseGlob(tpath))
	data := map[string]any{
		"commits": commits,
		"meta":    d.c.Meta,
		"name":    name,
		"ref":     ref,
		"desc":    getDescription(path),
		"log":     true,
	}
	if err = t.ExecuteTemplate(w, "log", data); chk.E(err) {
		return
	}
}

func (d *deps) Diff(w http.ResponseWriter, r *http.Request) {
	var err error
	name := r.PathValue("name")
	if d.isIgnored(name) {
		d.Write404(w)
		return
	}
	ref := r.PathValue("ref")
	path := filepath.Join(d.c.Repo.ScanPath, name)
	var gr *git.GitRepo
	if gr, err = git.Open(path, ref); chk.E(err) {
		d.Write404(w)
		return
	}
	var dif *git.NiceDiff
	if dif, err = gr.Diff(); chk.E(err) {
		d.Write500(w)
		return
	}
	tpath := filepath.Join(d.c.Dirs.Templates, "*")
	t := template.Must(template.ParseGlob(tpath))
	data := map[string]any{
		"commit": dif.Commit,
		"stat":   dif.Stat,
		"diff":   dif.Diff,
		"meta":   d.c.Meta,
		"name":   name,
		"ref":    ref,
		"desc":   getDescription(path),
	}
	if err = t.ExecuteTemplate(w, "commit", data); chk.E(err) {
		return
	}
}

func (d *deps) Refs(w http.ResponseWriter, r *http.Request) {
	var err error
	name := r.PathValue("name")
	if d.isIgnored(name) {
		log.E.Ln(name, "is ignored")
		d.Write404(w)
		return
	}
	path := filepath.Join(d.c.Repo.ScanPath, name)
	var gr *git.GitRepo
	if gr, err = git.Open(path, ""); chk.E(err) {
		d.Write404(w)
		return
	}
	var tags []*object.Tag
	if tags, err = gr.Tags(); chk.E(err) {
		// Non-fatal, we *should* have at least one branch to show.
	}
	var branches []*plumbing.Reference
	if branches, err = gr.Branches(); chk.E(err) {
		d.Write500(w)
		return
	}
	tpath := filepath.Join(d.c.Dirs.Templates, "*")
	t := template.Must(template.ParseGlob(tpath))
	data := map[string]any{
		"meta":     d.c.Meta,
		"name":     name,
		"branches": branches,
		"tags":     tags,
		"desc":     getDescription(path),
	}
	if err = t.ExecuteTemplate(w, "refs", data); chk.E(err) {
		return
	}
}

func (d *deps) ServeStatic(w http.ResponseWriter, r *http.Request) {
	f := r.PathValue("file")
	f = filepath.Clean(filepath.Join(d.c.Dirs.Static, f))
	http.ServeFile(w, r, f)
}
