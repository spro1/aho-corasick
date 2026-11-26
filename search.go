package ahocorasick

import "iter"

// text is the set of haystack types the search routines are generic over.
// Each instantiation is fully specialized, so strings are searched without
// being copied.
type text interface{ ~string | ~[]byte }

// match builds the i-th match of the special state sid ending at end.
func (a *Automaton) match(sid uint32, i int, end int) Match {
	idx := sid / a.stride
	pid := a.matchPats[int(a.matchOff[idx-1])+i]
	return Match{Pattern: int(pid), Start: end - len(a.patterns[pid]), End: end}
}

// isMatch reports whether h contains at least one match.
func isMatch[T text](a *Automaton, h T) bool {
	trans, classes, sid := a.trans, &a.classes, a.start
	for i := 0; i < len(h); i++ {
		sid = trans[sid+uint32(classes[h[i]])]
		if sid <= a.maxSpecial {
			// The dead state is only reachable after a match state.
			return true
		}
	}
	return false
}

// find returns the first non-overlapping match in h[at:].
func find[T text](a *Automaton, h T, at int) (Match, bool) {
	trans, classes, sid := a.trans, &a.classes, a.start
	if a.kind == Standard {
		for i := at; i < len(h); i++ {
			sid = trans[sid+uint32(classes[h[i]])]
			if sid <= a.maxSpecial {
				return a.match(sid, 0, i+1), true
			}
		}
		return Match{}, false
	}
	var last Match
	found := false
	for i := at; i < len(h); i++ {
		sid = trans[sid+uint32(classes[h[i]])]
		if sid <= a.maxSpecial {
			if sid == 0 {
				break
			}
			last, found = a.match(sid, 0, i+1), true
		}
	}
	return last, found
}

func iterAll[T text](a *Automaton, h T) iter.Seq[Match] {
	return func(yield func(Match) bool) {
		for at := 0; at < len(h); {
			m, ok := find(a, h, at)
			if !ok || !yield(m) {
				return
			}
			at = m.End
		}
	}
}

func iterOverlapping[T text](a *Automaton, h T) iter.Seq[Match] {
	if a.kind != Standard {
		panic("ahocorasick: overlapping search requires the Standard match kind, got " + a.kind.String())
	}
	return func(yield func(Match) bool) {
		trans, classes, sid := a.trans, &a.classes, a.start
		for i := 0; i < len(h); i++ {
			sid = trans[sid+uint32(classes[h[i]])]
			if sid <= a.maxSpecial {
				idx := sid / a.stride
				for j := a.matchOff[idx-1]; j < a.matchOff[idx]; j++ {
					pid := a.matchPats[j]
					m := Match{Pattern: int(pid), Start: i + 1 - len(a.patterns[pid]), End: i + 1}
					if !yield(m) {
						return
					}
				}
			}
		}
	}
}

func collect(seq iter.Seq[Match]) []Match {
	var out []Match
	for m := range seq {
		out = append(out, m)
	}
	return out
}

// IsMatch reports whether any pattern occurs in h. It returns as soon as a
// match is detected and is the fastest way to test for presence.
func (a *Automaton) IsMatch(h []byte) bool { return isMatch(a, h) }

// IsMatchString is like [Automaton.IsMatch] for a string haystack.
func (a *Automaton) IsMatchString(h string) bool { return isMatch(a, h) }

// Find returns the first match in h according to the automaton's
// [MatchKind]. The boolean is false if there is no match.
func (a *Automaton) Find(h []byte) (Match, bool) { return find(a, h, 0) }

// FindString is like [Automaton.Find] for a string haystack.
func (a *Automaton) FindString(h string) (Match, bool) { return find(a, h, 0) }

// FindAll returns all non-overlapping matches in h, in order, according to
// the automaton's [MatchKind]. It returns nil if there are no matches.
func (a *Automaton) FindAll(h []byte) []Match { return collect(iterAll(a, h)) }

// FindAllString is like [Automaton.FindAll] for a string haystack.
func (a *Automaton) FindAllString(h string) []Match { return collect(iterAll(a, h)) }

// FindAllOverlapping returns every occurrence of every pattern in h, ordered
// by end offset; matches with the same end are ordered longest first.
//
// It panics unless the automaton uses the [Standard] match kind.
func (a *Automaton) FindAllOverlapping(h []byte) []Match {
	return collect(iterOverlapping(a, h))
}

// FindAllOverlappingString is like [Automaton.FindAllOverlapping] for a
// string haystack.
func (a *Automaton) FindAllOverlappingString(h string) []Match {
	return collect(iterOverlapping(a, h))
}

// Iter returns an iterator over the non-overlapping matches in h. It does not
// allocate a result slice, and stopping the range loop stops the search.
func (a *Automaton) Iter(h []byte) iter.Seq[Match] { return iterAll(a, h) }

// IterString is like [Automaton.Iter] for a string haystack.
func (a *Automaton) IterString(h string) iter.Seq[Match] { return iterAll(a, h) }

// IterOverlapping returns an iterator over all overlapping matches in h.
//
// It panics unless the automaton uses the [Standard] match kind.
func (a *Automaton) IterOverlapping(h []byte) iter.Seq[Match] { return iterOverlapping(a, h) }

// IterOverlappingString is like [Automaton.IterOverlapping] for a string
// haystack.
func (a *Automaton) IterOverlappingString(h string) iter.Seq[Match] {
	return iterOverlapping(a, h)
}
