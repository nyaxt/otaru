package preview

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"path"
	"sort"
	"strings"

	"github.com/nyaxt/otaru/cli"
)

var allowedExt map[string]struct{}

func init() {
	allowedExtList := []string{
		".jpg", ".jpeg", ".png",
	}

	allowedExt = make(map[string]struct{})
	for _, e := range allowedExtList {
		allowedExt[e] = struct{}{}
	}
}

type zipPreviewer struct {
	cfg *cli.CliConfig
}

func (p *zipPreviewer) Serve(ctx context.Context, opath string, idx int, w http.ResponseWriter) error {
	r, err := cli.NewReader(opath, cli.WithCliConfig(p.cfg), cli.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("Failed to start read of given opath %q. err: %v", opath, err)
	}

	size := r.Size()
	if size > MaxArchiveSize {
		return fmt.Errorf("Refusing to read large archive %.02f MB", float64(size)/1024/1024)
	}

	ra, ok := r.(io.ReaderAt)
	if ok {
		defer r.Close()
	} else {
		// FIXME: cache!
		bs, err := ioutil.ReadAll(r)
		if err != nil {
			return fmt.Errorf("ioutil.ReadAll err: %v", err)
		}
		r.Close()
		ra = bytes.NewReader(bs)
	}

	z, err := zip.NewReader(ra, size)
	if err != nil {
		return fmt.Errorf("Failed to open zip reader: %v", err)
	}
	sort.Slice(z.File, func(i, j int) bool {
		n, m := z.File[i].Name, z.File[j].Name
		return strings.Compare(n, m) < 0
	})

	if idx < 0 {
		return listEntries(w, z)
	}
	return extract(w, z, idx)
}

func listEntries(w http.ResponseWriter, z *zip.Reader) error {
	type jentry struct {
		Name string  `json:"name"`
		Size float64 `json:"size"`
	}

	jes := make([]jentry, 0, len(z.File))
	for _, f := range z.File {
		jes = append(jes, jentry{Name: f.Name, Size: float64(f.UncompressedSize64)})
	}

	w.Header().Set("X-Otaru-Preview", "archive-listing")
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	if err := enc.Encode(jes); err != nil {
		return err
	}
	return nil
}

func extract(w http.ResponseWriter, z *zip.Reader, idx int) error {
	if idx >= len(z.File) {
		return fmt.Errorf("out of bounds")
	}

	f := z.File[idx]

	if f.UncompressedSize64 > MaxPreviewSize {
		return fmt.Errorf("File too large for preview.")
	}

	ext := path.Ext(z.File[idx].Name)
	if _, ok := allowedExt[ext]; !ok {
		return fmt.Errorf("Refusing to serve ext %q", ext)
	}
	mtype := mime.TypeByExtension(ext)
	w.Header().Set("Content-Type", mtype)

	r, err := f.Open()
	if err != nil {
		return fmt.Errorf("Failed to open entry: %v", err)
	}
	defer r.Close()
	if _, err := io.Copy(w, r); err != nil {
		return err
	}
	return nil
}
