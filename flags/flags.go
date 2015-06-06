package flags

import (
	"syscall"
)

const (
	O_RDONLY     int = syscall.O_RDONLY
	O_WRONLY     int = syscall.O_WRONLY
	O_RDWR       int = syscall.O_RDWR
	O_CREATE     int = syscall.O_CREAT
	O_EXCL       int = syscall.O_EXCL
	O_RDWRCREATE int = O_RDWR | O_CREATE
	O_VALIDMASK  int = O_RDONLY | O_WRONLY | O_RDWR | O_CREATE | O_EXCL
)

func IsReadAllowed(flags int) bool {
	mode := flags & syscall.O_ACCMODE
	return mode == O_RDONLY || mode == O_RDWR
}

func IsWriteAllowed(flags int) bool {
	mode := flags & syscall.O_ACCMODE
	return mode == O_WRONLY || mode == O_RDWR
}

func IsReadWriteAllowed(flags int) bool {
	mode := flags & syscall.O_ACCMODE
	return mode == O_RDWR
}
