package intn

import (
	"fmt"
	//"log"
	"math"
)

// var Printf = log.Printf
var Printf = func(...interface{}) {}

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
	return Patches{PatchSentinel}
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

func (ps Patches) Replace(lefti, righti int, newp Patch) Patches {
	ps[lefti] = newp
	ndel := righti - lefti
	copy(ps[lefti+1:len(ps)-ndel], ps[righti+1:])
	return ps[:len(ps)-ndel]
}

func (ps Patches) Merge(newp Patch) Patches {
	lefti, righti := ps.FindLRIndex(newp)
	Printf("newp: %v li, ri (%d, %d)\n", newp, lefti, righti)

	if lefti < len(ps)-1 {
		psl := &ps[lefti]
		if newp.Left() > ps[lefti].Left() {
			Printf("Trim L !!!\n")
			//    [lefti] ...
			//       [<------newp---...

			// Trim ps[lefti]
			psl.P = psl.P[:newp.Left()-psl.Left()]
			if len(psl.P) != 0 {
				lefti++
			}
		}
	}

	if righti >= 0 {
		psr := &ps[righti]
		if psr.Right() > newp.Right() {
			Printf("Trim R !!!\n")
			//            ... [righti]
			//         ---newp--->]

			// Trim ps[righti]
			psr.P = psr.P[newp.Right()-psr.Left():]
			psr.Offset = newp.Right()
			if len(psr.P) != 0 {
				righti--
			}
		}
	}

	if lefti > righti {
		Printf("Insert!!!\n")

		// Insert newp @ index lefti
		newps := append(ps, PatchSentinel)
		copy(newps[lefti+1:], newps[lefti:])
		newps[lefti] = newp
		return newps
	}

	// Insert newp replacing ps[lefti:righti]
	return ps.Replace(lefti, righti, newp)
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
