package main

import "testing"

func TestEscaped(t *testing.T) {
	tests := []struct {
		input  string
		output string
	}{
		{
			input:  "Line1\nLine2\n",
			output: "Line1\nLine2\n",
		},
		{
			input: "Line0\n>From 123\nLine2\n",
			output: "Line0\n>>From 123\nLine2\n",
		},
		{
			input: "Line0\nFrom 123\nLine2\n",
			output: "Line0\n>From 123\nLine2\n",
		},
	}
	for _, v := range tests {
		s := Escaped(v.input)
		if s != v.output {
			t.Errorf("Expected %s, but got: %s", v.output,s)
		}
	}
}
