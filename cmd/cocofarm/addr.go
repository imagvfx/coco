package main

import (
	"fmt"
	"strconv"
	"strings"
)

// AddressMatcher matches to a range of ips or domains.
// It is used to check whether a worker belongs to a worker group.
type AddressMatcher interface {
	Match(string) bool
}

// IPMatcher matches to an ip or more.
type IPMatcher []IPPartMatcher

func (m IPMatcher) Match(ip string) bool {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return false
		}
		if !m[i].Match(n) {
			return false
		}
	}
	return true
}

type IPPartMatcher interface {
	Match(int) bool
}

type IPPartAllMatcher struct{}

func (m IPPartAllMatcher) Match(n int) bool {
	if n < 0 || n >= 256 {
		return false
	}
	return true
}

type IPPartSingleMatcher struct {
	n int
}

func (m IPPartSingleMatcher) Match(n int) bool {
	if n < 0 || n >= 256 {
		return false
	}
	return n == m.n
}

type IPPartRangeMatcher struct {
	start, end int
}

func (m IPPartRangeMatcher) Match(n int) bool {
	if n < 0 || n >= 256 {
		return false
	}
	if m.start <= n && n <= m.end {
		return true
	}
	return false
}

// I don't know how to handle IPv6 yet.
// So, for now I will focus on IPv4.
func ipMatcherFromString(s string) (IPMatcher, error) {
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
		if s < 0 || s >= 256 {
			return -1, -1
		}
		e, err := strconv.Atoi(rng[1])
		if err != nil {
			return -1, -1
		}
		if e < 0 || e >= 256 {
			return -1, 01
		}
		return s, e
	}
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return nil, fmt.Errorf("ip does not consists of 4 parts: %v", s)
	}
	matcher := make(IPMatcher, 4)
	for i, p := range parts {
		if p == "*" {
			matcher[i] = IPPartAllMatcher{}
			continue
		}
		n, err := strconv.Atoi(p)
		if err == nil {
			if n < 0 || n >= 256 {
				return nil, fmt.Errorf("an ip part should be 0-255 when it is a number")
			}
			matcher[i] = IPPartSingleMatcher{n}
			continue
		}
		s, e := ipRange(p)
		if s != -1 {
			matcher[i] = IPPartRangeMatcher{s, e}
			continue
		}
		return nil, fmt.Errorf("unknown formatting for ip part: %v", p)
	}
	return matcher, nil
}

// DomainMatcher matches to a range of domains.
type DomainMatcher []DomainPartMatcher

func domainMatcherFromString(s string) (DomainMatcher, error) {
	if s == "" {
		return nil, fmt.Errorf("cannot create a domain matcher from empty string")
	}
	parts := strings.Split(s, ".")
	m := make(DomainMatcher, len(parts))
	for i, p := range parts {
		if p == "*" {
			m[i] = DomainPartAllMatcher{}
		} else {
			m[i] = DomainPartSingleMatcher{p}
		}
	}
	return m, nil
}

func (m DomainMatcher) Match(s string) bool {
	if len(m) == 0 {
		return false
	}
	if s == "" {
		return false
	}
	parts := strings.Split(s, ".")
	if len(m) != len(parts) {
		return false
	}
	for i, p := range parts {
		if !m[i].Match(p) {
			return false
		}
	}
	return true
}

type DomainPartMatcher interface {
	Match(string) bool
}

type DomainPartAllMatcher struct{}

func (m DomainPartAllMatcher) Match(s string) bool {
	return true
}

type DomainPartSingleMatcher struct {
	s string
}

func (m DomainPartSingleMatcher) Match(s string) bool {
	return s == m.s
}
