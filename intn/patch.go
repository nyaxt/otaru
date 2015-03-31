package intn

import (
	// "fmt"
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
	// fmt.Printf("newp: %v li, ri (%d, %d)\n", newp, lefti, righti)

	if lefti < len(ps)-1 {
		psl := &ps[lefti]
		if newp.Left() > ps[lefti].Left() {
			// fmt.Printf("Trim L !!!\n")
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
			// fmt.Printf("Trim R !!!\n")
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
		// fmt.Printf("Insert!!!\n")

		// Insert newp @ index lefti
		newps := append(ps, PatchSentinel)
		copy(newps[lefti+1:], newps[lefti:])
		newps[lefti] = newp
		return newps
	}

	// Insert newp replacing ps[lefti:righti]
	return ps.Replace(lefti, righti, newp)
}
