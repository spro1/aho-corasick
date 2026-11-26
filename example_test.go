package ahocorasick_test

import (
	"fmt"
	"log"

	ahocorasick "github.com/spro1/aho-corasick"
)

func Example() {
	ac, err := ahocorasick.New([]string{"he", "she", "his", "hers"})
	if err != nil {
		log.Fatal(err)
	}
	for m := range ac.IterOverlappingString("ushers") {
		fmt.Printf("%q at [%d:%d]\n", ac.Pattern(m.Pattern), m.Start, m.End)
	}
	// Output:
	// "she" at [1:4]
	// "he" at [2:4]
	// "hers" at [2:6]
}

func Example_matchKinds() {
	patterns := []string{"Sam", "Samwise"}
	for _, kind := range []ahocorasick.MatchKind{
		ahocorasick.Standard, ahocorasick.LeftmostFirst, ahocorasick.LeftmostLongest,
	} {
		ac := ahocorasick.MustNew(patterns, ahocorasick.WithMatchKind(kind))
		m, _ := ac.FindString("Samwise")
		fmt.Printf("%-15s %s\n", kind, ac.Pattern(m.Pattern))
	}
	// Output:
	// Standard        Sam
	// LeftmostFirst   Sam
	// LeftmostLongest Samwise
}

func ExampleAutomaton_FindAllString() {
	ac := ahocorasick.MustNew(
		[]string{"error", "warn"},
		ahocorasick.WithASCIICaseInsensitive(true),
	)
	line := "WARN: disk almost full; ERROR: write failed"
	for _, m := range ac.FindAllString(line) {
		fmt.Println(line[m.Start:m.End])
	}
	// Output:
	// WARN
	// ERROR
}

func ExampleAutomaton_IsMatchString() {
	banned := ahocorasick.MustNew([]string{"password", "secret"})
	fmt.Println(banned.IsMatchString("my password is hunter2"))
	fmt.Println(banned.IsMatchString("hello world"))
	// Output:
	// true
	// false
}

func ExampleAutomaton_ReplaceAllString() {
	ac := ahocorasick.MustNew(
		[]string{"apple", "banana"},
		ahocorasick.WithMatchKind(ahocorasick.LeftmostLongest),
	)
	fmt.Println(ac.ReplaceAllString("apple and banana", []string{"사과", "바나나"}))
	// Output:
	// 사과 and 바나나
}

func ExampleAutomaton_ReplaceAllStringFunc() {
	ac := ahocorasick.MustNew([]string{"alice@example.com", "bob@example.com"})
	masked := ac.ReplaceAllStringFunc("mail alice@example.com now", func(ahocorasick.Match) string {
		return "<redacted>"
	})
	fmt.Println(masked)
	// Output:
	// mail <redacted> now
}
