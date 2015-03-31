package intn

import (
	"bytes"
	"testing"
)

func CreateTestPatch(o, l int) Patch {
	return Patch{Offset: o, P: bytes.Repeat([]byte{byte(o)}, l)}
}

func Test_FindLRIndex(t *testing.T) {
	ps := Patches{
		CreateTestPatch(10, 10),
		CreateTestPatch(30, 10),
		CreateTestPatch(50, 10),
		CreateTestPatch(70, 10),
		PatchSentinel,
	}

	var l, r int

	// 0    10    20    30    40    50    60    70    80    90    ..  INT_MAX
	//       [  0 ]      [  1 ]      [  2 ]      [  3 ]

	//                      40[  ]45
	l, r = ps.FindLRIndex(CreateTestPatch(40, 5))
	if l != 1 || r != 1 {
		t.Errorf("Expected (1, 1) but got (%d, %d)", l, r)
	}
	//   5[xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx]85
	l, r = ps.FindLRIndex(CreateTestPatch(5, 80))
	if l != 0 || r != 3 {
		t.Errorf("Expected (0, 3) but got (%d, %d)", l, r)
	}

	//               25[xxxxxxxxxxxxxxxx]55
	l, r = ps.FindLRIndex(CreateTestPatch(25, 30))
	if l != 1 || r != 2 {
		t.Errorf("Expected (1, 2) but got (%d, %d)", l, r)
	}

	//                 30[xxxxxxxxxxxxxxxx]60
	l, r = ps.FindLRIndex(CreateTestPatch(30, 30))
	if l != 1 || r != 2 {
		t.Errorf("Expected (1, 2) but got (%d, %d)", l, r)
	}

	//                    35[xxxxxxxxxxxxxxxx]65
	l, r = ps.FindLRIndex(CreateTestPatch(25, 30))
	if l != 1 || r != 2 {
		t.Errorf("Expected (1, 2) but got (%d, %d)", l, r)
	}

	//                          45[]46
	l, r = ps.FindLRIndex(CreateTestPatch(45, 1))
	if l != 2 || r != 1 {
		t.Errorf("Expected (2, 1) but got (%d, %d)", l, r)
	}

	//  5[]6
	l, r = ps.FindLRIndex(CreateTestPatch(5, 1))
	if l != 0 || r != -1 {
		t.Errorf("Expected (0, 0) but got (%d, %d)", l, r)
	}

	//       [  0 ]      [  1 ]      [  2 ]      [  3 ]
	//              25[  ]30
	l, r = ps.FindLRIndex(CreateTestPatch(25, 5))
	if l != 1 || r != 0 {
		t.Errorf("Expected (1, 0) but got (%d, %d)", l, r)
	}

	//                    35[]36
	l, r = ps.FindLRIndex(CreateTestPatch(35, 1))
	if l != 1 || r != 1 {
		t.Errorf("Expected (1, 1) but got (%d, %d)", l, r)
	}

	//                                                85[]86
	l, r = ps.FindLRIndex(CreateTestPatch(85, 1))
	if l != 4 || r != 3 {
		t.Errorf("Expected (4, 3) but got (%d, %d)", l, r)
	}
}

func TestReplace(t *testing.T) {
	ps := Patches{
		CreateTestPatch(10, 5),
		CreateTestPatch(20, 5),
		CreateTestPatch(30, 5),
		CreateTestPatch(40, 5),
		PatchSentinel,
	}

	ps = ps.Replace(1, 2, CreateTestPatch(18, 14))
	if len(ps) != 4 {
		t.Errorf("Invalid len: %d, expected: 4", len(ps))
	}
	if ps[0].Offset != 10 {
		t.Errorf("Fail")
	}
	if ps[1].Offset != 18 {
		t.Errorf("Fail")
	}
	if ps[2].Offset != 40 {
		t.Errorf("Fail")
	}
	if ps[3].Offset != PatchSentinel.Offset {
		t.Errorf("Fail")
	}
}

func TestReplace_SingleElem(t *testing.T) {
	ps := Patches{
		CreateTestPatch(10, 5),
		CreateTestPatch(20, 5),
		CreateTestPatch(30, 5),
		PatchSentinel,
	}

	ps = ps.Replace(1, 1, CreateTestPatch(18, 9))
	if len(ps) != 4 {
		t.Errorf("Invalid len: %d, expected: 4", len(ps))
	}
	if ps[0].Offset != 10 {
		t.Errorf("Fail")
	}
	if ps[1].Offset != 18 {
		t.Errorf("Fail")
	}
	if ps[2].Offset != 30 {
		t.Errorf("Fail")
	}
	if ps[3].Offset != PatchSentinel.Offset {
		t.Errorf("Fail")
	}
}

func PatchesToOffsetArrayForTesting(ps Patches) []int {
	ret := make([]int, len(ps))
	for i, p := range ps {
		ret[i] = p.Offset
	}
	return ret
}

func TestMerge_AppendLast(t *testing.T) {
	ps := Patches{PatchSentinel}

	ps = ps.Merge(CreateTestPatch(10, 5))
	if len(ps) != 2 {
		t.Errorf("Invalid len: %d, expected: 2", len(ps))
	}
	if ps[0].Offset != 10 {
		t.Errorf("Fail")
	}
	if ps[1].Offset != PatchSentinel.Offset {
		t.Errorf("Fail")
	}

	ps = ps.Merge(CreateTestPatch(20, 5))
	if len(ps) != 3 {
		t.Errorf("Invalid len: %d, expected: 3", len(ps))
	}
	if ps[0].Offset != 10 {
		t.Errorf("Fail")
	}
	if ps[1].Offset != 20 {
		t.Errorf("Fail")
	}
	if ps[2].Offset != PatchSentinel.Offset {
		t.Errorf("Fail")
	}

	ps = ps.Merge(CreateTestPatch(25, 10))
	if len(ps) != 4 {
		t.Errorf("Invalid len: %d, expected: 4", len(ps))
	}
	if ps[0].Offset != 10 {
		t.Errorf("Fail")
	}
	if ps[1].Offset != 20 {
		t.Errorf("Fail")
	}
	if ps[2].Offset != 25 {
		t.Errorf("Fail")
	}
	if ps[3].Offset != PatchSentinel.Offset {
		t.Errorf("Fail")
	}
}

func TestMerge_Prepend(t *testing.T) {
	ps := Patches{
		CreateTestPatch(10, 5),
		PatchSentinel,
	}

	ps = ps.Merge(CreateTestPatch(0, 3))
	if len(ps) != 3 {
		t.Errorf("Invalid len: %d, expected: 3", len(ps))
	}
	if ps[0].Offset != 0 {
		t.Errorf("Fail")
	}
	if ps[1].Offset != 10 {
		t.Errorf("Fail")
	}
	if ps[2].Offset != PatchSentinel.Offset {
		t.Errorf("Fail")
	}

	ps = ps.Merge(CreateTestPatch(5, 5))
	if ps[0].Offset != 0 {
		t.Errorf("Fail")
	}
	if ps[1].Offset != 5 {
		t.Errorf("Fail")
	}
	if ps[2].Offset != 10 {
		t.Errorf("Fail")
	}
	if ps[3].Offset != PatchSentinel.Offset {
		t.Errorf("Fail")
	}
}

func TestMerge_FullOverwrite(t *testing.T) {
	ps := Patches{
		CreateTestPatch(10, 5),
		CreateTestPatch(20, 5),
		CreateTestPatch(30, 5),
		PatchSentinel,
	}

	ps = ps.Merge(CreateTestPatch(5, 40))
	if len(ps) != 2 {
		t.Errorf("Invalid len: %d, expected: 2", len(ps))
	}
	if ps[0].Offset != 5 {
		t.Errorf("Fail")
	}
	if len(ps[0].P) != 40 {
		t.Errorf("Fail")
	}
	if ps[1].Offset != PatchSentinel.Offset {
		t.Errorf("Fail")
	}
}

func TestMerge_PartialOverwrite(t *testing.T) {
	ps := Patches{
		CreateTestPatch(10, 5),
		CreateTestPatch(20, 5),
		CreateTestPatch(30, 5),
		PatchSentinel,
	}

	ps = ps.Merge(CreateTestPatch(23, 10))
	if len(ps) != 5 {
		t.Errorf("Invalid len: %d, expected: 5", len(ps))
	}
	if ps[0].Offset != 10 {
		t.Errorf("Fail")
	}
	if len(ps[0].P) != 5 {
		t.Errorf("Fail")
	}
	if ps[1].Offset != 20 {
		t.Errorf("Fail")
	}
	if len(ps[1].P) != 3 {
		t.Errorf("Fail")
	}
	if ps[2].Offset != 23 {
		t.Errorf("Fail")
	}
	if len(ps[2].P) != 10 {
		t.Errorf("Fail")
	}
	if ps[3].Offset != 33 {
		t.Errorf("Fail")
	}
	if len(ps[3].P) != 2 {
		t.Errorf("Fail")
	}
	if ps[4].Offset != PatchSentinel.Offset {
		t.Errorf("Fail")
	}
}

func TestMerge_FillHole(t *testing.T) {
	ps := Patches{
		CreateTestPatch(10, 10),
		CreateTestPatch(30, 10),
		PatchSentinel,
	}

	ps = ps.Merge(CreateTestPatch(20, 10))
	if len(ps) != 4 {
		t.Errorf("Invalid len: %d, expected: 4", len(ps))
	}
	if ps[0].Offset != 10 {
		t.Errorf("Fail")
	}
	if len(ps[0].P) != 10 {
		t.Errorf("Fail")
	}
	if ps[1].Offset != 20 {
		t.Errorf("Fail")
	}
	if len(ps[1].P) != 10 {
		t.Errorf("Fail")
	}
	if ps[2].Offset != 30 {
		t.Errorf("Fail")
	}
	if len(ps[2].P) != 10 {
		t.Errorf("Fail")
	}
	if ps[3].Offset != PatchSentinel.Offset {
		t.Errorf("Fail")
	}
}
