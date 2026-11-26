// Package ahocorasick implements fast multi-pattern string search using the
// Aho-Corasick algorithm.
//
// An [Automaton] is compiled once from a set of patterns and can then search
// any number of haystacks in time linear in the haystack length, regardless
// of how many patterns there are. Internally the automaton is compiled into a
// dense DFA over byte equivalence classes, so each input byte costs a single
// table lookup.
//
// # Quick start
//
//	ac, err := ahocorasick.New([]string{"he", "she", "his", "hers"})
//	if err != nil {
//		log.Fatal(err)
//	}
//	for m := range ac.IterString("ushers") {
//		fmt.Println(ac.Pattern(m.Pattern), m.Start, m.End)
//	}
//
// # Match semantics
//
// Three [MatchKind] values control which matches are reported by the
// non-overlapping search methods ([Automaton.Find], [Automaton.FindAll],
// [Automaton.Iter], [Automaton.ReplaceAll], ...):
//
//   - [Standard] reports a match as soon as one is detected, i.e. the match
//     with the earliest end position. This is the classical Aho-Corasick
//     behaviour and the only kind that supports overlapping search
//     ([Automaton.FindAllOverlapping], [Automaton.IterOverlapping]).
//   - [LeftmostFirst] reports the match that starts first; among matches
//     starting at the same position, the pattern given first wins. This is
//     the semantics of a regular expression alternation such as "a|ab".
//   - [LeftmostLongest] reports the match that starts first; among matches
//     starting at the same position, the longest pattern wins (POSIX style).
//
// # Offsets and encodings
//
// Patterns and haystacks are treated as raw bytes and all offsets in a
// [Match] are byte offsets, so h[m.Start:m.End] is always the matched text.
// Because UTF-8 is self-synchronizing, searching valid UTF-8 patterns in
// valid UTF-8 text never yields a match that splits a code point.
//
// Case-insensitive matching ([WithASCIICaseInsensitive]) folds only the
// ASCII letters A-Z / a-z.
//
// # Concurrency
//
// An [Automaton] is immutable after construction and safe for concurrent use
// by multiple goroutines.
package ahocorasick
