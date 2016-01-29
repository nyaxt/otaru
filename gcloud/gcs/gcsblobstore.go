package gcs

import (
	"io"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"google.golang.org/cloud"
	"google.golang.org/cloud/storage"

	"github.com/nyaxt/otaru"
	"github.com/nyaxt/otaru/blobstore"
	oflags "github.com/nyaxt/otaru/flags"
	gcutil "github.com/nyaxt/otaru/gcloud/util"
	"github.com/nyaxt/otaru/logger"
)

var mylog = logger.Registry().Category("gcsblobstore")

type GCSBlobStoreStats struct {
	NumOpenWriter int `json:"num_open_writer"`
	NumOpenReader int `json:"num_open_reader"`
	NumListBlobs  int `json:"num_list_blobs"`
	NumBlobSize   int `json:"num_blob_size"`
	NumRemoveBlob int `json:"num_remove_blob"`
}

type GCSBlobStore struct {
	flags  int
	bucket *storage.BucketHandle

	stats GCSBlobStoreStats
}

var _ = blobstore.BlobStore(&GCSBlobStore{})

func NewGCSBlobStore(projectName string, bucketName string, tsrc oauth2.TokenSource, flags int) (*GCSBlobStore, error) {
	client, err := storage.NewClient(context.Background(), cloud.WithTokenSource(tsrc))
	if err != nil {
		return nil, err
	}
	bucket := client.Bucket(bucketName)

	return &GCSBlobStore{
		flags:  flags,
		bucket: bucket,
	}, nil
}

type Writer struct {
	gcsw *storage.Writer
}

func (bs *GCSBlobStore) OpenWriter(blobpath string) (io.WriteCloser, error) {
	if !oflags.IsWriteAllowed(bs.flags) {
		return nil, otaru.EPERM
	}

	bs.stats.NumOpenWriter++

	obj := bs.bucket.Object(blobpath)
	gcsw := obj.NewWriter(context.Background())
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

	return nil
}

func (bs *GCSBlobStore) tryOpenReaderOnce(blobpath string) (io.ReadCloser, error) {
	bs.stats.NumOpenReader++

	obj := bs.bucket.Object(blobpath)
	rc, err := obj.NewReader(context.Background())
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return nil, blobstore.ENOENT
		}
		return nil, err
	}
	return rc, nil
}

func (bs *GCSBlobStore) OpenReader(blobpath string) (rc io.ReadCloser, err error) {
	gcutil.RetryIfNeeded(func() error {
		rc, err = bs.tryOpenReaderOnce(blobpath)
		return err
	}, mylog)
	return
}

func (bs *GCSBlobStore) Flags() int {
	return bs.flags
}

var _ = blobstore.BlobLister(&GCSBlobStore{})

func (bs *GCSBlobStore) ListBlobs() ([]string, error) {
	bs.stats.NumListBlobs++

	ret := make([]string, 0)

	q := &storage.Query{}
	for q != nil {
		olist, err := bs.bucket.List(context.Background(), q)
		if err != nil {
			return nil, err
		}
		for _, o := range olist.Results {
			blobpath := o.Name
			ret = append(ret, blobpath)
		}
		q = olist.Next
	}

	return ret, nil
}

var _ = blobstore.BlobSizer(&GCSBlobStore{})

func (bs *GCSBlobStore) BlobSize(blobpath string) (int64, error) {
	bs.stats.NumBlobSize++

	object := bs.bucket.Object(blobpath)
	attrs, err := object.Attrs(context.Background())
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return -1, blobstore.ENOENT
		}
		return -1, err
	}

	return attrs.Size, nil
}

var _ = blobstore.BlobRemover(&GCSBlobStore{})

func (bs *GCSBlobStore) RemoveBlob(blobpath string) error {
	bs.stats.NumRemoveBlob++

	object := bs.bucket.Object(blobpath)
	if err := object.Delete(context.Background()); err != nil {
		return err
	}
	return nil
}

func (*GCSBlobStore) ImplName() string { return "GCSBlobStore" }

func (bs *GCSBlobStore) GetStats() GCSBlobStoreStats { return bs.stats }
