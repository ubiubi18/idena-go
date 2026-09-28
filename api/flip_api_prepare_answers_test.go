package api

import (
	"bytes"
	"testing"

	"github.com/idena-network/idena-go/blockchain/types"
	"github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/require"
)

// oldPrepareAnswers reproduces the original prepareAnswers, which for every flip
// linearly scanned all answers and re-parsed each answer's cid. The hoisted
// implementation must produce identical Answers for the same inputs.
func oldPrepareAnswers(answers []FlipAnswer, flips [][]byte, isShort bool) *types.Answers {
	findAnswer := func(hash []byte) *FlipAnswer {
		for _, h := range answers {
			c, err := cid.Parse(h.Hash)
			if err == nil && bytes.Compare(c.Bytes(), hash) == 0 {
				return &h
			}
		}
		return nil
	}
	result := types.NewAnswers(uint(len(flips)))
	reportsCount := 0
	for i, flip := range flips {
		answer := findAnswer(flip)
		if answer == nil {
			continue
		}
		switch answer.Answer {
		case types.None:
			continue
		case types.Left:
			result.Left(uint(i))
		case types.Right:
			result.Right(uint(i))
		}
		if isShort {
			continue
		}
		grade := answer.Grade
		if grade == types.GradeNone && answer.WrongWords != nil && *answer.WrongWords {
			grade = types.GradeReported
		}
		if grade == types.GradeReported {
			reportsCount++
			if float32(reportsCount)/float32(len(flips)) >= 0.34 {
				grade = types.GradeNone
			}
		}
		result.Grade(uint(i), grade)
	}
	return result
}

func mkCid(t *testing.T, seed string) cid.Cid {
	t.Helper()
	h, err := mh.Sum([]byte(seed), mh.SHA2_256, -1)
	require.NoError(t, err)
	return cid.NewCidV1(cid.DagProtobuf, h)
}

func boolp(b bool) *bool { return &b }

func requireSameAnswers(t *testing.T, answers []FlipAnswer, flips [][]byte, isShort bool) {
	t.Helper()
	got := prepareAnswers(answers, flips, isShort)
	want := oldPrepareAnswers(answers, flips, isShort)
	require.Equalf(t, want.Bytes(), got.Bytes(), "isShort=%v", isShort)
}

func TestPrepareAnswers_MatchesOriginalScan(t *testing.T) {
	cA := mkCid(t, "flip-a")
	cB := mkCid(t, "flip-b")
	cC := mkCid(t, "flip-c")
	cD := mkCid(t, "flip-d") // answered but not among flips
	cMissing := mkCid(t, "flip-missing")

	reported := boolp(true)
	answers := []FlipAnswer{
		{Hash: cA.String(), Answer: types.Left, Grade: types.GradeD},
		{Hash: cB.String(), Answer: types.Right, WrongWords: reported}, // -> reported grade
		{Hash: cC.String(), Answer: types.None},                        // None: skipped
		{Hash: cD.String(), Answer: types.Left},                        // no matching flip
		{Hash: "not-a-cid", Answer: types.Right},                       // unparseable: skipped
	}
	// flips include one with no answer (cMissing) and interleaved order.
	flips := [][]byte{cC.Bytes(), cA.Bytes(), cMissing.Bytes(), cB.Bytes()}

	for _, isShort := range []bool{true, false} {
		requireSameAnswers(t, answers, flips, isShort)
	}

	// Spot-check concrete values (long mode).
	got := prepareAnswers(answers, flips, false)
	a1, _ := got.Answer(1) // cA -> Left
	require.Equal(t, types.Left, a1)
	a3, g3 := got.Answer(3) // cB -> Right, reported
	require.Equal(t, types.Right, a3)
	require.Equal(t, types.GradeReported, g3)
	a0, _ := got.Answer(0) // cC -> None -> unset
	require.Equal(t, types.None, a0)
	a2, _ := got.Answer(2) // missing -> unset
	require.Equal(t, types.None, a2)
}

// TestPrepareAnswers_DuplicateCidKeepsFirst pins that when two answers carry the
// same cid, the first one wins -- matching the original first-match scan.
func TestPrepareAnswers_DuplicateCidKeepsFirst(t *testing.T) {
	c := mkCid(t, "dup")
	answers := []FlipAnswer{
		{Hash: c.String(), Answer: types.Left},  // first wins
		{Hash: c.String(), Answer: types.Right}, // shadowed
	}
	flips := [][]byte{c.Bytes()}

	requireSameAnswers(t, answers, flips, false)
	got := prepareAnswers(answers, flips, false)
	a0, _ := got.Answer(0)
	require.Equal(t, types.Left, a0)
}

func TestPrepareAnswers_Empty(t *testing.T) {
	requireSameAnswers(t, nil, nil, false)
	c := mkCid(t, "x")
	requireSameAnswers(t, nil, [][]byte{c.Bytes()}, false) // flips, no answers
}

// TestPrepareAnswers_ReportsThresholdCrossed exercises the "ignore excess
// reports" branch (reportsCount/len(flips) >= 0.34), which none of the other
// cases reach. The branch is order-sensitive: reportsCount accrues in flip
// iteration order, so only the first reported flip keeps GradeReported and the
// rest are cleared to GradeNone once the fraction crosses the threshold. The
// equivalence check pins that the hoisted lookup preserves this accrual; the
// concrete asserts pin the threshold arithmetic itself.
func TestPrepareAnswers_ReportsThresholdCrossed(t *testing.T) {
	cA, cB, cC := mkCid(t, "a"), mkCid(t, "b"), mkCid(t, "c")
	rep := boolp(true)
	answers := []FlipAnswer{
		{Hash: cA.String(), Answer: types.Left, WrongWords: rep},  // reported
		{Hash: cB.String(), Answer: types.Right, WrongWords: rep}, // reported
		{Hash: cC.String(), Answer: types.Left, WrongWords: rep},  // reported
	}
	flips := [][]byte{cA.Bytes(), cB.Bytes(), cC.Bytes()}

	requireSameAnswers(t, answers, flips, false)

	got := prepareAnswers(answers, flips, false)
	_, g0 := got.Answer(0)
	require.Equal(t, types.GradeReported, g0, "1/3=0.33 < 0.34: first report survives")
	_, g1 := got.Answer(1)
	require.Equal(t, types.GradeNone, g1, "2/3 >= 0.34: excess report cleared")
	_, g2 := got.Answer(2)
	require.Equal(t, types.GradeNone, g2, "3/3 >= 0.34: excess report cleared")
}

// TestPrepareAnswers_DuplicateFlipDoubleCountsReports pins that a flip repeated
// in the flips slice is graded at each position and increments reportsCount each
// time -- so a single reported answer at two positions can cross the threshold.
// The hoisted map lookup must reproduce the original per-position scan here too.
func TestPrepareAnswers_DuplicateFlipDoubleCountsReports(t *testing.T) {
	c := mkCid(t, "dup-report")
	answers := []FlipAnswer{
		{Hash: c.String(), Answer: types.Left, WrongWords: boolp(true)}, // reported
	}
	flips := [][]byte{c.Bytes(), c.Bytes()} // same flip twice

	requireSameAnswers(t, answers, flips, false)

	got := prepareAnswers(answers, flips, false)
	// len(flips)==2, so even the first occurrence crosses the threshold
	// (1/2=0.5 >= 0.34); the point here is that one answer is counted at both
	// positions, driving reportsCount to 2.
	_, g0 := got.Answer(0)
	require.Equal(t, types.GradeNone, g0, "1/2=0.5 >= 0.34: first occurrence already cleared")
	_, g1 := got.Answer(1)
	require.Equal(t, types.GradeNone, g1, "2/2=1.0 >= 0.34: second occurrence cleared")
}
