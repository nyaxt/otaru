package fuzzymv

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/schollz/closestmatch"
	"google.golang.org/grpc"

	"github.com/nyaxt/otaru/cli"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/pb"
)

type CacheEntry struct {
	Id       uint64
	FullPath string
	Type     pb.INodeType
	Depth    int
}

var RootCE = &CacheEntry{Id: 1, FullPath: "", Type: pb.INodeType_DIR, Depth: 0}

type Cache struct {
	entries []*CacheEntry
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
				ce := &CacheEntry{
					Id:       e.Id,
					FullPath: fmt.Sprintf("%s/%s", parent.FullPath, e.Name),
					Type:     e.Type,
					Depth:    parent.Depth + 1,
				}

				// logger.Debugf(mylog, "append %+v", ce)

				c.entries = append(c.entries, ce)
				if ce.Depth < maxDepth && e.Type == pb.INodeType_DIR {
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

	/*
		if fset.NArg() != 1 {
			fset.Usage()
			return fmt.Errorf("Invalid number of arguments.")
		}
	*/
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
	sort.Slice(c.entries, func(i, j int) bool {
		return c.entries[i].FullPath < c.entries[j].FullPath
	})
	logger.Infof(mylog, "%d entries", len(c.entries))
	fullpaths := make([]string, 0, len(c.entries))
	for _, ce := range c.entries {
		fullpaths = append(fullpaths, ce.FullPath)
	}
	cm := closestmatch.New(fullpaths, []int{2})
	logger.Infof(mylog, "cm init done")
	fmt.Println(cm.ClosestN(fset.Arg(1), 3))

	return nil
}
