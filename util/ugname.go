package util

import (
	"os/user"
	"strconv"
)

// FIXME: cache?

func TryUserName(id uint32) string {
	idstr := strconv.FormatUint(uint64(id), 10)
	u, err := user.LookupId(idstr)
	if err != nil {
		return idstr
	}
	return u.Username
}

func TryGroupName(id uint32) string {
	idstr := strconv.FormatUint(uint64(id), 10)
	g, err := user.LookupGroupId(idstr)
	if err != nil {
		return idstr
	}
	return g.Name
}
