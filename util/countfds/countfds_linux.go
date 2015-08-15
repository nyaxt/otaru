package countfds

import (
	"fmt"
	"io/ioutil"
	"os"
)

func CountFds() int {
	es, err := os.ReadDir(fmt.Sprintf("/proc/%d/fd", os.Getpid()))
	if err != nil {
		return 0
	}
	return len(es)
}
