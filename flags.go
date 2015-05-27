package otaru

import (
	"github.com/nyaxt/otaru/blobstore"
)

const (
	O_RDONLY     int = blobstore.O_RDONLY
	O_WRONLY     int = blobstore.O_WRONLY
	O_RDWR       int = blobstore.O_RDWR
	O_CREATE     int = blobstore.O_CREATE
	O_EXCL       int = blobstore.O_EXCL
	O_RDWRCREATE int = blobstore.O_RDWRCREATE
	O_VALIDMASK  int = blobstore.O_VALIDMASK
)
