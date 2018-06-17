package fuzzymv

import (
	"compress/gzip"
	"context"
	"encoding/gob"
	"flag"
	"fmt"
	"os"
	"path"
	"sort"

	"github.com/auxesis/closestmatch"
	"google.golang.org/grpc"

	"github.com/nyaxt/otaru/cli"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/pb"
)

type CacheEntry struct {
	Id       uint64
	FullPath string
	Depth    int
}

var RootCE = &CacheEntry{Id: 1, FullPath: "", Depth: 0}

type Cache struct {
	Entries []*CacheEntry
	CM      *closestmatch.ClosestMatch
}

const ListDirMaxIds = 10

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
	c.CM = closestmatch.New(cmap, []int{2})
	logger.Infof(mylog, "cm init done")

	if err := os.MkdirAll(cfg.FuzzyMvCacheDir, os.ModePerm); err != nil {
		return fmt.Errorf("mkdir(%s): %v", cfg.FuzzyMvCacheDir, err)
	}
	cachepath := path.Join(cfg.FuzzyMvCacheDir, fmt.Sprintf("%v.cache", vhost))
	f, err := os.OpenFile(cachepath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("OpenFile: %v", err)
	}
	defer f.Close()
	w := gzip.NewWriter(f)
	defer w.Close()
	e := gob.NewEncoder(w)
	if err := e.Encode(c); err != nil {
		return fmt.Errorf("gob encode: %v", err)
	}

	/*
		matches := cm.ClosestN(fset.Arg(1), 3)
		for _, m := range matches {
			i := m.Data.(int)
			logger.Infof(mylog, "match: %v score: %d", c.Entries[i], m.Score)
		}
	*/

	return nil
}
