package main

import (
	"strings"
	"sync"
)

type competitionsLookup func(wcaID string) ([]PersonCompetition, error)

func MarkNewcomers(s *Standings, compID string, lookup competitionsLookup) error {
	if len(s.StartDate) < 4 {
		return nil
	}
	year := s.StartDate[:4]

	var candidates []*Competitor
	for _, c := range s.Competitors {
		if !c.Newcomer && strings.HasPrefix(c.WCAID, year) {
			candidates = append(candidates, c)
		}
	}

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
		sem      = make(chan struct{}, 8)
	)
	for _, c := range candidates {
		wg.Add(1)
		go func(c *Competitor) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			comps, err := lookup(c.WCAID)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				return
			}
			c.Newcomer = !competedBefore(comps, compID, s.StartDate)
		}(c)
	}
	wg.Wait()
	return firstErr
}

func competedBefore(comps []PersonCompetition, compID, startDate string) bool {
	for _, comp := range comps {
		if comp.ID != compID && comp.StartDate < startDate {
			return true
		}
	}
	return false
}
