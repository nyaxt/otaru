package preview

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/nyaxt/otaru/apiserver"
	"github.com/nyaxt/otaru/cli"
	"github.com/nyaxt/otaru/logger"
)

var mylog = logger.Registry().Category("fe-preview")

type server struct {
	cfg *cli.CliConfig
}

type jentry struct {
	Name string
	Size float64
}

func listEntries(w http.ResponseWriter, z *zip.Reader) error {
	jes := make([]jentry, 0, len(z.File))
	for _, f := range z.File {
		jes = append(jes, jentry{Name: f.Name, Size: float64(f.UncompressedSize64)})
	}

	bs, err := json.Marshal(jes)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	if err := enc.Encode(jes); err != nil {
		return err
	}
	return nil
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

	r, err := cli.NewReader(opath, cli.WithCliConfig(s.cfg), cli.WithContext(req.Context()))
	if err != nil {
		http.Error(w, "Failed to start read of given opath.", http.StatusInternalServerError)
		logger.Warningf(mylog, "cli.NewReader(opath: %q). err: %v", opath, err)
		return
	}

	ra, ok := r.(io.ReaderAt)
	if ok {
		defer r.Close()
	} else {
		bs, err := ioutil.ReadAll(r)
		if err != nil {
			http.Error(w, "Failed to ReadAll", http.StatusInternalServerError)
			logger.Warningf(mylog, "ioutil.ReadAll err: ", err)
			return
		}
		r.Close()
		ra = bytes.NewReader(bs)
	}

	size := r.Size()
	// FIXME: if size >

	z, err := zip.NewReader(ra, size)
	if err != nil {
		return fmt.Errorf("Failed to open zip reader: %v", err)
	}
	sort.Slice(z.File, func(i, j int) bool {
		n, m := z.File[i].Name, z.File[j].Name
		return strings.Compare(n, m) < 0
	})

	if idx < 0 {
		if err := listEntries(w, z); err != nil {
			http.Error(w, "Failed to generate entries list")
		}
		return
	}
	http.Error(w, "FIXME", http.StatusInternalServerError)
}

func Install(cfg *cli.CliConfig) apiserver.Option {
	return apiserver.AddMuxHook(func(mux *http.ServeMux) {
		mux.Handle("/preview", &server{cfg})
	})
}
