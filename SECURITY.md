# Security Policy

## Supported versions

Only the latest release receives security fixes.

## Reporting a vulnerability

Please **do not** open a public issue for security problems. Report them
privately through
[GitHub Security Advisories](https://github.com/spro1/aho-corasick/security/advisories/new).
You should receive a response within a few days.

## Notes for users

- The library has no dependencies outside the Go standard library. CI runs
  `govulncheck` on every change and weekly.
- Security fixes in the Go standard library are delivered by your Go
  toolchain; build with the latest patch release.
- When building automatons from untrusted patterns, use `WithMaxMemory` to
  bound memory use.
