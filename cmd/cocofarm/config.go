package main

import (
	"log"
	"sort"

	"github.com/imagvfx/coco"
	"github.com/pelletier/go-toml"
)

// Match go-toml.Tree keys to the same order with the file.
// Would be great it can be done with go-toml package, but didn't find the way.
func orderedKeys(t *toml.Tree) []string {
	type keyPos struct {
		Key  string
		Line int
		Col  int
	}
	keys := t.Keys()
	poses := make([]keyPos, len(keys))
	for i, k := range keys {
		subt := t.Get(k).(*toml.Tree)
		p := keyPos{
			Key:  k,
			Line: subt.Position().Line,
			Col:  subt.Position().Col,
		}
		poses[i] = p
	}
	sort.Slice(poses, func(i, j int) bool {
		if poses[i].Line < poses[j].Line {
			return true
		}
		if poses[i].Line > poses[j].Line {
			return false
		}
		if poses[i].Col < poses[j].Col {
			return true
		}
		return false
	})
	ordkeys := make([]string, len(poses))
	for i, p := range poses {
		ordkeys[i] = p.Key
	}
	return ordkeys
}

func loadWorkerGroupsFromConfig() ([]*coco.WorkerGroup, error) {
	config, err := toml.LoadFile("config/worker.toml")
	if err != nil {
		return nil, err
	}

	type WorkerGroupConfig struct {
		IPs          []string
		Domains      []string
		ServeTargets []string
	}
	wgrpCfgs := make(map[string]*WorkerGroupConfig)
	err = config.Unmarshal(&wgrpCfgs)
	if err != nil {
		return nil, err
	}

	wgrps := make([]*coco.WorkerGroup, 0)
	for _, grp := range orderedKeys(config) {
		g := &coco.WorkerGroup{}
		g.Name = grp
		cfg := wgrpCfgs[grp]
		for _, w := range cfg.IPs {
			m, err := coco.IPMatcherFromString(w)
			if err != nil {
				log.Fatalf("%v", err)
			}
			g.Matchers = append(g.Matchers, m)
		}
		for _, w := range cfg.Domains {
			m, err := coco.DomainMatcherFromString(w)
			if err != nil {
				log.Fatalf("%v", err)
			}
			g.Matchers = append(g.Matchers, m)
		}
		g.ServeTargets = cfg.ServeTargets
		wgrps = append(wgrps, g)
	}
	return wgrps, nil
}
