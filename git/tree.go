package git

import (
	"os"

	"github.com/go-git/go-git/v5/plumbing/object"
)

func (g *GitRepo) FileTree(path string) (files []NiceTree, err error) {
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
	if path == "" {
		if files, err = makeNiceTree(tree); chk.E(err) {
			return
		}
	} else {
		var o *object.TreeEntry
		if o, err = tree.FindEntry(path); chk.E(err) {
			return
		}
		if !o.Mode.IsFile() {
			var subtree *object.Tree
			if subtree, err = tree.Tree(path); chk.E(err) {
				return
			}
			if files, err = makeNiceTree(subtree); chk.E(err) {
				return
			}
		}
	}
	return
}

// A nicer git tree representation.

type NiceTree struct {
	Name      string
	Mode      string
	Size      int64
	IsFile    bool
	IsSubtree bool
}

func makeNiceTree(t *object.Tree) (nts []NiceTree, err error) {
	for _, e := range t.Entries {
		var mode os.FileMode
		if mode, err = e.Mode.ToOSFileMode(); chk.E(err) {
			return
		}
		var sz int64
		if sz, err = t.Size(e.Name); chk.E(err) {
			return
		}
		nts = append(nts, NiceTree{
			Name:   e.Name,
			Mode:   mode.String(),
			IsFile: e.Mode.IsFile(),
			Size:   sz,
		})
	}
	return
}
