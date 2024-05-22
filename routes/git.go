package routes

import (
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"path/filepath"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5/plumbing/format/pktline"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/server"
)

func (d *deps) InfoRefs(w http.ResponseWriter, r *http.Request) {
	var err error
	name := r.PathValue("name")
	name = filepath.Clean(name)
	repo := filepath.Join(d.c.Repo.ScanPath, name)
	w.Header().Set("content-type", "application/x-git-upload-pack-advertisement")
	var ep *transport.Endpoint
	if ep, err = transport.NewEndpoint("/"); chk.E(err) {
		http.Error(w, err.Error(), 500)
		log.E.F("git: %s", err)
		return
	}
	billyfs := osfs.New(repo)
	loader := server.NewFilesystemLoader(billyfs)
	srv := server.NewServer(loader)
	var sess transport.UploadPackSession
	if sess, err = srv.NewUploadPackSession(ep, nil); chk.E(err) {
		http.Error(w, err.Error(), 500)
		log.E.F("git: %s", err)
		return
	}
	var ar *packp.AdvRefs
	if ar, err = sess.AdvertisedReferencesContext(r.Context()); errors.Is(err,
		transport.ErrRepositoryNotFound) {

		http.Error(w, err.Error(), 404)
		return
	} else if chk.E(err) {
		http.Error(w, err.Error(), 500)
		return
	}
	ar.Prefix = [][]byte{[]byte("# service=git-upload-pack"), pktline.Flush}
	if err = ar.Encode(w); err != nil {
		http.Error(w, err.Error(), 500)
		log.E.F("git: %s", err)
		return
	}
}

func (d *deps) UploadPack(w http.ResponseWriter, r *http.Request) {
	var err error
	name := r.PathValue("name")
	name = filepath.Clean(name)
	repo := filepath.Join(d.c.Repo.ScanPath, name)
	w.Header().Set("content-type", "application/x-git-upload-pack-result")
	upr := packp.NewUploadPackRequest()
	// if err = upr.Decode(r.Body); chk.E(err) {
	var rdr io.Reader
	if rdr, err = gzip.NewReader(r.Body); chk.E(err) {
		http.Error(w, err.Error(), 400)
		return
	}
	if err = upr.Decode(rdr); chk.E(err) {
		http.Error(w, err.Error(), 400)
		return
	}
	// }
	var ep *transport.Endpoint
	ep, err = transport.NewEndpoint("/")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	bfs := osfs.New(repo)
	loader := server.NewFilesystemLoader(bfs)
	svr := server.NewServer(loader)
	var session transport.UploadPackSession
	if session, err = svr.NewUploadPackSession(ep, nil); chk.E(err) {
		http.Error(w, err.Error(), 500)
		log.E.F("git: %s", err)
		return
	}
	var res *packp.UploadPackResponse
	if res, err = session.UploadPack(r.Context(), upr); chk.E(err) {
		http.Error(w, err.Error(), 500)
		log.E.F("git: %s", err)
		return
	}
	if err = res.Encode(w); chk.E(err) {
		http.Error(w, err.Error(), 500)
		log.E.F("git: %s", err)
		return
	}
}
