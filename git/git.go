package git

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/mleku/lol"
)

var log, chk = lol.New(os.Stderr)

type GitRepo struct {
	r *git.Repository
	h plumbing.Hash
}

type TagList []*object.Tag

func (t TagList) Len() int           { return len(t) }
func (t TagList) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
func (t TagList) Less(i, j int) bool { return t[j].Tagger.When.After(t[i].Tagger.When) }

func Open(path string, ref string) (gr *GitRepo, err error) {
	gr = &GitRepo{}
	gr.r, err = git.PlainOpen(path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	if ref == "" {
		var head *plumbing.Reference
		if head, err = gr.r.Head(); chk.E(err) {
			err = log.E.Err("getting head of %s: %w", path, err)
			return
		}
		gr.h = head.Hash()
	} else {
		var h *plumbing.Hash
		if h, err = gr.r.ResolveRevision(plumbing.Revision(ref)); chk.E(err) {
			err = log.E.Err("resolving rev %s for %s: %w", ref, path, err)
			return
		}
		gr.h = *h
	}
	return
}

func (g *GitRepo) Commits() (oc []*object.Commit, err error) {
	var ci object.CommitIter
	if ci, err = g.r.Log(&git.LogOptions{From: g.h}); chk.E(err) {
		err = log.E.Err("commits from ref: %w", err)
		return
	}
	oc = []*object.Commit{}
	chk.E(ci.ForEach(func(c *object.Commit) (err error) {
		oc = append(oc, c)
		return
	}))
	return
}

func (g *GitRepo) LastCommit() (oc *object.Commit, err error) {
	return g.r.CommitObject(g.h)
}

func (g *GitRepo) FileContent(path string) (content string, err error) {
	var c *object.Commit
	if c, err = g.r.CommitObject(g.h); chk.E(err) {
		err = log.E.Err("commit object: %w", err)
		return
	}
	var tree *object.Tree
	if tree, err = c.Tree(); chk.E(err) {
		err = log.E.Err("file tree: %w", err)
		return
	}
	var file *object.File
	if file, err = tree.File(path); chk.E(err) {
		err = log.E.Err("%s: %s", err, path)
		return
	}
	isBinary, _ := file.IsBinary()
	if !isBinary {
		content, err = file.Contents()
	} else {
		content = "Not displaying binary file"
	}
	return
}

func (g *GitRepo) Tags() (tags []*object.Tag, err error) {
	log.I.Ln("tags")
	var ti *object.TagIter
	if ti, err = g.r.TagObjects(); chk.E(err) {
		err = log.E.Err("tag objects: %w", err)
		return
	}
	var tg storer.ReferenceIter
	if tg, err = g.r.Tags(); chk.E(err) {
	}
	chk.E(tg.ForEach(func(pr *plumbing.Reference) (err error) {
		log.I.S(pr)
		name := pr.Name().String()
		split := strings.Split(name, "/")
		tags = append(tags, &object.Tag{
			Hash:   pr.Hash(),
			Name:   split[2],
			Target: pr.Hash(),
		})
		return
	}))
	chk.E(ti.ForEach(func(t *object.Tag) (err error) {
		log.I.S(t)
		for i, existing := range tags {
			if existing.Name == t.Name {
				if t.Tagger.When.After(existing.Tagger.When) {
					tags[i] = t
				}
				return
			}
		}
		tags = append(tags, t)
		return
	}))
	t := TagList(tags)
	sort.Sort(t)
	tags = t
	return
}

func (g *GitRepo) Branches() (branches []*plumbing.Reference, err error) {
	var bi storer.ReferenceIter
	if bi, err = g.r.Branches(); chk.E(err) {
		err = log.E.Err("branches: %w", err)
		return
	}
	chk.E(bi.ForEach(func(ref *plumbing.Reference) (err error) {
		branches = append(branches, ref)
		return
	}))
	return
}

func (g *GitRepo) FindMainBranch(branches []string) (b string, err error) {
	for _, b = range branches {
		if _, err = g.r.ResolveRevision(plumbing.Revision(b)); !chk.E(err) {
			return
		}
	}
	err = log.E.Err("unable to find main branch")
	return
}
