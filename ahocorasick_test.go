package ahocorasick

import (
	"fmt"
	"testing"
)

func TestMatch(t *testing.T) {
	fattern := []string{"hi", "his", "shis", "she", "he", "her"}
	word := "he likes her, but she doesn't like him"
	transitions, outputs, fails := MakeNode(fattern)
	result := Match(word, transitions, outputs, fails)

	fmt.Println(result)
}
