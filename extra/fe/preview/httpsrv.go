package preview

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"

	"github.com/nyaxt/otaru/apiserver"
	"github.com/nyaxt/otaru/cli"
	"github.com/nyaxt/otaru/logger"
)

const MaxArchiveSize = 256 * 1024 * 1024 // 256MiB
const MaxPreviewSize = 8 * 1024 * 1024   // 8MiB
const MaxTextPreviewSize = 32 * 1024     // 32KiB

var mylog = logger.Registry().Category("fe-preview")

type server struct {
	cfg *cli.CliConfig
	zp  *zipPreviewer
}

func (s *server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	q := req.URL.Query()
	opath := q.Get("opath")
	idx, err := strconv.Atoi(q.Get("i"))
	if err != nil {
		idx = -1
	}

	ctx := req.Context()

	ext := path.Ext(opath)
	switch ext {
	case ".zip":
		if err := s.zp.Serve(ctx, opath, idx, w); err != nil {
			logger.Warningf(mylog, "zip preview failed: %v", err)
			http.Error(w, "zip preview failed", http.StatusInternalServerError)
			return
		}

	case ".txt", ".json", ".c", ".h", ".md":
		if err := s.dumpAsText(ctx, opath, w); err != nil {
			logger.Warningf(mylog, "text preview failed: %v", err)
			http.Error(w, "text preview failed", http.StatusInternalServerError)
			return
		}

	default:
		http.Error(w, fmt.Sprintf("no preview for %q", ext), http.StatusUnsupportedMediaType)
	}
}

func (s *server) dumpAsText(ctx context.Context, opath string, w http.ResponseWriter) error {
	r, err := cli.NewReader(opath, cli.WithCliConfig(s.cfg), cli.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("Failed to start read of given opath %q. err: %v", opath, err)
	}
	defer r.Close()

	if r.Size() > MaxTextPreviewSize {
		return fmt.Errorf("File too large for text preview.")
	}

	w.Header().Set("Content-Type", "text/plain")
	if _, err := io.Copy(w, r); err != nil {
		return err
	}
	return nil
}

func Install(cfg *cli.CliConfig) apiserver.Option {
	s := &server{
		cfg: cfg,
		zp:  &zipPreviewer{cfg},
	}

	return apiserver.AddMuxHook(func(mux *http.ServeMux) {
		mux.Handle("/preview", s)
	})
}
