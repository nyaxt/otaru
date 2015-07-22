package gcs

import (
	"io"

	"golang.org/x/net/context"
	"google.golang.org/cloud"
	"google.golang.org/cloud/storage"

	"github.com/nyaxt/otaru"
	"github.com/nyaxt/otaru/blobstore"
	oflags "github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/gcloud/auth"
)

type GCSBlobStoreStats struct {
	NumOpenWriter int `json:num_open_writer`
	NumOpenReader int `json:num_open_reader`
	NumListBlobs  int `json:num_list_blobs`
	NumBlobSize   int `json:num_blob_size`
	NumRemoveBlob int `json:num_remove_blob`
}

type GCSBlobStore struct {
	projectName string
	bucketName  string
	flags       int
	clisrc      auth.ClientSource

	stats GCSBlobStoreStats
}

var _ = blobstore.BlobStore(&GCSBlobStore{})

func NewGCSBlobStore(projectName string, bucketName string, clisrc auth.ClientSource, flags int) (*GCSBlobStore, error) {
	return &GCSBlobStore{
		projectName: projectName,
		bucketName:  bucketName,
		flags:       flags,
		clisrc:      clisrc,
	}, nil
}

type Writer struct {
	gcsw *storage.Writer
}

func (bs *GCSBlobStore) newAuthedContext(basectx context.Context) context.Context {
	return cloud.NewContext(bs.projectName, bs.clisrc(context.TODO()))
}

func (bs *GCSBlobStore) OpenWriter(blobpath string) (io.WriteCloser, error) {
	if !oflags.IsWriteAllowed(bs.flags) {
		return nil, otaru.EPERM
	}

	bs.stats.NumOpenWriter++

	ctx := bs.newAuthedContext(context.TODO())
	gcsw := storage.NewWriter(ctx, bs.bucketName, blobpath)
	gcsw.ContentType = "application/octet-stream"
	return &Writer{gcsw}, nil
}

func (w *Writer) Write(p []byte) (int, error) {
	return w.gcsw.Write(p)
}

func (w *Writer) Close() error {
	if err := w.gcsw.Close(); err != nil {
		return err
	}

	// obj := w.gcsw.Object()
	// do something???

	return nil
}

func (bs *GCSBlobStore) OpenReader(blobpath string) (io.ReadCloser, error) {
	bs.stats.NumOpenReader++

	ctx := bs.newAuthedContext(context.TODO())
	rc, err := storage.NewReader(ctx, bs.bucketName, blobpath)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return nil, blobstore.ENOENT
		}
		return nil, err
	}
	return rc, nil
}

func (bs *GCSBlobStore) Flags() int {
	return bs.flags
}

var _ = blobstore.BlobLister(&GCSBlobStore{})

func (bs *GCSBlobStore) ListBlobs() ([]string, error) {
	bs.stats.NumListBlobs++

	ctx := bs.newAuthedContext(context.TODO())
	ret := make([]string, 0)

	q := &storage.Query{}
	for q != nil {
		res, err := storage.ListObjects(ctx, bs.bucketName, q)
		if err != nil {
			return nil, err
		}
		for _, o := range res.Results {
			blobpath := o.Name
			ret = append(ret, blobpath)
		}
		q = res.Next
	}

	return ret, nil
}

var _ = blobstore.BlobSizer(&GCSBlobStore{})

func (bs *GCSBlobStore) BlobSize(blobpath string) (int64, error) {
	bs.stats.NumBlobSize++

	ctx := bs.newAuthedContext(context.TODO())

	obj, err := storage.StatObject(ctx, bs.bucketName, blobpath)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return -1, blobstore.ENOENT
		}
		return -1, err
	}

	return obj.Size, nil
}

var _ = blobstore.BlobRemover(&GCSBlobStore{})

func (bs *GCSBlobStore) RemoveBlob(blobpath string) error {
	bs.stats.NumRemoveBlob++

	ctx := bs.newAuthedContext(context.TODO())
	if err := storage.DeleteObject(ctx, bs.bucketName, blobpath); err != nil {
		return err
	}
	return nil
}

func (*GCSBlobStore) ImplName() string { return "GCSBlobStore" }

func (bs *GCSBlobStore) GetStats() GCSBlobStoreStats { return bs.stats }
