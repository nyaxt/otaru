package intn

import (
	"math"
)

type Patch struct {
	Offset int
	P      []byte
}

func (p Patch) Left() int {
	return p.Offset
}

func (p Patch) Right() int {
	return p.Offset + len(p.P)
}

type Patches []Patch

var PatchSentinel = Patch{Offset: math.MaxInt64}

func (ps Patches) FindLRIndex(newp Patch) (int, int) {
	lefti := 0
	for {
		patch := ps[lefti]
		if newp.Left() < patch.Right() {
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
		if newp.Right() < patch.Left() {
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

	if lefti > righti {
		// [lefti-1]          [lefti]
		//            [newp]

		// Insert newp @ index lefti
		newps := append(ps, PatchSentinel)
		copy(newps[lefti+1:], newps[lefti:])
		newps[lefti] = newp
		return newps
	}

	psl := ps[lefti]
	if newp.Left() > psl.Left() {
		//    [lefti] ...
		//       [<------newp---...

		// Modify newp to include ps[lefti]
		newp.P = append(psl.P[:newp.Left()-psl.Left()], newp.P...)
	}

	psr := ps[righti]
	if psr.Right() > newp.Right() {
		//            ... [righti]
		//         ---newp--->]

		// Modify newp to include ps[righti]
		newp.P = append(newp.P, psr.P[psr.Right()-newp.Left():]...)
	}

	// Insert newp replacing ps[lefti:righti]
	return ps.Replace(lefti, righti, newp)
}
