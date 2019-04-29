package gcs

import (
	"io"

	"cloud.google.com/go/storage"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/nyaxt/otaru/blobstore"
	oflags "github.com/nyaxt/otaru/flags"
	gcutil "github.com/nyaxt/otaru/gcloud/util"
	"github.com/nyaxt/otaru/logger"
	oprometheus "github.com/nyaxt/otaru/prometheus"
	"github.com/nyaxt/otaru/util"
)

var mylog = logger.Registry().Category("gcsblobstore")

const promSubsystem = "gcsblobstore"

var issuedOps = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: prometheus.BuildFQName(oprometheus.Namespace, promSubsystem, "issued_ops"),
		Help: "Number of Google Cloud Storage operations issued, partitioned by bucket name and operation type",
	},
	[]string{"optype", "bucketName"},
)
var (
	issuedOpenWriterOps = issuedOps.MustCurryWith(prometheus.Labels{"optype": "openWriter"})
	issuedOpenReaderOps = issuedOps.MustCurryWith(prometheus.Labels{"optype": "openReader"})
	issuedBlobSizeOps   = issuedOps.MustCurryWith(prometheus.Labels{"optype": "blobSize"})
	issuedListBlobOps   = issuedOps.MustCurryWith(prometheus.Labels{"optype": "listBlob"})
	issuedRemoveBlobOps = issuedOps.MustCurryWith(prometheus.Labels{"optype": "RemoveBlob"})
)

func init() {
	prometheus.MustRegister(issuedOps)
}

type GCSBlobStore struct {
	flags      int
	bucket     *storage.BucketHandle
	bucketName string
}

var _ = blobstore.BlobStore(&GCSBlobStore{})

func NewGCSBlobStore(projectName string, bucketName string, tsrc oauth2.TokenSource, flags int) (*GCSBlobStore, error) {
	client, err := storage.NewClient(context.Background(), option.WithTokenSource(tsrc))
	if err != nil {
		return nil, err
	}
	bucket := client.Bucket(bucketName)

	return &GCSBlobStore{
		flags:      flags,
		bucket:     bucket,
		bucketName: bucketName,
	}, nil
}

type Writer struct {
	gcsw *storage.Writer
}

func (bs *GCSBlobStore) OpenWriter(blobpath string) (io.WriteCloser, error) {
	if !oflags.IsWriteAllowed(bs.flags) {
		return nil, util.EACCES
	}

	issuedOpenWriterOps.WithLabelValues(bs.bucketName).Inc()
	logger.Infof(mylog, "OpenWriter(bucketName: %q, %q)", bs.bucketName, blobpath)

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
	issuedOpenReaderOps.WithLabelValues(bs.bucketName).Inc()
	logger.Infof(mylog, "OpenReader(bucketName: %q, %q)", bs.bucketName, blobpath)

	obj := bs.bucket.Object(blobpath)
	rc, err := obj.NewReader(context.Background())
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return nil, util.ENOENT
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
	issuedListBlobOps.WithLabelValues(bs.bucketName).Inc()
	logger.Infof(mylog, "ListBlobs(bucketName: %q) started.", bs.bucketName)
	defer func() {
		logger.Infof(mylog, "ListBlobs(bucketName: %q) done.", bs.bucketName)
	}()

	ret := make([]string, 0)

	it := bs.bucket.Objects(context.Background(), &storage.Query{})
	for {
		oattr, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		ret = append(ret, oattr.Name)
	}

	return ret, nil
}

var _ = blobstore.BlobSizer(&GCSBlobStore{})

func (bs *GCSBlobStore) BlobSize(blobpath string) (int64, error) {
	issuedBlobSizeOps.WithLabelValues(bs.bucketName).Inc()

	object := bs.bucket.Object(blobpath)
	attrs, err := object.Attrs(context.Background())
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return -1, util.ENOENT
		}
		return -1, err
	}

	logger.Infof(mylog, "BlobSize(bucketName: %q, %q) -> %d", bs.bucketName, blobpath, attrs.Size)
	return attrs.Size, nil
}

var _ = blobstore.BlobRemover(&GCSBlobStore{})

func (bs *GCSBlobStore) RemoveBlob(blobpath string) error {
	if !oflags.IsWriteAllowed(bs.flags) {
		return util.EACCES
	}

	issuedRemoveBlobOps.WithLabelValues(bs.bucketName).Inc()
	logger.Infof(mylog, "RemoveBlob(bucketName: %q, %q)", bs.bucketName, blobpath)

	object := bs.bucket.Object(blobpath)
	if err := object.Delete(context.Background()); err != nil {
		return err
	}
	return nil
}

func (*GCSBlobStore) ImplName() string { return "GCSBlobStore" }
