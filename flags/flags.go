package flags

import (
	"bytes"
	"syscall"
)

const (
	O_RDONLY     int = syscall.O_RDONLY
	O_WRONLY     int = syscall.O_WRONLY
	O_RDWR       int = syscall.O_RDWR
	O_CREATE     int = syscall.O_CREAT
	O_EXCL       int = syscall.O_EXCL
	O_TRUNCATE   int = syscall.O_TRUNC
	O_APPEND     int = syscall.O_APPEND
	O_RDWRCREATE int = O_RDWR | O_CREATE
	O_VALIDMASK  int = O_RDONLY | O_WRONLY | O_RDWR | O_CREATE | O_EXCL | O_TRUNCATE | O_APPEND
)

type FlagsReader interface {
	Flags() int
}

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

func IsCreateAllowed(flags int) bool {
	return flags&O_CREATE != 0
}

func IsCreateExclusive(flags int) bool {
	return IsCreateAllowed(flags) && flags&O_EXCL != 0
}

func IsWriteTruncate(flags int) bool {
	return IsWriteAllowed(flags) && flags&O_TRUNCATE != 0
}

func IsWriteAppend(flags int) bool {
	return IsWriteAllowed(flags) && flags&O_APPEND != 0
}

func FlagsToString(flags int) string {
	var b bytes.Buffer
	if IsReadAllowed(flags) {
		b.WriteString("R")
	}
	if IsWriteAllowed(flags) {
		b.WriteString("W")
	}
	if IsCreateAllowed(flags) {
		b.WriteString("C")
	}
	if IsCreateExclusive(flags) {
		b.WriteString("X")
	}
	if IsWriteTruncate(flags) {
		b.WriteString("T")
	}
	if IsWriteAppend(flags) {
		b.WriteString("A")
	}

	return b.String()
}

func Mask(a, b int) int {
	rok := IsReadAllowed(a) && IsReadAllowed(b)
	wok := IsWriteAllowed(a) && IsWriteAllowed(b)
	cok := IsCreateAllowed(a) && IsCreateAllowed(b)

	ret := 0
	if rok && wok {
		ret = O_RDWR
	} else if rok {
		ret = O_RDONLY
	} else if wok {
		ret = O_WRONLY
	}

	if cok {
		ret |= O_CREATE
	}

	return ret
}

func MaskPermMode(pm uint16, flags int) uint16 {
	if !IsReadAllowed(flags) {
		pm &= ^uint16(0444)
	}
	if !IsWriteAllowed(flags) {
		pm &= ^uint16(0222)
	}
	/*
		if !IsCreateAllowed(flags) {
			pm = pm & ^uint16(0111)
		}
	*/

	return pm
}
