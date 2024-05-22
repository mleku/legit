package git

import (
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type TextFragment struct {
	Header string
	Lines  []gitdiff.Line
}

type Diff struct {
	Name struct {
		Old string
		New string
	}
	TextFragments []TextFragment
	IsBinary      bool
	IsNew         bool
	IsDelete      bool
}

// A nicer git diff representation.

type NiceDiff struct {
	Commit struct {
		Message string
		Author  object.Signature
		This    string
		Parent  string
	}
	Stat struct {
		FilesChanged int
		Insertions   int
		Deletions    int
	}
	Diff []Diff
}

func (g *GitRepo) Diff() (nd *NiceDiff, err error) {
	var c *object.Commit
	if c, err = g.r.CommitObject(g.h); chk.E(err) {
		return nil, log.E.Err("commit object: %w", err)
	}
	patch := &object.Patch{}
	parent := &object.Commit{}
	var commitTree *object.Tree
	if commitTree, err = c.Tree(); chk.E(err) {
		parentTree := &object.Tree{}
		if c.NumParents() != 0 {
			if parent, err = c.Parents().Next(); !chk.E(err) {
				if parentTree, err = parent.Tree(); !chk.E(err) {
					if patch, err = parentTree.Patch(commitTree); chk.E(err) {
						return nil, log.E.Err("patch: %w", err)
					}
				}
			}
		} else {
			if patch, err = parentTree.Patch(commitTree); chk.E(err) {
				return nil, log.E.Err("patch: %w", err)
			}
		}
	}
	var diffs []*gitdiff.File
	if diffs, _, err = gitdiff.Parse(strings.NewReader(patch.String())); chk.E(err) {
		return
	}

	nd = &NiceDiff{}
	nd.Commit.This = c.Hash.String()
	if parent.Hash.IsZero() {
		nd.Commit.Parent = ""
	} else {
		nd.Commit.Parent = parent.Hash.String()
	}
	nd.Commit.Author = c.Author
	nd.Commit.Message = c.Message
	for _, d := range diffs {
		nDiff := Diff{}
		nDiff.Name.New = d.NewName
		nDiff.Name.Old = d.OldName
		nDiff.IsBinary = d.IsBinary
		nDiff.IsNew = d.IsNew
		nDiff.IsDelete = d.IsDelete
		for _, tf := range d.TextFragments {
			nDiff.TextFragments = append(nDiff.TextFragments, TextFragment{
				Header: tf.Header(),
				Lines:  tf.Lines,
			})
			for _, l := range tf.Lines {
				switch l.Op {
				case gitdiff.OpAdd:
					nd.Stat.Insertions += 1
				case gitdiff.OpDelete:
					nd.Stat.Deletions += 1
				default:
					panic("unhandled default case")
				}
			}
		}
		nd.Diff = append(nd.Diff, nDiff)
	}
	nd.Stat.FilesChanged = len(diffs)
	return
}
