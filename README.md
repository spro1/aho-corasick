# aho-corasick

[![CI](https://github.com/spro1/aho-corasick/actions/workflows/ci.yml/badge.svg)](https://github.com/spro1/aho-corasick/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/spro1/aho-corasick.svg)](https://pkg.go.dev/github.com/spro1/aho-corasick)
[![Go Report Card](https://goreportcard.com/badge/github.com/spro1/aho-corasick)](https://goreportcard.com/report/github.com/spro1/aho-corasick)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

**English** | [한국어](README.ko.md)

A fast, zero-dependency [Aho-Corasick](https://en.wikipedia.org/wiki/Aho%E2%80%93Corasick_algorithm)
multi-pattern string search library for Go.

Compile a set of patterns once, then find all of them in any text in a single
linear pass, no matter how many patterns there are.

## Features

- **Fast**: patterns are compiled into a dense DFA over byte equivalence
  classes, so each input byte costs one table lookup. No allocations while
  iterating.
- **Three match semantics**: `Standard` (classic, supports overlapping
  matches), `LeftmostFirst` (like regex alternation `a|ab`) and
  `LeftmostLongest` (POSIX-style longest match).
- **ASCII case-insensitive** matching.
- **Go 1.23+ iterators** (`iter.Seq[Match]`): stop early simply by `break`ing.
- **Replace** with fixed replacements or a callback.
- **`string` and `[]byte`** APIs, with no copying of string haystacks.
- **Safe for concurrent use**: an `Automaton` is immutable after construction.
- **Memory limits** (`WithMaxMemory`) for building from untrusted patterns.
- **Byte-exact offsets** that are safe for UTF-8 text.
- Thoroughly tested: table tests, a randomized differential test and a fuzz
  test against a naive reference implementation, ~97% coverage.

## Installation

```sh
go get github.com/spro1/aho-corasick
```

Requires Go 1.27 or newer. There are no dependencies outside the standard
library.

## Quick start

```go
package main

import (
	"fmt"
	"log"

	ahocorasick "github.com/spro1/aho-corasick"
)

func main() {
	ac, err := ahocorasick.New([]string{"he", "she", "his", "hers"})
	if err != nil {
		log.Fatal(err)
	}

	for m := range ac.IterOverlappingString("ushers") {
		fmt.Printf("%q at [%d:%d]\n", ac.Pattern(m.Pattern), m.Start, m.End)
	}
	// "she" at [1:4]
	// "he" at [2:4]
	// "hers" at [2:6]
}
```

## Usage

### Match semantics

```go
patterns := []string{"Sam", "Samwise"}

std := ahocorasick.MustNew(patterns) // Standard
lf := ahocorasick.MustNew(patterns, ahocorasick.WithMatchKind(ahocorasick.LeftmostFirst))
ll := ahocorasick.MustNew(patterns, ahocorasick.WithMatchKind(ahocorasick.LeftmostLongest))

std.FindString("Samwise") // "Sam"     – earliest match end
lf.FindString("Samwise")  // "Sam"     – leftmost; earlier pattern wins ties
ll.FindString("Samwise")  // "Samwise" – leftmost; longest pattern wins ties
```

| Kind              | Non-overlapping search reports …                          | Overlapping search |
|-------------------|-----------------------------------------------------------|:------------------:|
| `Standard`        | the match that **ends** first (classic Aho-Corasick)      | ✅                 |
| `LeftmostFirst`   | the leftmost match; ties go to the pattern listed first  | –                  |
| `LeftmostLongest` | the leftmost match; ties go to the longest pattern        | –                  |

For tokenizing, highlighting or replacing, `LeftmostFirst` or
`LeftmostLongest` usually gives the most intuitive results.

### Searching

```go
ac := ahocorasick.MustNew([]string{"error", "warn"}, ahocorasick.WithASCIICaseInsensitive(true))

ac.IsMatchString(line)            // bool, returns at the first match
ac.FindString(line)               // (Match, bool), first match
ac.FindAllString(line)            // []Match, all non-overlapping matches
ac.FindAllOverlappingString(line) // []Match, every occurrence (Standard only)

for m := range ac.IterString(line) { // allocation-free iteration
	fmt.Println(line[m.Start:m.End])
}
```

Every method also has a `[]byte` variant without the `String` suffix
(`IsMatch`, `Find`, `FindAll`, `Iter`, ...).

A `Match` has the pattern index and byte offsets:

```go
type Match struct {
	Pattern int // index into the patterns passed to New
	Start   int // haystack[Start:End] is the matched text
	End     int
}
```

### Replacing

```go
ac := ahocorasick.MustNew([]string{"apple", "banana"},
	ahocorasick.WithMatchKind(ahocorasick.LeftmostLongest))

ac.ReplaceAllString("apple and banana", []string{"사과", "바나나"})
// "사과 and 바나나"

ac.ReplaceAllStringFunc(text, func(m ahocorasick.Match) string {
	return strings.Repeat("*", m.Len())
})
```

### Options

| Option                            | Description                                                     |
|-----------------------------------|-----------------------------------------------------------------|
| `WithMatchKind(kind)`             | `Standard` (default), `LeftmostFirst` or `LeftmostLongest`.     |
| `WithASCIICaseInsensitive(bool)`  | Fold ASCII `A-Z` / `a-z` when matching.                         |
| `WithMaxMemory(bytes)`            | Fail with `ErrTooLarge` if the automaton would exceed the limit. |

### Errors

`New` and `NewBytes` return:

- `ErrEmptyPattern` – a pattern is empty (an empty pattern would match everywhere).
- `ErrInvalidMatchKind` – an unknown `MatchKind` was given.
- `ErrTooLarge` – the automaton exceeds the 32-bit state space or `WithMaxMemory`.

Use `errors.Is` to test for them. `MustNew` panics instead and is meant for
package-level variables. The overlapping methods panic if the automaton does
not use `Standard`, and `ReplaceAll`/`ReplaceAllString` panic if the number of
replacements differs from the number of patterns; both are programming errors.

## Performance

Search throughput on a 1 MiB random lowercase text (Intel Xeon @ 2.10 GHz,
Go 1.27.4, `go test -bench .`):

| Patterns | Standard   | LeftmostFirst | LeftmostLongest | Overlapping |
|---------:|-----------:|--------------:|----------------:|------------:|
| 10       | 464 MB/s   | 450 MB/s      | 459 MB/s        | 439 MB/s    |
| 1,000    | 285 MB/s   | 334 MB/s      | 328 MB/s        | 341 MB/s    |
| 10,000   | 181 MB/s   | 188 MB/s      | 200 MB/s        | 207 MB/s    |

With 100 patterns the new implementation searches at about 460 MB/s, versus
7.8 MB/s for the previous version of this package (about 59× faster).

Memory use is roughly `states × byte-classes × 4` bytes, where the number of
states is at most the total length of all patterns and byte classes is the
number of distinct bytes in the patterns plus one. `Automaton.MemoryUsage()`
reports the exact figure.

## How it works

1. **Trie.** Patterns are inserted into a trie. Under `LeftmostFirst`, a
   pattern whose prefix is an earlier pattern is dropped, because it can
   never win.
2. **Failure links.** A breadth-first pass links each state to its longest
   proper suffix that is also in the trie and propagates matches along those
   links. Under leftmost semantics, match states fail to a *dead* state, so a
   search never restarts after the leftmost match is known.
3. **Byte classes.** Bytes that never occur in a pattern are merged into one
   class (and, with case folding, `A`/`a` share a class), which shrinks the
   transition table.
4. **DFA.** Failure links are resolved ahead of time into a dense transition
   table. States are numbered so that the dead state and all match states
   come first; the search loop needs only a single comparison per byte to
   detect anything special.

## Migrating from the previous version

The old `MakeNode` / `Match` functions were replaced by the `Automaton` type:

```go
// Before
transitions, outputs, fails := ahocorasick.MakeNode(patterns)
results := ahocorasick.Match(text, transitions, outputs, fails)
for _, r := range results { fmt.Println(r.StartIdx, r.EndIdx, r.Result) }

// After
ac, err := ahocorasick.New(patterns)
if err != nil { /* ... */ }
for _, m := range ac.FindAllOverlappingString(text) {
	fmt.Println(m.Start, m.End-1, ac.Pattern(m.Pattern))
}
```

Note that `Match.End` is exclusive (the old `EndIdx` was inclusive) and that
all offsets are now byte offsets. The old version also had bugs that the
rewrite fixes, such as missing matches when one pattern was a prefix of
another and reporting only one pattern when several shared a suffix.

## Development

```sh
go test -race ./...                                  # unit + randomized tests
go test -run '^$' -fuzz FuzzAgainstNaive -fuzztime 1m # fuzzing
go test -run '^$' -bench .                           # benchmarks
golangci-lint run                                    # lint (v2)
govulncheck ./...                                    # vulnerability scan
```

See [CONTRIBUTING.md](CONTRIBUTING.md) and [SECURITY.md](SECURITY.md).

## License

[Apache License 2.0](LICENSE)
