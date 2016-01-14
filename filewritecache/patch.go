package filewritecache

import (
	"fmt"
	"math"

	//"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/util"
)

type Patch struct {
	Offset int64
	P      []byte
}

func (p Patch) Left() int64 {
	return p.Offset
}

func (p Patch) Right() int64 {
	return p.Offset + int64(len(p.P))
}

func (p Patch) String() string {
	return fmt.Sprintf("{Offset: %d, len(P): %d}", p.Offset, len(p.P))
}

type Patches []Patch

var PatchSentinel = Patch{Offset: math.MaxInt64}

func (p Patch) IsSentinel() bool {
	return p.Offset == PatchSentinel.Offset
}

func NewPatches() Patches {
	return Patches{PatchSentinel} // FIXME: should allocate cap == MaxPatches
}

func (ps Patches) FindLRIndex(newp Patch) (int, int) {
	lefti := 0
	for {
		patch := ps[lefti]
		if newp.Left() <= patch.Right() {
			break
		}

		lefti++
		if lefti >= len(ps) {
			panic("failed to find lefti")
		}
	}

	righti := lefti
	for {
		patch := ps[righti]
		if newp.Right() <= patch.Left() {
			break
		}

		righti++
		if righti >= len(ps) {
			panic("failed to find righti")
		}
	}
	righti--

	return lefti, righti
}

func (ps Patches) Replace(lefti, righti int, newps Patches) Patches {
	// logger.Debugf(mylog, "before: %v", ps)
	// logger.Debugf(mylog, "(%d, %d) newps: %v", lefti, righti, newps)

	ndel := util.IntMax(righti-lefti+1, 0)
	nexp := util.IntMax(0, len(newps)-ndel)
	for i := 0; i < nexp; i++ {
		ps = append(ps, PatchSentinel)
	}
	// logger.Debugf(mylog, "ndel: %d, nexp: %d", ndel, nexp)

	newr := len(ps) - nexp + len(newps) - ndel
	// logger.Debugf(mylog, "[%d:%d], [%d:]", lefti+len(newps), newr, righti+1)
	copy(ps[lefti+len(newps):newr], ps[righti+1:])
	copy(ps[lefti:lefti+len(newps)], newps)
	ps = ps[:newr]
	// logger.Debugf(mylog, "after : %v", ps)
	return ps
}

func (ps Patches) Merge(newp Patch) Patches {
	lefti, righti := ps.FindLRIndex(newp)
	// logger.Debugf(mylog, "ps: %v", ps)
	// logger.Debugf(mylog, "newp: %v li, ri (%d, %d)", newp, lefti, righti)

	newps := []Patch{newp}

	if lefti < len(ps)-1 {
		psl := ps[lefti]
		if newp.Left() > psl.Left() {
			//    [<---lefti--->] ...
			//               [<------newp---...
			// logger.Debugf(mylog, "Trim L !!!")

			// Trim L: ps[lefti]
			psl.P = psl.P[:newp.Left()-psl.Left()]
			if len(psl.P) > 0 {
				newps = []Patch{psl, newp}
			}
		}
	}

	if righti >= 0 {
		psr := ps[righti]
		if psr.Right() > newp.Right() {
			// logger.Debugf(mylog, "Trim R !!!")
			//            ... [righti]
			//         ---newp--->]

			// Trim R: ps[righti]
			psr.P = psr.P[newp.Right()-psr.Left():]
			psr.Offset = newp.Right()
			if len(psr.P) != 0 {
				newps = append(newps, psr)
			}
		}
	}

	// Insert newp replacing ps[lefti:righti]
	return ps.Replace(lefti, righti, newps)
}

func (ps Patches) Truncate(size int64) Patches {
	for i := len(ps) - 1; i >= 0; i-- {
		p := &ps[i]

		if p.Left() >= size {
			// drop the patch
			continue
		}

		if p.Right() > size {
			p.P = p.P[:size-p.Left()]
		}
		return append(ps[:i+1], PatchSentinel)
	}

	return NewPatches()
}

func (ps Patches) Reset() Patches {
	return append(ps[:0], PatchSentinel)
}
