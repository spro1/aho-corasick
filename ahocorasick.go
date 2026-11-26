package ahocorasick

import (
	"errors"
	"fmt"
	"unsafe"
)

// MatchKind selects which matches the non-overlapping search methods report.
// See the package documentation for a full description.
type MatchKind uint8

const (
	// Standard reports matches as soon as they are found (earliest end).
	// It is the only kind that supports overlapping search.
	Standard MatchKind = iota
	// LeftmostFirst reports the leftmost match, preferring patterns that
	// were given earlier when several matches start at the same position.
	LeftmostFirst
	// LeftmostLongest reports the leftmost match, preferring the longest
	// pattern when several matches start at the same position.
	LeftmostLongest
)

// String returns the name of the match kind.
func (k MatchKind) String() string {
	switch k {
	case Standard:
		return "Standard"
	case LeftmostFirst:
		return "LeftmostFirst"
	case LeftmostLongest:
		return "LeftmostLongest"
	default:
		return fmt.Sprintf("MatchKind(%d)", uint8(k))
	}
}

func (k MatchKind) isLeftmost() bool { return k == LeftmostFirst || k == LeftmostLongest }

// Errors returned by [New] and [NewBytes].
var (
	// ErrEmptyPattern is returned when one of the patterns is empty.
	ErrEmptyPattern = errors.New("ahocorasick: empty pattern")
	// ErrInvalidMatchKind is returned for an unknown [MatchKind].
	ErrInvalidMatchKind = errors.New("ahocorasick: invalid match kind")
	// ErrTooLarge is returned when the compiled automaton would exceed the
	// addressable state space or the configured memory limit.
	ErrTooLarge = errors.New("ahocorasick: automaton too large")
)

// Match describes a single occurrence of a pattern in a haystack.
//
// Start and End are byte offsets: haystack[Start:End] is the matched text.
type Match struct {
	// Pattern is the index of the matched pattern in the slice passed to New.
	Pattern int
	// Start is the byte offset of the first byte of the match.
	Start int
	// End is the byte offset just past the last byte of the match.
	End int
}

// Len returns the length of the match in bytes.
func (m Match) Len() int { return m.End - m.Start }

// Option configures an [Automaton] at construction time.
type Option func(*config)

type config struct {
	kind            MatchKind
	caseInsensitive bool
	maxMemory       int
}

// WithMatchKind sets the match semantics. The default is [Standard].
func WithMatchKind(kind MatchKind) Option {
	return func(c *config) { c.kind = kind }
}

// WithASCIICaseInsensitive enables or disables ASCII case-insensitive
// matching. Only the letters A-Z and a-z are folded.
func WithASCIICaseInsensitive(enabled bool) Option {
	return func(c *config) { c.caseInsensitive = enabled }
}

// WithMaxMemory limits the approximate heap size, in bytes, of the compiled
// automaton. If the limit would be exceeded, construction fails with
// [ErrTooLarge]. A value <= 0 means no limit (the default). This is useful
// when patterns come from untrusted input.
func WithMaxMemory(bytes int) Option {
	return func(c *config) { c.maxMemory = bytes }
}

// Automaton is a compiled Aho-Corasick automaton. It is immutable and safe for
// concurrent use.
type Automaton struct {
	// trans is the dense transition table. State identifiers are
	// premultiplied by stride, so the next state of sid on byte b is
	// trans[sid+classes[b]].
	trans []uint32
	// classes maps every byte to its equivalence class.
	classes [256]uint8
	stride  uint32
	start   uint32
	// maxSpecial is the largest premultiplied identifier that is either the
	// dead state (0) or a match state. Every sid <= maxSpecial is special.
	maxSpecial uint32

	// Match state with index i (1-based, i = sid/stride) reports patterns
	// matchPats[matchOff[i-1]:matchOff[i]].
	matchOff  []uint32
	matchPats []uint32

	patterns []string
	kind     MatchKind
	fold     bool
}

// New compiles an automaton from string patterns.
//
// Pattern indexes in reported matches correspond to positions in patterns.
// Duplicate patterns are allowed. Empty patterns are rejected with
// [ErrEmptyPattern].
func New(patterns []string, opts ...Option) (*Automaton, error) {
	ps := make([]string, len(patterns))
	copy(ps, patterns)
	return build(ps, opts)
}

// NewBytes is like [New] but takes byte-slice patterns. The patterns are
// copied, so the caller may modify them afterwards.
func NewBytes(patterns [][]byte, opts ...Option) (*Automaton, error) {
	ps := make([]string, len(patterns))
	for i, p := range patterns {
		ps[i] = string(p)
	}
	return build(ps, opts)
}

// MustNew is like [New] but panics if the automaton cannot be built. It is
// intended for package-level variables with fixed patterns.
func MustNew(patterns []string, opts ...Option) *Automaton {
	a, err := New(patterns, opts...)
	if err != nil {
		panic(err)
	}
	return a
}

// PatternCount returns the number of patterns.
func (a *Automaton) PatternCount() int { return len(a.patterns) }

// Pattern returns the pattern with the given index.
func (a *Automaton) Pattern(i int) string { return a.patterns[i] }

// MatchKind returns the match semantics the automaton was built with.
func (a *Automaton) MatchKind() MatchKind { return a.kind }

// ASCIICaseInsensitive reports whether the automaton folds ASCII case.
func (a *Automaton) ASCIICaseInsensitive() bool { return a.fold }

// StateCount returns the number of DFA states, including the dead state.
func (a *Automaton) StateCount() int { return len(a.trans) / int(a.stride) }

// MemoryUsage returns the approximate heap size of the automaton in bytes.
func (a *Automaton) MemoryUsage() int {
	n := int(unsafe.Sizeof(*a))
	n += len(a.trans) * 4
	n += len(a.matchOff)*4 + len(a.matchPats)*4
	n += len(a.patterns) * int(unsafe.Sizeof(""))
	for _, p := range a.patterns {
		n += len(p)
	}
	return n
}
