package main

import (
	"fmt"
	"strconv"
	"strings"
)

// IPFilter matches to an ip or more.
type IPFilter []IPPartMatcher

type IPPartMatcher interface {
	Match(int) bool
}

type IPPartAllMatcher struct{}

func (m IPPartAllMatcher) Match(n int) bool {
	return true
}

type IPPartSingleMatcher struct {
	n int
}

func (m IPPartSingleMatcher) Match(n int) bool {
	return n == m.n
}

type IPPartRangeMatcher struct {
	start, end int
}

func (m IPPartRangeMatcher) Match(n int) bool {
	if m.start <= n && n <= m.end {
		return true
	}
	return false
}

// I don't know how to handle IPv6 yet.
// So, for now I will focus on IPv4.
func ipFilterFromString(s string) (IPFilter, error) {
	ipRange := func(p string) (int, int) {
		if !strings.HasPrefix(p, "[") || !strings.HasSuffix(p, "]") {
			return -1, -1
		}
		rng := strings.Split(p[1:len(p)-1], "-")
		if len(rng) != 2 {
			return -1, -1
		}
		s, err := strconv.Atoi(rng[0])
		if err != nil {
			return -1, -1
		}
		if s >= 256 {
			return -1, -1
		}
		e, err := strconv.Atoi(rng[1])
		if err != nil {
			return -1, -1
		}
		if e >= 256 {
			return -1, 01
		}
		return s, e
	}
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return nil, fmt.Errorf("ip does not consists of 4 parts: %v", s)
	}
	filter := make(IPFilter, 4)
	for i, p := range parts {
		if p == "*" {
			filter[i] = IPPartAllMatcher{}
			continue
		}
		n, err := strconv.Atoi(p)
		if err == nil {
			filter[i] = IPPartSingleMatcher{n}
			continue
		}
		s, e := ipRange(p)
		if s != -1 {
			filter[i] = IPPartRangeMatcher{s, e}
			continue
		}
		return nil, fmt.Errorf("unknown formatting for ip part: %v", p)
	}
	return filter, nil
}

func (f IPFilter) Match(ip string) bool {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return false
		}
		if n >= 256 {
			return false
		}
		if !f[i].Match(n) {
			return false
		}
	}
	return true
}
