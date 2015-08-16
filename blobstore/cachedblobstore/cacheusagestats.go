package cachedblobstore

import (
	"sort"
	"sync"
	"time"

	fl "github.com/nyaxt/otaru/flags"
)

type usageStatEntry struct {
	lastUsed   time.Time
	readCount  int
	writeCount int
}

type CacheUsageStats struct {
	mu      sync.Mutex
	entries map[string]usageStatEntry
}

func NewCacheUsageStats() *CacheUsageStats {
	return &CacheUsageStats{entries: make(map[string]usageStatEntry)}
}

func (s *CacheUsageStats) ObserveOpen(blobpath string, flags int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	e := s.entries[blobpath]
	if fl.IsReadAllowed(flags) {
		e.readCount++
	}
	if fl.IsWriteAllowed(flags) {
		e.writeCount++
	}
	e.lastUsed = time.Now()

	s.entries[blobpath] = e
}

func (s *CacheUsageStats) ObserveRemoveBlob(blobpath string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.entries, blobpath)
}

func (s *CacheUsageStats) ImportBlobList(blobpaths []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var tzero time.Time

	for _, bp := range blobpaths {
		s.entries[bp] = usageStatEntry{lastUsed: tzero}
	}
}

type lastUsedSorter struct {
	bps     []string
	entries map[string]usageStatEntry
}

func (s lastUsedSorter) Len() int { return len(s.bps) }

func (s lastUsedSorter) Swap(i, j int) {
	s.bps[i], s.bps[j] = s.bps[j], s.bps[i]
}

func (s lastUsedSorter) Less(i, j int) bool {
	a := s.entries[s.bps[i]].lastUsed
	b := s.entries[s.bps[j]].lastUsed

	return a.Before(b)
}

func (s *CacheUsageStats) FindLeastUsed() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	bps := make([]string, 0, len(s.entries))
	for bp, _ := range s.entries {
		bps = append(bps, bp)
	}
	sort.Sort(lastUsedSorter{bps, s.entries})

	return bps
}

type CacheUsageStatsView struct {
	NumEntries int `json:"num_entries"`
}

func (s *CacheUsageStats) View() CacheUsageStatsView {
	s.mu.Lock()
	defer s.mu.Unlock()

	return CacheUsageStatsView{
		NumEntries: len(s.entries),
	}
}
