package webdav

import (
	"net/http"

	"github.com/nyaxt/otaru/cli"
)

type Marshaler interface {
	WriteResponse(w http.ResponseWriter, entry *Entry, listing Listing)
}

type PropStatMarshaler struct{}

func (m PropStatMarshaler) WriteResponse(w http.ResponseWriter, entry *Entry, listing Listing) {
	w.Write([]byte("ok"))
}

type HtmlMarshaler struct{}

func (m HtmlMarshaler) WriteResponse(w http.ResponseWriter, entry *Entry, listing Listing) {
	// FIXME: implement html marshal
	(PropStatMarshaler{}).WriteResponse(w, entry, listing)
}

type ContentServerMarshaler struct {
	cinfo *cli.ConnectionInfo
}

func (m ContentServerMarshaler) WriteResponse(w http.ResponseWriter, entry *Entry, listing Listing) {
	// FIXME: do http proxy to server's filehandler
	w.Write([]byte("ok"))
}
