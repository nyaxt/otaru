package util

import "syscall"

const (
	EACCES    = syscall.Errno(syscall.EACCES)
	EBADF     = syscall.Errno(syscall.EBADF)
	EEXIST    = syscall.Errno(syscall.EEXIST)
	EISDIR    = syscall.Errno(syscall.EISDIR)
	ENOENT    = syscall.Errno(syscall.ENOENT)
	ENOTDIR   = syscall.Errno(syscall.ENOTDIR)
	ENOTEMPTY = syscall.Errno(syscall.ENOTEMPTY)
	EPERM     = syscall.Errno(syscall.EPERM)
)
