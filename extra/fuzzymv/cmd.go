package fuzzymv

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"golang.org/x/text/width"
	"google.golang.org/grpc"

	"github.com/nyaxt/otaru/cli"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/pb"
)

type CacheEntry struct {
	Id       uint64 `json:"id"`
	FullPath string `json:"full_path"`
	Depth    int    `json:"depth"`
}

var RootCE = &CacheEntry{Id: 1, FullPath: "", Depth: 0}

type Cache struct {
	Entries []*CacheEntry
}

const ListDirMaxIds = 10

func openCacheFile(cfg *cli.CliConfig, vhost string, flag int) (*os.File, error) {
	if flag != os.O_RDONLY {
		if err := os.MkdirAll(cfg.FuzzyMvCacheDir, os.ModePerm); err != nil {
			return nil, fmt.Errorf("mkdir(%s): %v", cfg.FuzzyMvCacheDir, err)
		}
	}
	cachepath := path.Join(cfg.FuzzyMvCacheDir, fmt.Sprintf("%v.cache.gz", vhost))
	return os.OpenFile(cachepath, flag, 0644)
}

func populateCache(ctx context.Context, c *Cache, conn *grpc.ClientConn, maxDepth int) error {
	dirs := []*CacheEntry{RootCE}

	fsc := pb.NewFileSystemServiceClient(conn)

	for len(dirs) > 0 {
		id2ce := make(map[uint64]*CacheEntry)
		ids := make([]uint64, 0, ListDirMaxIds)

		var pdirs []*CacheEntry
		if len(dirs) > ListDirMaxIds {
			pdirs = dirs[:ListDirMaxIds]
			dirs = dirs[ListDirMaxIds:]
		} else {
			pdirs = dirs
			dirs = nil
		}
		for _, d := range pdirs {
			ids = append(ids, d.Id)
			id2ce[d.Id] = d
		}

		resp, err := fsc.ListDir(ctx, &pb.ListDirRequest{Id: ids})
		if err != nil {
			return nil
		}
		for _, l := range resp.Listing {
			parent := id2ce[l.DirId]
			for _, e := range l.Entry {
				if e.Type != pb.INodeType_DIR {
					continue
				}

				ce := &CacheEntry{
					Id:       e.Id,
					FullPath: fmt.Sprintf("%s/%s", parent.FullPath, e.Name),
					Depth:    parent.Depth + 1,
				}

				// logger.Debugf(mylog, "append %+v", ce)

				c.Entries = append(c.Entries, ce)
				if ce.Depth < maxDepth {
					dirs = append(dirs, ce)
				}
			}
		}
	}

	return nil
}

func Update(ctx context.Context, cfg *cli.CliConfig, args []string) error {
	fset := flag.NewFlagSet("update", flag.ExitOnError)
	flagMaxDepth := fset.Int("maxDepth", 2, "depth to index")
	fset.Usage = func() {
		fmt.Printf("Usage of %s update:\n", os.Args[0])
		fmt.Printf(" %s update OTARU_VHOST\n", os.Args[0])
		fset.PrintDefaults()
	}
	fset.Parse(args[1:])

	if fset.NArg() != 1 {
		fset.Usage()
		return fmt.Errorf("Invalid number of arguments.")
	}
	vhost := fset.Arg(0)

	c := &Cache{}

	ep, tc, err := cli.ConnectionInfo(cfg, vhost)
	if err != nil {
		return err
	}
	conn, err := cli.DialGrpc(ep, tc)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := populateCache(ctx, c, conn, *flagMaxDepth); err != nil {
		return err
	}
	sort.Slice(c.Entries, func(i, j int) bool {
		return c.Entries[i].FullPath < c.Entries[j].FullPath
	})
	logger.Infof(mylog, "%d Entries", len(c.Entries))
	cmap := make(map[string]interface{})
	for i, ce := range c.Entries {
		cmap[ce.FullPath] = i
	}

	f, err := openCacheFile(cfg, vhost, os.O_RDWR|os.O_TRUNC|os.O_CREATE)
	if err != nil {
		return fmt.Errorf("OpenFile: %v", err)
	}
	defer f.Close()
	w := gzip.NewWriter(f)
	defer w.Close()
	e := json.NewEncoder(w)
	if err := e.Encode(c); err != nil {
		return fmt.Errorf("json encode: %v", err)
	}

	return nil
}

type Match struct {
	*CacheEntry
	Dist int
}

func fold(s string) string {
	return width.Fold.String(strings.ToLower(s))
}

func Search(ctx context.Context, cfg *cli.CliConfig, args []string) error {
	fset := flag.NewFlagSet("search", flag.ExitOnError)
	flagNumCandidates := fset.Int("n", 10, "number of candidates to show.")
	fset.Usage = func() {
		fmt.Printf("Usage of %s search:\n", os.Args[0])
		fmt.Printf(" %s search OTARU_VHOST kw\n", os.Args[0])
		fset.PrintDefaults()
	}
	fset.Parse(args[1:])

	if fset.NArg() != 2 {
		fset.Usage()
		return fmt.Errorf("Invalid number of arguments.")
	}
	vhost := fset.Arg(0)

	c := &Cache{}

	f, err := openCacheFile(cfg, vhost, os.O_RDONLY)
	if err != nil {
		return fmt.Errorf("openCacheFile: %v", err)
	}
	defer f.Close()
	r, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip.NewReader: %v", err)
	}
	defer r.Close()
	d := json.NewDecoder(r)
	if err := d.Decode(c); err != nil {
		return fmt.Errorf("json decode: %v", err)
	}

	kw := fset.Arg(1)
	kwfold := fold(kw)
	logger.Infof(mylog, "kw: %q -> %q len(ent) %v", kw, kwfold, len(c.Entries))

	matches := []Match{}
	for _, e := range c.Entries {
		target := fold(path.Base(e.FullPath))
		dist := fuzzy.RankMatch(kwfold, target)

		//logger.Infof(mylog, "Dist %d Fullpath %v", dist, e.FullPath)
		if dist > 0 && dist < 30 {
			matches = append(matches, Match{e, dist})
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Dist < matches[j].Dist
	})
	if len(matches) > *flagNumCandidates {
		matches = matches[:*flagNumCandidates]
	}
	for _, m := range matches {
		logger.Infof(mylog, "dist %d entry: %v", m.Dist, m.CacheEntry.FullPath)
	}

	return nil
}
