# Changelog

All notable changes to this project are documented in this file. The format is
based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the
project follows [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- `Automaton` type compiled with `New`, `NewBytes` or `MustNew`.
- Match semantics: `Standard`, `LeftmostFirst`, `LeftmostLongest`.
- `WithASCIICaseInsensitive` and `WithMaxMemory` options.
- Search methods for `[]byte` and `string`: `IsMatch`, `Find`, `FindAll`,
  `FindAllOverlapping`, and the iterators `Iter` / `IterOverlapping`.
- Replacement: `ReplaceAll`, `ReplaceAllString`, `ReplaceAllFunc`,
  `ReplaceAllStringFunc`.
- Go module (`go.mod`), CI (tests on Linux/macOS/Windows, race detector,
  fuzzing, golangci-lint, govulncheck) and Dependabot.

### Changed

- Complete rewrite as a dense DFA over byte classes, about 59× faster than
  the previous implementation.
- Offsets are byte offsets and `Match.End` is exclusive.

### Removed

- `MakeNode`, `Match`, `Remove`, `MatchResult` and `FAIL`.

### Fixed

- Matches were missed when a pattern was a prefix of a previously added
  pattern (e.g. `["hers", "he"]` never matched `"he"`).
- Only the first of several suffix matches was reported
  (e.g. `["abc", "bc", "c"]` on `"abc"` missed `"c"`).
