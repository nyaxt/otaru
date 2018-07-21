package path

import (
	"errors"
	"fmt"
	"log"
	"path"
	"strings"
)

type Path struct {
	Vhost  string
	FsPath string
}

type State int

const (
	Initial State = iota
	AfterOtaruScheme
	BeforeVhost
	BeforeFsPath
	End
)

const OtaruScheme = "otaru:"
const VhostLocal = "[local]"

func advance(st *State, p *Path, s string) (string, error) {
	switch *st {
	case Initial:
		if strings.HasPrefix(s, OtaruScheme) {
			*st = AfterOtaruScheme
			return s[len(OtaruScheme):], nil
		}
		fallthrough

	case AfterOtaruScheme:
		if strings.HasPrefix(s, "//") {
			*st = BeforeVhost
			return s[2:], nil
		}
		*st = BeforeFsPath
		return s, nil

	case BeforeVhost:
		i := strings.Index(s, "/")
		if i < 0 {
			*st = End
			return s, fmt.Errorf("parser: Expected vhost/path, but got \"%s\"", s)
		}
		*st = BeforeFsPath
		p.Vhost, s = s[:i], s[i:]
		return s, nil

	case BeforeFsPath:
		p.FsPath = path.Clean(s)
		*st = End
		return "", nil

	case End:
		return "", nil

	default:
		log.Panicf("Unknown state: %q", *st)
	}
	return "", errors.New("NOTREACHED")
}

func Parse(s string) (Path, error) {
	p := Path{Vhost: "default", FsPath: "/"}
	var err error
	for st := Initial; st != End; s, err = advance(&st, &p, s) {
		// fmt.Printf("state: %v, path: %+v, left: \"%s\"\n", st, p, s)
	}
	return p, err
}
