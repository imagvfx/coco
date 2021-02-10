package main

import "testing"

func TestIPMatcherMatch(t *testing.T) {
	cases := []struct {
		label     string
		matcher   string
		matches   []string
		unmatches []string
	}{
		{
			matcher:   "127.0.0.1",
			matches:   []string{"127.0.0.1"},
			unmatches: []string{"127.0.0.2", "127.0.0.3"},
		},
		{
			matcher:   "127.0.0.*",
			matches:   []string{"127.0.0.1", "127.0.0.2", "127.0.0.3"},
			unmatches: []string{"127.0.1.1", "127.0.0.256", "127.0.0.-1"},
		},
		{
			matcher:   "127.0.*.*",
			matches:   []string{"127.0.0.1", "127.0.0.2", "127.0.0.3", "127.0.1.1", "127.0.2.1"},
			unmatches: []string{"127.0.0.256", "127.0.256.0"},
		},
		{
			matcher:   "*.*.*.*",
			matches:   []string{"0.0.0.0", "255.255.255.255"},
			unmatches: []string{"127.0.0.256", "127.0.256.0"},
		},
		{
			matcher:   "127.0.[0-2].*",
			matches:   []string{"127.0.0.1", "127.0.0.2", "127.0.0.3", "127.0.1.1", "127.0.2.1"},
			unmatches: []string{"127.0.3.1", "127.0.4.1"},
		},
	}
	for _, c := range cases {
		f, err := ipMatcherFromString(c.matcher)
		if err != nil {
			t.Fatalf("matcher %v: ipFilterFromString: %v", c.matcher, err)
		}
		for _, m := range c.matches {
			match := f.Match(m)
			if !match {
				t.Fatalf("matcher %v: want match, got unmatch: %v", c.matcher, m)
			}
		}
		for _, um := range c.unmatches {
			match := f.Match(um)
			if match {
				t.Fatalf("matcher %v: want unmatch, got match: %v", c.matcher, um)
			}
		}
	}
}

func TestDomainMatcherMatch(t *testing.T) {
	cases := []struct {
		label     string
		matcher   string
		matches   []string
		unmatches []string
	}{
		{
			matcher:   "localhost",
			matches:   []string{"localhost"},
			unmatches: []string{"remotehost", "mylocalhost"},
		},
		{
			matcher:   "imagvfx.com",
			matches:   []string{"imagvfx.com"},
			unmatches: []string{"a.imagvfx.com", "b.imagvfx.com"},
		},
		{
			matcher:   "*.imagvfx.com",
			matches:   []string{"a.imagvfx.com", "b.imagvfx.com"},
			unmatches: []string{"imagvfx.com"},
		},
		{
			// possible, but not recommended
			matcher:   "*.*.*",
			matches:   []string{"a.imagvfx.com", "b.imagvfx.com"},
			unmatches: []string{"imagvfx.com"},
		},
	}
	for _, c := range cases {
		f, err := domainMatcherFromString(c.matcher)
		if err != nil {
			t.Fatalf("matcher %v: domainMatcherFromString: %v", c.matcher, err)
		}
		for _, m := range c.matches {
			match := f.Match(m)
			if !match {
				t.Fatalf("matcher %v: want match, got unmatch: %v", c.matcher, m)
			}
		}
		for _, um := range c.unmatches {
			match := f.Match(um)
			if match {
				t.Fatalf("matcher %v: want unmatch, got match: %v", c.matcher, um)
			}
		}
	}
}
