package main

import "testing"

func TestIPFilterMatch(t *testing.T) {
	cases := []struct {
		label     string
		filter    string
		matches   []string
		unmatches []string
	}{
		{
			filter:    "127.0.0.1",
			matches:   []string{"127.0.0.1"},
			unmatches: []string{"127.0.0.2", "127.0.0.3"},
		},
		{
			filter:    "127.0.0.*",
			matches:   []string{"127.0.0.1", "127.0.0.2", "127.0.0.3"},
			unmatches: []string{"127.0.1.1", "127.0.0.256"},
		},
		{
			filter:    "127.0.*.*",
			matches:   []string{"127.0.0.1", "127.0.0.2", "127.0.0.3", "127.0.1.1", "127.0.2.1"},
			unmatches: []string{"127.0.0.256", "127.0.256.0"},
		},
		{
			filter:    "*.*.*.*",
			matches:   []string{"0.0.0.0", "255.255.255.255"},
			unmatches: []string{"127.0.0.256", "127.0.256.0"},
		},
		{
			filter:    "127.0.[0-2].*",
			matches:   []string{"127.0.0.1", "127.0.0.2", "127.0.0.3", "127.0.1.1", "127.0.2.1"},
			unmatches: []string{"127.0.3.1", "127.0.4.1"},
		},
	}
	for _, c := range cases {
		f, err := ipFilterFromString(c.filter)
		if err != nil {
			t.Fatalf("filter %v: ipFilterFromString: %v", c.filter, err)
		}
		for _, m := range c.matches {
			match := f.Match(m)
			if !match {
				t.Fatalf("filter %v: want match, got unmatch: %v", c.filter, m)
			}
		}
		for _, um := range c.unmatches {
			match := f.Match(um)
			if match {
				t.Fatalf("filter %v: want unmatch, got match: %v", c.filter, um)
			}
		}
	}
}
