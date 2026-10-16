package main

import (
	"fmt"
	"testing"
)

func TestCleanInput(t *testing.T) {
	cases := []struct {
		input    string
		expected []string
	}{
		{
			input:    "  hello  world  ",
			expected: []string{"hello", "world"},
		},
		{
			input:    "Charmander Bulbasaur PIKACHU",
			expected: []string{"charmander", "bulbasaur", "pikachu"},
		},
		{
			input:    "  empty first",
			expected: []string{"empty", "first"},
		},
		{
			input:    "empty last  ",
			expected: []string{"empty", "last"},
		},
		{
			input:    " - ",
			expected: []string{"-"},
		},
		{
			input:    " a, b, c ",
			expected: []string{"a,", "b,", "c"},
		},
		{
			input:    " ",
			expected: []string{""},
		},
	}

	for _, c := range cases {
		fmt.Printf("Input: '%s'\n", c.input)

		actual := CleanInput(c.input)
		// Check the length of the actual slice
		// if they don't match, use t.Errorf and continue to the next case
		if len(actual) != len(c.expected) {
			// error and continue here
			t.Errorf("the length of the slice between actual and expected doesn't match")
			// continue
		}
		for i := range actual {
			word := actual[i]
			expectedWord := c.expected[i]
			fmt.Printf("[*] Expected: '%s' - Got: '%s'\n", expectedWord, word)
			// Check each word in the slice
			// if they don't match, use t.Errorf to print an error message
			// and fail the test
			if word != expectedWord {
				// error and continue here
				t.Errorf("the word and the expectedWord didn't match")
				// continue
			}
		}
		fmt.Println("=======================")
	}
}
