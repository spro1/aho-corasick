# Contributing

Thanks for your interest in improving aho-corasick!

1. Open an issue to discuss larger changes before starting work.
2. Keep the library free of dependencies outside the standard library.
3. Add tests for every change. Behavioural changes must keep the randomized
   and fuzz tests (which compare against a naive reference implementation)
   passing.
4. Before sending a pull request, run:

   ```sh
   gofmt -l .
   go vet ./...
   go test -race ./...
   go test -run '^$' -fuzz FuzzAgainstNaive -fuzztime 1m
   golangci-lint run
   ```

5. For performance changes, include `benchstat` output for `go test -bench .`
   before and after.

By contributing you agree that your contributions are licensed under the
Apache License 2.0.
