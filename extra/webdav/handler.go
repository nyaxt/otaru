package webdav

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"google.golang.org/grpc"

	"github.com/nyaxt/otaru/cli"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/pb"
)

const MethodPropFind = "PROPFIND"

func ServeOptions(w http.ResponseWriter) {
	// w.Header().Set("Allow", "OPTIONS, LOCK, DELETE, PROPPATCH, COPY, MOVE, UNLOCK, PROPFIND")
	w.Header().Set("Allow", "OPTIONS, PROPFIND")
	w.Header().Set("DAV", "1")
	w.Header().Set("Ms-Author-Via", "DAV")
}

type Handler struct {
	cfg *cli.CliConfig
}

func (h *Handler) VhostListing() Listing {
	es := make([]*Entry, 0, len(h.cfg.Host))
	for vhost, _ := range h.cfg.Host {
		e := &Entry{Name: vhost, Size: 0, ModifiedTime: time.Now(), IsDir: true}
		es = append(es, e)
	}
	sort.Slice(es, func(i, j int) bool {
		return es[i].Name < es[j].Name
	})
	return es
}

func (h *Handler) EntryForPath(ctx context.Context, conn *grpc.ClientConn, p string) (*Entry, error) {
	fsc := pb.NewFileSystemServiceClient(conn)

	logger.Debugf(mylog, "path: %q", p)
	fnresp, err := fsc.FindNodeFullPath(ctx, &pb.FindNodeFullPathRequest{Path: p})
	if err != nil {
		return nil, ErrorFromGrpc(err, "FindNodeFullPath")
	}

	aresp, err := fsc.Attr(ctx, &pb.AttrRequest{Id: fnresp.Id})
	if err != nil {
		return nil, ErrorFromGrpc(err, "Attr")
	}
	entry := INodeViewToEntry(aresp.Entry)

	return entry, nil
}

func (h *Handler) ListingForId(ctx context.Context, conn *grpc.ClientConn, id uint64) (Listing, error) {
	fsc := pb.NewFileSystemServiceClient(conn)

	resp, err := fsc.ListDir(ctx, &pb.ListDirRequest{Id: []uint64{id}})
	if err != nil {
		return nil, ErrorFromGrpc(err, "ListDir")
	}
	if len(resp.Listing) != 1 {
		return nil, fmt.Errorf("Expected 1 listing, but got %d listings.", len(resp.Listing))
	}
	ls := resp.Listing[0].Entry
	es := make([]*Entry, 0, len(ls))
	for _, l := range ls {
		e := INodeViewToEntry(l)
		es = append(es, e)
	}
	return Listing(es), nil
}

func ParseURLPath(p string) (string, string, error) {
	if len(p) < 2 {
		return "", "", errors.New("Input string too short.")
	}
	if p[0] != '/' {
		return "", "", errors.New("Input string doesn't start with /.")
	}
	p = p[1:]

	si := strings.Index(p, "/")
	if si < 0 {
		return p, "/", nil
	}

	vhost, fspath := p[0:si], p[si:]
	if vhost == "" {
		return "", "", errors.New("Empty vhost.")
	}
	if fspath == "" {
		fspath = "/"
	}
	var err error
	vhost, err = url.PathUnescape(vhost)
	if err != nil {
		return "", "", errors.New("Failed to url.PathUnescape vhost.")
	}
	fspath, err = url.PathUnescape(fspath)
	if err != nil {
		return "", "", errors.New("Failed to url.PathUnescape fspath.")
	}
	return vhost, fspath, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	if p == "/" {
		ls := h.VhostListing()

		var marshaler Marshaler
		switch r.Method {
		case http.MethodOptions:
			ServeOptions(w)
			return
		case http.MethodGet, http.MethodHead:
			marshaler = HtmlMarshaler{}
		case MethodPropFind:
			marshaler = PropStatMarshaler{}
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
		marshaler.WriteResponse(w, "/", &Entry{ModifiedTime: time.Now(), IsDir: true}, ls)
		return
	}

	vhost, fspath, err := ParseURLPath(p)
	if err != nil {
		http.Error(w, fmt.Sprintf("ParseURLPath: %v", err), http.StatusNotFound)
		return
	}

	cinfo, err := cli.QueryConnectionInfo(h.cfg, vhost)
	if err != nil {
		http.Error(w, fmt.Sprintf("QueryConnectionInfo: %v", err), http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	conn, err := cinfo.DialGrpc(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("DialGrpc: %v", err), http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	entry, err := h.EntryForPath(ctx, conn, fspath)
	if err != nil {
		WriteError(w, err)
		return
	}

	needLs := false
	var marshaler Marshaler
	switch r.Method {
	case http.MethodOptions:
		ServeOptions(w)
		return
	case http.MethodGet, http.MethodHead:
		if entry.IsDir {
			needLs = true
			marshaler = HtmlMarshaler{}
		} else {
			marshaler = ContentServerMarshaler{OrigReq: r, CInfo: cinfo}
		}
	case MethodPropFind:
		needLs = true //needLs = r.Header.Get("Depth") != "0"
		marshaler = PropStatMarshaler{}
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}

	var listing Listing
	if needLs {
		var err error
		listing, err = h.ListingForId(ctx, conn, entry.Id)
		if err != nil {
			WriteError(w, err)
			return
		}
	}

	basepath := r.URL.EscapedPath()
	marshaler.WriteResponse(w, basepath, entry, listing)
}
