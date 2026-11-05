package main

import "sort"

type Competitor struct {
	ID          int
	Name        string
	WCAID       string
	Newcomer    bool
	Adjusted    map[string]int
	SumOfRanks  int
	roundRanks  map[string]map[int]int
	lastRoundIn map[string]int
}

func (c *Competitor) HasPodium() bool {
	for _, r := range c.Adjusted {
		if r <= 3 {
			return true
		}
	}
	return false
}

type Standings struct {
	CompetitionName string
	StartDate       string
	EventIDs        []string
	Competitors     []*Competitor
}

func Compute(w *WCIF) *Standings {
	s := &Standings{CompetitionName: w.Name, StartDate: w.Schedule.StartDate}
	byID := map[int]*Competitor{}

	for _, p := range w.Persons {
		if !p.isCompetitor() {
			continue
		}
		c := &Competitor{
			ID:          *p.RegistrantID,
			Name:        p.Name,
			WCAID:       p.WCAID,
			Newcomer:    p.WCAID == "",
			Adjusted:    map[string]int{},
			roundRanks:  map[string]map[int]int{},
			lastRoundIn: map[string]int{},
		}
		byID[c.ID] = c
		s.Competitors = append(s.Competitors, c)
	}

	competed := map[int]bool{}
	for _, ev := range w.Events {
		for _, round := range ev.Rounds {
			for _, res := range round.Results {
				competed[res.PersonID] = true
			}
		}
	}
	var competitors []*Competitor
	for _, c := range s.Competitors {
		if competed[c.ID] {
			competitors = append(competitors, c)
		} else {
			delete(byID, c.ID)
		}
	}
	s.Competitors = competitors

	lastRank := map[string][]int{}
	for _, ev := range w.Events {
		if len(ev.Rounds) == 0 {
			continue
		}
		s.EventIDs = append(s.EventIDs, ev.ID)
		perRound := make([]int, len(ev.Rounds))

		for ri, round := range ev.Rounds {
			roundNo := ri + 1
			last := 4
			for _, res := range round.Results {
				if res.Ranking == nil {
					continue
				}
				rank := *res.Ranking
				if rank <= 3 && res.Best < 0 && res.Average <= 0 {
					rank = 4
				}
				last = max(last, rank)

				c, ok := byID[res.PersonID]
				if !ok {
					continue
				}
				if c.roundRanks[ev.ID] == nil {
					c.roundRanks[ev.ID] = map[int]int{}
				}
				c.roundRanks[ev.ID][roundNo] = rank
				c.lastRoundIn[ev.ID] = max(c.lastRoundIn[ev.ID], roundNo)
			}
			perRound[ri] = last
		}
		lastRank[ev.ID] = perRound
	}

	for _, c := range s.Competitors {
		for _, evID := range s.EventIDs {
			rounds := lastRank[evID]
			reached := c.lastRoundIn[evID]
			adj := 0
			if reached > 0 {
				adj = c.roundRanks[evID][reached]
			}
			for r := reached + 1; r <= len(rounds); r++ {
				adj += rounds[r-1] + 1
			}
			c.Adjusted[evID] = adj
			c.SumOfRanks += adj
		}
	}

	return s
}

func (s *Standings) HasEvent(id string) bool {
	for _, e := range s.EventIDs {
		if e == id {
			return true
		}
	}
	return false
}

func (s *Standings) WithoutPodium() []*Competitor {
	var out []*Competitor
	for _, c := range s.Competitors {
		if !c.HasPodium() {
			out = append(out, c)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].SumOfRanks < out[j].SumOfRanks })
	return out
}

func (s *Standings) Newcomers(eventID string) []*Competitor {
	var out []*Competitor
	for _, c := range s.Competitors {
		if c.Newcomer {
			out = append(out, c)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Adjusted[eventID] < out[j].Adjusted[eventID] })
	return out
}

func Positions(list []*Competitor, key func(*Competitor) int) []int {
	pos := make([]int, len(list))
	for i := range list {
		if i > 0 && key(list[i]) == key(list[i-1]) {
			pos[i] = pos[i-1]
		} else {
			pos[i] = i + 1
		}
	}
	return pos
}
