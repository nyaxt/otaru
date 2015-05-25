package blobstore

import (
	"os"
)

const (
	O_RDONLY     int = os.O_RDONLY
	O_WRONLY     int = os.O_WRONLY
	O_RDWR       int = os.O_RDWR
	O_CREATE     int = os.O_CREATE
	O_EXCL       int = os.O_EXCL
	O_RDWRCREATE int = O_RDWR | O_CREATE
	O_VALIDMASK  int = O_RDONLY | O_WRONLY | O_RDWR | O_CREATE | O_EXCL
)

func IsReadAllowed(flags int) bool {
	return flags&(O_RDWR|O_RDONLY) != 0
}

func IsWriteAllowed(flags int) bool {
	return flags&(O_RDWR|O_WRONLY) != 0
}

func IsReadWriteAllowed(flags int) bool {
	return flags&O_RDWR != 0
}
