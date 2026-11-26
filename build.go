package ahocorasick

import (
	"fmt"
	"math"
	"sort"
)

// NFA state identifiers used during construction.
const (
	nfaDead  uint32 = 0
	nfaStart uint32 = 1
	noTrans  uint32 = math.MaxUint32
)

type nfaTrans struct {
	b    byte
	next uint32
}

type nfaState struct {
	trans   []nfaTrans // sorted by b
	fail    uint32
	matches []uint32 // pattern ids, in reporting order
}

type nfa struct {
	states []nfaState
	// startTrans is a dense copy of the start state's transitions, which are
	// by far the most frequently followed during construction.
	startTrans [256]uint32
}

// follow returns the transition of s on b, or noTrans if there is none.
// The start state loops back to itself and the dead state never leaves.
func (n *nfa) follow(s uint32, b byte) uint32 {
	switch s {
	case nfaStart:
		if t := n.startTrans[b]; t != noTrans {
			return t
		}
		return nfaStart
	case nfaDead:
		return nfaDead
	}
	tr := n.states[s].trans
	i := sort.Search(len(tr), func(i int) bool { return tr[i].b >= b })
	if i < len(tr) && tr[i].b == b {
		return tr[i].next
	}
	return noTrans
}

// child returns the trie child of s on b, or noTrans.
func (n *nfa) child(s uint32, b byte) uint32 {
	if s == nfaStart {
		return n.startTrans[b]
	}
	tr := n.states[s].trans
	i := sort.Search(len(tr), func(i int) bool { return tr[i].b >= b })
	if i < len(tr) && tr[i].b == b {
		return tr[i].next
	}
	return noTrans
}

func (n *nfa) addChild(s uint32, b byte) uint32 {
	id := uint32(len(n.states))
	n.states = append(n.states, nfaState{})
	st := &n.states[s]
	i := sort.Search(len(st.trans), func(i int) bool { return st.trans[i].b >= b })
	st.trans = append(st.trans, nfaTrans{})
	copy(st.trans[i+1:], st.trans[i:])
	st.trans[i] = nfaTrans{b: b, next: id}
	if s == nfaStart {
		n.startTrans[b] = id
	}
	return id
}

func foldByte(b byte) byte {
	if 'A' <= b && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}

func build(patterns []string, opts []Option) (*Automaton, error) {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.kind > LeftmostLongest {
		return nil, fmt.Errorf("%w: %d", ErrInvalidMatchKind, uint8(cfg.kind))
	}
	if uint64(len(patterns)) >= math.MaxUint32 {
		return nil, ErrTooLarge
	}

	key := func(b byte) byte { return b }
	if cfg.caseInsensitive {
		key = foldByte
	}

	// Phase 1: build the trie.
	n := &nfa{states: make([]nfaState, 2, 64)}
	for i := range n.startTrans {
		n.startTrans[i] = noTrans
	}
	var used [256]bool
	totalLen := uint64(0)
	for id, p := range patterns {
		if len(p) == 0 {
			return nil, fmt.Errorf("%w (index %d)", ErrEmptyPattern, id)
		}
		totalLen += uint64(len(p))
		if totalLen >= math.MaxUint32-2 {
			return nil, ErrTooLarge
		}
		s := nfaStart
		skip := false
		for j := 0; j < len(p); j++ {
			// Under leftmost-first semantics a pattern that has an earlier
			// pattern as a prefix can never match, so it is not added.
			if cfg.kind == LeftmostFirst && len(n.states[s].matches) > 0 {
				skip = true
				break
			}
			b := key(p[j])
			used[b] = true
			next := n.child(s, b)
			if next == noTrans {
				next = n.addChild(s, b)
			}
			s = next
		}
		if !skip {
			n.states[s].matches = append(n.states[s].matches, uint32(id))
		}
	}

	// Phase 2: failure links (breadth-first), giving the BFS order used to
	// fill the DFA so that every failure target is filled before its users.
	order := n.fillFailures(cfg.kind.isLeftmost())

	// Phase 3: byte classes.
	var classes [256]uint8
	var rep []byte // representative byte for every class
	anyUnused := false
	for b := 0; b < 256; b++ {
		if !used[key(byte(b))] {
			anyUnused = true
			break
		}
	}
	var keyClass [256]uint8
	if anyUnused {
		rep = append(rep, 0) // class 0: bytes that occur in no pattern
	}
	for b := 0; b < 256; b++ {
		if used[b] {
			keyClass[b] = uint8(len(rep))
			rep = append(rep, byte(b))
		}
	}
	for b := 0; b < 256; b++ {
		k := key(byte(b))
		if used[k] {
			classes[b] = keyClass[k]
		}
	}
	if anyUnused {
		// Pick a representative that really occurs in no pattern.
		for b := 0; b < 256; b++ {
			if !used[key(byte(b))] {
				rep[0] = byte(b)
				break
			}
		}
	}
	stride := uint32(len(rep))

	// Phase 4: number DFA states: dead, match states, start, then the rest.
	numStates := uint64(len(n.states))
	if numStates*uint64(stride) >= math.MaxUint32 {
		return nil, ErrTooLarge
	}
	if cfg.maxMemory > 0 {
		est := numStates*uint64(stride)*4 + totalLen + uint64(len(patterns))*32
		if est > uint64(cfg.maxMemory) {
			return nil, fmt.Errorf("%w: needs about %d bytes, limit is %d", ErrTooLarge, est, cfg.maxMemory)
		}
	}
	index := make([]uint32, len(n.states))
	next := uint32(1)
	matchOff := []uint32{0}
	var matchPats []uint32
	for _, s := range order {
		if len(n.states[s].matches) > 0 {
			index[s] = next
			next++
			matchPats = append(matchPats, n.states[s].matches...)
			matchOff = append(matchOff, uint32(len(matchPats)))
		}
	}
	maxSpecialIdx := next - 1
	index[nfaStart] = next
	next++
	for _, s := range order {
		if s != nfaStart && len(n.states[s].matches) == 0 {
			index[s] = next
			next++
		}
	}

	// Phase 5: fill the dense transition table.
	trans := make([]uint32, numStates*uint64(stride))
	for _, s := range order {
		row := trans[index[s]*stride : (index[s]+1)*stride]
		fail := n.states[s].fail
		for c, b := range rep {
			t := n.follow(s, b)
			switch {
			case t != noTrans:
				row[c] = index[t] * stride
			case fail == nfaDead:
				row[c] = 0
			default:
				// The failure state is shallower, so its row is final.
				row[c] = trans[index[fail]*stride+uint32(c)]
			}
		}
	}
	// The dead state's row is all zeros, i.e. it loops to itself.

	return &Automaton{
		trans:      trans,
		classes:    classes,
		stride:     stride,
		start:      index[nfaStart] * stride,
		maxSpecial: maxSpecialIdx * stride,
		matchOff:   matchOff,
		matchPats:  matchPats,
		patterns:   patterns,
		kind:       cfg.kind,
		fold:       cfg.caseInsensitive,
	}, nil
}

// fillFailures computes failure links and propagates matches along them. It
// returns all live states in breadth-first order, starting with the start
// state.
//
// Under leftmost semantics, failure links of match states (and therefore of
// all their descendants) point to the dead state: once a match has been seen
// the search must not restart at a later position, because the leftmost match
// is already known.
func (n *nfa) fillFailures(leftmost bool) []uint32 {
	order := make([]uint32, 0, len(n.states)-1)
	order = append(order, nfaStart)
	n.states[nfaStart].fail = nfaStart
	for _, t := range n.states[nfaStart].trans {
		order = append(order, t.next)
		if leftmost && len(n.states[t.next].matches) > 0 {
			n.states[t.next].fail = nfaDead
		} else {
			n.states[t.next].fail = nfaStart
		}
	}
	for qi := 1; qi < len(order); qi++ {
		id := order[qi]
		for _, t := range n.states[id].trans {
			child := t.next
			order = append(order, child)
			if leftmost && len(n.states[child].matches) > 0 {
				n.states[child].fail = nfaDead
				continue
			}
			f := n.states[id].fail
			for n.follow(f, t.b) == noTrans {
				f = n.states[f].fail
			}
			f = n.follow(f, t.b)
			n.states[child].fail = f
			if fm := n.states[f].matches; len(fm) > 0 {
				n.states[child].matches = append(n.states[child].matches, fm...)
			}
		}
	}
	return order
}
