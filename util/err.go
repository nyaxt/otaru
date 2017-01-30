package util

import (
	"syscall"

	"github.com/nyaxt/fuse"
)

type Error syscall.Errno

const (
	EACCES    = Error(syscall.EACCES)
	EBADF     = Error(syscall.EBADF)
	EEXIST    = Error(syscall.EEXIST)
	EISDIR    = Error(syscall.EISDIR)
	ENFILE    = Error(syscall.ENFILE)
	ENOENT    = Error(syscall.ENOENT)
	ENOTDIR   = Error(syscall.ENOTDIR)
	ENOTEMPTY = Error(syscall.ENOTEMPTY)
	EPERM     = Error(syscall.EPERM)
)

func (e Error) Errno() fuse.Errno {
	return fuse.Errno(e)
}

func (e Error) Error() string {
	return syscall.Errno(e).Error()
}

func IsNotExist(e error) bool {
	e2, ok := e.(Error)
	if !ok {
		return false
	}
	return e2 == ENOENT
}
