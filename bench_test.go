package ahocorasick

import (
	"math/rand/v2"
	"strconv"
	"strings"
	"testing"
)

func benchPatterns(n int) []string {
	r := rand.New(rand.NewPCG(7, 7))
	ps := make([]string, n)
	for i := range ps {
		ps[i] = randString(r, "abcdefghijklmnopqrstuvwxyz", 4, 12)
	}
	return ps
}

func benchHaystack() string {
	r := rand.New(rand.NewPCG(8, 8))
	return randString(r, "abcdefghijklmnopqrstuvwxyz      ", 1<<20, 1<<20)
}

func BenchmarkBuild(b *testing.B) {
	for _, n := range []int{100, 10_000} {
		ps := benchPatterns(n)
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := New(ps); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkSearch(b *testing.B) {
	h := benchHaystack()
	for _, n := range []int{10, 1_000, 10_000} {
		ps := benchPatterns(n)
		for _, k := range allKinds {
			a := MustNew(ps, WithMatchKind(k))
			b.Run(strconv.Itoa(n)+"/"+k.String(), func(b *testing.B) {
				b.SetBytes(int64(len(h)))
				for b.Loop() {
					for range a.IterString(h) {
					}
				}
			})
		}
		a := MustNew(ps)
		b.Run(strconv.Itoa(n)+"/Overlapping", func(b *testing.B) {
			b.SetBytes(int64(len(h)))
			for b.Loop() {
				for range a.IterOverlappingString(h) {
				}
			}
		})
		b.Run(strconv.Itoa(n)+"/IsMatch-none", func(b *testing.B) {
			none := strings.Repeat(" ", len(h))
			b.SetBytes(int64(len(none)))
			for b.Loop() {
				a.IsMatchString(none)
			}
		})
	}
}
