// ahocorasick.go : Aho-Corasick string matching algorithm.
// multi string matcing

package ahocorasick

import (
	"strconv"
)

// MatchResult : match result
type MatchResult struct {
	StartIdx int
	EndIdx   int
	Result   string
}

//FAIL : fail
var FAIL = -1

// Remove : slice remove value
func Remove(s []int, i int) []int {
	s[len(s)-1], s[i] = s[i], s[len(s)-1]
	return s[:len(s)-1]
}

// MakeNode : aho-corasick node create
func MakeNode(pattern []string) (map[[2]string]int, map[int][]string, map[int]int) {
	//pattern node set
	transitions := make(map[[2]string]int)
	//pattern result node set
	outputs := make(map[int][]string)
	//match fail return node
	fails := make(map[int]int)

	newState := 0
	for _, v := range pattern {
		var State int
		var J int
		State = 0
		// make root node
		for j, char := range v {
			value := [2]string{strconv.Itoa(State), string(char)}
			k, l := transitions[value]
			if l == false {
				J = j
				break
			} else {
				State = k
			}
		}
		// make sub node
		for _, char := range v[J:] {
			newState = newState + 1
			value := [2]string{strconv.Itoa(State), string(char)}
			transitions[value] = newState
			State = newState
		}
		// result save
		outputs[State] = []string{v}

	}

	// make fail set
	queue := []int{}
	for k, v := range transitions {
		if k[0] == "0" && v != 0 {
			queue = append(queue, v)
			fails[v] = 0
		}
	}
	var r int

	// make fail set && output set
	for {
		r = queue[0]
		queue = Remove(queue, 0)
		for k, v := range transitions {
			if k[0] == strconv.Itoa(r) {
				queue = append(queue, v)
				value, _ := strconv.Atoi(k[0])
				State := fails[value]

				for {
					value := [2]string{strconv.Itoa(State), k[1]}
					res, l := transitions[value]
					if l == false && State != 0 {
						if res == 0 {
							res = -1
						}
					}

					if res != FAIL {
						break
					}
					State = fails[State]

				}
				value2 := [2]string{strconv.Itoa(State), k[1]}
				failure, l := transitions[value2]
				if l == false {
					if State == 0 {
						failure = 0
					} else {
						failure = -1
					}
				}
				fails[v] = failure

				data, _ := outputs[v]
				if data == nil {
					outputs[v] = data

				}
				data2, _ := outputs[failure]
				if data2 != nil {
					outputs[v] = append(outputs[v], data2[0])

				}
			}
		}

		if len(queue) == 0 {
			break
		}
	}

	return transitions, outputs, fails
}

//Match : Aho-corasick match
func Match(word string, transitions map[[2]string]int, outputs map[int][]string, fails map[int]int) []MatchResult {
	state := 0
	results := []MatchResult{}
	for k, v := range word {
		//pattern match
		//return fail node if it can't go next node
		var value [2]string
		var res int
		var l bool
		for {
			value = [2]string{strconv.Itoa(state), string(v)}
			res, l = transitions[value]
			if l == false && state != 0 {
				if res == 0 {
					res = -1
				}
			}
			if res != FAIL {
				state = res
				break
			}
			state = fails[state]
		}
		res2, _ := outputs[state]
		//find output node. output node and index position input results
		if len(res2) != 0 {
			for _, v2 := range res2 {
				pos := k - len(v2) + 1
				posend := k
				results = append(results, MatchResult{pos, posend, v2})
			}
		}
	}
	return results
}
