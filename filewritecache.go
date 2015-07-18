package otaru

import (
	"log"
	"math"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/intn"
	"github.com/nyaxt/otaru/util"
)

type FileWriteCache struct {
	ps intn.Patches
}

func NewFileWriteCache() *FileWriteCache {
	return &FileWriteCache{ps: intn.NewPatches()}
}

func (wc *FileWriteCache) PWrite(p []byte, offset int64) error {
	pcopy := make([]byte, len(p))
	copy(pcopy, p)

	newp := intn.Patch{Offset: offset, P: pcopy}
	log.Printf("PWrite: %v", newp)
	// log.Printf("PWrite: p=%v", pcopy)

	wc.ps = wc.ps.Merge(newp)
	return nil
}

type ReadAter interface {
	ReadAt(p []byte, offset int64) (int, error)
}

func zerofill(p []byte) {
	for i := range p {
		p[i] = 0
	}
}

func (wc *FileWriteCache) ReadAtThrough(p []byte, offset int64, r ReadAter) (int, error) {
	nr := 0
	remo := offset
	remp := p

	for _, patch := range wc.ps {
		if len(remp) == 0 {
			return nr, nil
		}

		if patch.IsSentinel() {
			break
		}

		if remo > patch.Right() {
			continue
		}

		if remo < patch.Left() {
			fallbackLen64 := util.Int64Min(int64(len(remp)), patch.Left()-remo)
			if fallbackLen64 > math.MaxInt32 {
				panic("Logic error: fallbackLen should always be in int32 range")
			}
			fallbackLen := int(fallbackLen64)

			n, err := r.ReadAt(remp[:fallbackLen], remo)
			log.Printf("BeforePatch: ReadAt issued offset %d, len %d bytes, read %d bytes", remo, fallbackLen, n)
			if err != nil {
				return nr + n, err
			}
			if n < fallbackLen {
				zerofill(remp[n:fallbackLen])
			}

			nr += fallbackLen
			remp = remp[fallbackLen:]
			remo += int64(fallbackLen)
		}

		if len(remp) == 0 {
			return nr, nil
		}

		applyOffset64 := remo - patch.Offset
		if applyOffset64 > math.MaxInt32 {
			panic("Logic error: applyOffset should always be in int32 range")
		}
		applyOffset := int(applyOffset64)
		applyLen := util.IntMin(len(patch.P)-applyOffset, len(remp))
		copy(remp[:applyLen], patch.P[applyOffset:])

		nr += applyLen
		remp = remp[applyLen:]
		remo += int64(applyLen)
	}

	n, err := r.ReadAt(remp, remo)
	log.Printf("Last: ReadAt read %d bytes", n)
	if err != nil {
		return nr, err
	}
	nr += n

	return nr, nil
}

func (wc *FileWriteCache) ContentLen() int64 {
	l := int64(0)
	for _, p := range wc.ps {
		l += int64(len(p.P))
	}
	return l
}

func (wc *FileWriteCache) NeedsSync() bool {
	if len(wc.ps) > FileWriteCacheMaxPatches {
		return true
	}
	if wc.ContentLen() > FileWriteCacheMaxPatchContentLen {
		return true
	}

	return false
}

func (wc *FileWriteCache) Sync(bh blobstore.PWriter) error {
	for _, p := range wc.ps {
		if p.IsSentinel() {
			continue
		}

		log.Printf("Sync: %v", p)
		// log.Printf("Sync: p=%v", p.P)
		if err := bh.PWrite(p.P, p.Offset); err != nil {
			return err
		}
	}
	wc.ps = wc.ps.Reset()

	return nil
}

func (wc *FileWriteCache) Right() int64 {
	if len(wc.ps) == 0 {
		return 0
	}

	return wc.ps[0].Right()
}

func (wc *FileWriteCache) Truncate(size int64) {
	wc.ps = wc.ps.Truncate(size)
}
