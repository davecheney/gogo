package gogo

// build statistics

import (
	"sync"
	"time"
)

// statistics records the various Durations
type statistics struct {
	sync.Mutex
	stats map[string]time.Duration
}

func (s *statistics) Record(name string, d time.Duration) {
	s.Lock()
	defer s.Unlock()
	if s.stats == nil {
		s.stats = make(map[string]time.Duration)
	}
	acc := s.stats[name]
	acc += d
	s.stats[name] = acc
}

func (s *statistics) Total() time.Duration {
	s.Lock()
	defer s.Unlock()
	var d time.Duration
	for _, v := range s.stats {
		d += v
	}
	return d
}
