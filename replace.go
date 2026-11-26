package ahocorasick

import "fmt"

func replace[T text](a *Automaton, h T, fn func(Match) T) T {
	m, ok := find(a, h, 0)
	if !ok {
		return h
	}
	buf := make([]byte, 0, len(h))
	last := 0
	for ok {
		buf = append(buf, h[last:m.Start]...)
		buf = append(buf, fn(m)...)
		last = m.End
		m, ok = find(a, h, last)
	}
	buf = append(buf, h[last:]...)
	return T(buf)
}

func (a *Automaton) checkReplacements(n int) {
	if n != len(a.patterns) {
		panic(fmt.Sprintf("ahocorasick: got %d replacements for %d patterns", n, len(a.patterns)))
	}
}

// ReplaceAll returns a copy of h in which every non-overlapping match of
// pattern i is replaced by replacements[i].
//
// It panics if len(replacements) != a.PatternCount(). If there is no match,
// h itself is returned.
func (a *Automaton) ReplaceAll(h []byte, replacements [][]byte) []byte {
	a.checkReplacements(len(replacements))
	return replace(a, h, func(m Match) []byte { return replacements[m.Pattern] })
}

// ReplaceAllString is like [Automaton.ReplaceAll] for strings.
func (a *Automaton) ReplaceAllString(h string, replacements []string) string {
	a.checkReplacements(len(replacements))
	return replace(a, h, func(m Match) string { return replacements[m.Pattern] })
}

// ReplaceAllFunc returns a copy of h in which every non-overlapping match is
// replaced by the return value of fn. If there is no match, h itself is
// returned.
func (a *Automaton) ReplaceAllFunc(h []byte, fn func(Match) []byte) []byte {
	return replace(a, h, fn)
}

// ReplaceAllStringFunc is like [Automaton.ReplaceAllFunc] for strings.
func (a *Automaton) ReplaceAllStringFunc(h string, fn func(Match) string) string {
	return replace(a, h, fn)
}
