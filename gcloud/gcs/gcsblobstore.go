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

type GCSBlobStore struct {
	projectName string
	bucketName  string
	flags       int
	clisrc      auth.ClientSource
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

var _ = blobstore.BlobRemover(&GCSBlobStore{})

func (bs *GCSBlobStore) RemoveBlob(blobpath string) error {
	ctx := bs.newAuthedContext(context.TODO())
	if err := storage.DeleteObject(ctx, bs.bucketName, blobpath); err != nil {
		return err
	}
	return nil
}

func (*GCSBlobStore) ImplName() string { return "GCSBlobStore" }
