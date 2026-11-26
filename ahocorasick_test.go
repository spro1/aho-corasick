package ahocorasick

import (
	"bytes"
	"errors"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"testing"
)

// ---------------------------------------------------------------------------
// Reference implementation (quadratic, obviously correct).

func eqFold(a, b byte, fold bool) bool {
	if fold {
		return foldByte(a) == foldByte(b)
	}
	return a == b
}

func occursAt(p, h string, pos int, fold bool) bool {
	if pos+len(p) > len(h) {
		return false
	}
	for i := 0; i < len(p); i++ {
		if !eqFold(p[i], h[pos+i], fold) {
			return false
		}
	}
	return true
}

func naiveOverlapping(ps []string, h string, fold bool) []Match {
	var out []Match
	for end := 1; end <= len(h); end++ {
		var here []Match
		for id, p := range ps {
			if start := end - len(p); start >= 0 && occursAt(p, h, start, fold) {
				here = append(here, Match{id, start, end})
			}
		}
		slices.SortStableFunc(here, func(x, y Match) int { return y.Len() - x.Len() })
		out = append(out, here...)
	}
	return out
}

func naiveFindAll(ps []string, h string, kind MatchKind, fold bool) []Match {
	var out []Match
	for at := 0; at < len(h); {
		m, ok := naiveFind(ps, h, at, kind, fold)
		if !ok {
			break
		}
		out = append(out, m)
		at = m.End
	}
	return out
}

func naiveFind(ps []string, h string, at int, kind MatchKind, fold bool) (Match, bool) {
	if kind == Standard {
		for end := at + 1; end <= len(h); end++ {
			best, ok := Match{}, false
			for id, p := range ps {
				start := end - len(p)
				if start >= at && occursAt(p, h, start, fold) && (!ok || len(p) > best.Len()) {
					best, ok = Match{id, start, end}, true
				}
			}
			if ok {
				return best, true
			}
		}
		return Match{}, false
	}
	for start := at; start < len(h); start++ {
		best, ok := Match{}, false
		for id, p := range ps {
			if !occursAt(p, h, start, fold) {
				continue
			}
			if !ok || (kind == LeftmostLongest && len(p) > best.Len()) {
				best, ok = Match{id, start, start + len(p)}, true
			}
		}
		if ok {
			return best, true
		}
	}
	return Match{}, false
}

// ---------------------------------------------------------------------------

var allKinds = []MatchKind{Standard, LeftmostFirst, LeftmostLongest}

func mustNew(t testing.TB, ps []string, opts ...Option) *Automaton {
	t.Helper()
	a, err := New(ps, opts...)
	if err != nil {
		t.Fatalf("New(%q): %v", ps, err)
	}
	return a
}

func TestClassicExample(t *testing.T) {
	a := mustNew(t, []string{"he", "she", "his", "hers"})
	got := a.FindAllOverlappingString("ushers")
	want := []Match{{1, 1, 4}, {0, 2, 4}, {3, 2, 6}}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestLegacyExample(t *testing.T) {
	// The example from the original version of this package.
	ps := []string{"hi", "his", "shis", "she", "he", "her"}
	h := "he likes her, but she doesn't like him"
	a := mustNew(t, ps)
	got := a.FindAllOverlappingString(h)
	want := naiveOverlapping(ps, h, false)
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for _, m := range got {
		if h[m.Start:m.End] != ps[m.Pattern] {
			t.Fatalf("match %v covers %q, want %q", m, h[m.Start:m.End], ps[m.Pattern])
		}
	}
}

func TestMatchKinds(t *testing.T) {
	ps := []string{"Samwise", "Sam"}
	h := "Samwise"
	tests := []struct {
		kind MatchKind
		want []Match
	}{
		{Standard, []Match{{1, 0, 3}}},
		{LeftmostFirst, []Match{{0, 0, 7}}},
		{LeftmostLongest, []Match{{0, 0, 7}}},
	}
	for _, tt := range tests {
		t.Run(tt.kind.String(), func(t *testing.T) {
			a := mustNew(t, ps, WithMatchKind(tt.kind))
			if got := a.FindAllString(h); !slices.Equal(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}

	// Leftmost-first prefers earlier patterns, leftmost-longest longer ones.
	ps = []string{"abc", "abcdef"}
	if m, _ := mustNew(t, ps, WithMatchKind(LeftmostFirst)).FindString("abcdef"); m != (Match{0, 0, 3}) {
		t.Fatalf("LeftmostFirst: got %v", m)
	}
	if m, _ := mustNew(t, ps, WithMatchKind(LeftmostLongest)).FindString("abcdef"); m != (Match{1, 0, 6}) {
		t.Fatalf("LeftmostLongest: got %v", m)
	}
}

func TestCaseInsensitive(t *testing.T) {
	a := mustNew(t, []string{"GoLang", "rust"}, WithASCIICaseInsensitive(true))
	got := a.FindAllString("I like golang and RUST and GOLANG.")
	want := []Match{{0, 7, 13}, {1, 18, 22}, {0, 27, 33}}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if !a.ASCIICaseInsensitive() {
		t.Fatal("ASCIICaseInsensitive() = false")
	}
	// Non-ASCII bytes are not folded.
	b := mustNew(t, []string{"ÄÖ"}, WithASCIICaseInsensitive(true))
	if b.IsMatchString("äö") {
		t.Fatal("non-ASCII letters must not be folded")
	}
}

func TestUTF8(t *testing.T) {
	ps := []string{"안녕", "세계", "하세"}
	h := "안녕하세요, 세계!"
	a := mustNew(t, ps)
	for _, m := range a.FindAllOverlappingString(h) {
		if h[m.Start:m.End] != ps[m.Pattern] {
			t.Fatalf("bad match %v", m)
		}
	}
	if got := len(a.FindAllString(h)); got != 3 {
		t.Fatalf("got %d matches, want 3", got)
	}
}

func TestBinaryPatterns(t *testing.T) {
	ps := [][]byte{{0x00, 0xff}, {0xff, 0x00, 0xff}}
	var all [][]byte
	for b := 0; b < 256; b++ {
		all = append(all, []byte{byte(b), byte(b)})
	}
	for _, set := range [][][]byte{ps, all} {
		a, err := NewBytes(set)
		if err != nil {
			t.Fatal(err)
		}
		h := []byte{1, 0xff, 0x00, 0xff, 0x00, 0x00, 7, 7}
		strs := make([]string, len(set))
		for i, p := range set {
			strs[i] = string(p)
		}
		if got, want := a.FindAllOverlapping(h), naiveOverlapping(strs, string(h), false); !slices.Equal(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestNoPatterns(t *testing.T) {
	a := mustNew(t, nil)
	if a.IsMatchString("anything") || a.FindAllString("anything") != nil {
		t.Fatal("empty automaton must not match")
	}
	if _, ok := a.FindString("x"); ok {
		t.Fatal("empty automaton must not match")
	}
}

func TestEmptyHaystack(t *testing.T) {
	for _, k := range allKinds {
		a := mustNew(t, []string{"a"}, WithMatchKind(k))
		if a.IsMatch(nil) || a.FindAll(nil) != nil {
			t.Fatal("empty haystack must not match")
		}
	}
}

func TestDuplicatePatterns(t *testing.T) {
	ps := []string{"ab", "ab", "b"}
	a := mustNew(t, ps)
	want := []Match{{0, 0, 2}, {1, 0, 2}, {2, 1, 2}}
	if got := a.FindAllOverlappingString("ab"); !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for _, k := range allKinds {
		a := mustNew(t, ps, WithMatchKind(k))
		if m, _ := a.FindString("ab"); m.Pattern != 0 {
			t.Fatalf("%v: got %v, want pattern 0", k, m)
		}
	}
}

func TestErrors(t *testing.T) {
	if _, err := New([]string{"a", ""}); !errors.Is(err, ErrEmptyPattern) {
		t.Fatalf("got %v, want ErrEmptyPattern", err)
	}
	if _, err := New([]string{"a"}, WithMatchKind(MatchKind(42))); !errors.Is(err, ErrInvalidMatchKind) {
		t.Fatalf("got %v, want ErrInvalidMatchKind", err)
	}
	if _, err := New([]string{strings.Repeat("x", 10000)}, WithMaxMemory(1024)); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("got %v, want ErrTooLarge", err)
	}
	if _, err := New([]string{"abc"}, WithMaxMemory(1<<20)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("MustNew did not panic")
		}
	}()
	MustNew([]string{""})
}

func TestOverlappingPanicsForLeftmost(t *testing.T) {
	a := mustNew(t, []string{"a"}, WithMatchKind(LeftmostFirst))
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	a.FindAllOverlappingString("a")
}

func TestIterEarlyStop(t *testing.T) {
	a := mustNew(t, []string{"a"})
	n := 0
	for range a.IterString("aaaaaaaa") {
		n++
		if n == 3 {
			break
		}
	}
	n = 0
	for range a.IterOverlapping([]byte("aaaaaaaa")) {
		n++
		if n == 2 {
			break
		}
	}
	if n != 2 {
		t.Fatalf("n = %d", n)
	}
}

func TestReplace(t *testing.T) {
	a := mustNew(t, []string{"apple", "banana", "cherry"}, WithMatchKind(LeftmostFirst))
	got := a.ReplaceAllString("apple, banana and cherry", []string{"사과", "바나나", "체리"})
	if want := "사과, 바나나 and 체리"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	gotB := a.ReplaceAll([]byte("one apple"), [][]byte{[]byte("A"), nil, nil})
	if want := "one A"; string(gotB) != want {
		t.Fatalf("got %q, want %q", gotB, want)
	}
	gotF := a.ReplaceAllStringFunc("banana!", func(m Match) string { return strings.ToUpper(a.Pattern(m.Pattern)) })
	if gotF != "BANANA!" {
		t.Fatalf("got %q", gotF)
	}
	gotBF := a.ReplaceAllFunc([]byte("cherry"), func(Match) []byte { return []byte("*") })
	if !bytes.Equal(gotBF, []byte("*")) {
		t.Fatalf("got %q", gotBF)
	}
	if got := a.ReplaceAllString("nothing", []string{"", "", ""}); got != "nothing" {
		t.Fatalf("got %q", got)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on wrong replacement count")
		}
	}()
	a.ReplaceAllString("x", nil)
}

func TestAccessors(t *testing.T) {
	ps := []string{"foo", "bar"}
	a := mustNew(t, ps, WithMatchKind(LeftmostLongest))
	ps[0] = "mutated"
	if a.PatternCount() != 2 || a.Pattern(0) != "foo" || a.MatchKind() != LeftmostLongest {
		t.Fatal("unexpected accessor values")
	}
	if a.StateCount() < 7 || a.MemoryUsage() <= 0 {
		t.Fatalf("StateCount=%d MemoryUsage=%d", a.StateCount(), a.MemoryUsage())
	}
	if MatchKind(9).String() != "MatchKind(9)" {
		t.Fatal(MatchKind(9).String())
	}
	if (Match{Start: 2, End: 5}).Len() != 3 {
		t.Fatal("Len")
	}
}

func TestConcurrentUse(t *testing.T) {
	ps := []string{"needle", "haystack", "edl"}
	a := mustNew(t, ps)
	h := strings.Repeat("a needle in a haystack ", 100)
	want := a.FindAllOverlappingString(h)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			if got := a.FindAllOverlappingString(h); !slices.Equal(got, want) {
				t.Error("concurrent result mismatch")
			}
		})
	}
	wg.Wait()
}

// checkAgainstNaive verifies every search method against the reference
// implementation.
func checkAgainstNaive(t *testing.T, ps []string, h string, kind MatchKind, fold bool) {
	t.Helper()
	a, err := New(ps, WithMatchKind(kind), WithASCIICaseInsensitive(fold))
	if err != nil {
		t.Fatal(err)
	}
	want := naiveFindAll(ps, h, kind, fold)
	if got := a.FindAllString(h); !slices.Equal(got, want) {
		t.Fatalf("FindAll kind=%v fold=%v patterns=%q haystack=%q\n got %v\nwant %v", kind, fold, ps, h, got, want)
	}
	if got := a.FindAll([]byte(h)); !slices.Equal(got, want) {
		t.Fatalf("FindAll([]byte) mismatch")
	}
	if got := a.IsMatchString(h); got != (len(want) > 0) {
		t.Fatalf("IsMatch kind=%v patterns=%q haystack=%q: got %v", kind, ps, h, got)
	}
	m, ok := a.FindString(h)
	if ok != (len(want) > 0) || (ok && m != want[0]) {
		t.Fatalf("Find: got %v %v, want %v", m, ok, want)
	}
	if kind == Standard {
		wantO := naiveOverlapping(ps, h, fold)
		if got := a.FindAllOverlappingString(h); !slices.Equal(got, wantO) {
			t.Fatalf("Overlapping fold=%v patterns=%q haystack=%q\n got %v\nwant %v", fold, ps, h, got, wantO)
		}
	}
}

func randString(r *rand.Rand, alphabet string, minLen, maxLen int) string {
	n := minLen + r.IntN(maxLen-minLen+1)
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[r.IntN(len(alphabet))]
	}
	return string(b)
}

func TestRandomizedAgainstNaive(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	iters := 3000
	if testing.Short() {
		iters = 300
	}
	for range iters {
		alphabet := []string{"ab", "abc", "aAbB", "abcdefgh"}[r.IntN(4)]
		ps := make([]string, 1+r.IntN(8))
		for i := range ps {
			ps[i] = randString(r, alphabet, 1, 5)
		}
		h := randString(r, alphabet, 0, 40)
		for _, k := range allKinds {
			checkAgainstNaive(t, ps, h, k, r.IntN(2) == 0)
		}
	}
}

func FuzzAgainstNaive(f *testing.F) {
	f.Add("he\x00she\x00his\x00hers", "ushers", uint8(0))
	f.Add("Sam\x00Samwise", "Samwise", uint8(1))
	f.Add("abcd\x00bc\x00bcd", "abcdbcd", uint8(2))
	f.Add("aB\x00Ab", "abAB", uint8(3))
	f.Fuzz(func(t *testing.T, patterns, h string, mode uint8) {
		if len(patterns) > 200 || len(h) > 500 {
			return
		}
		var ps []string
		for p := range strings.SplitSeq(patterns, "\x00") {
			if p != "" {
				ps = append(ps, p)
			}
		}
		checkAgainstNaive(t, ps, h, allKinds[int(mode)%3], mode&4 != 0)
	})
}
