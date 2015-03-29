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

	//              25[  ]30
	l, r = ps.FindLRIndex(CreateTestPatch(25, 5))
	if l != 1 || r != 1 {
		t.Errorf("Expected (1, 1) but got (%d, %d)", l, r)
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

}
