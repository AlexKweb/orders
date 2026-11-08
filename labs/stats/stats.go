// Package stats считает обработанные заказы.
package stats

import "sync"

// Stats — статистика по заказам. Создаётся через New.
type Stats struct {
	mu       sync.Mutex
	total    int64
	byStatus map[string]int64
}

// New создаёт пустую статистику.
func New() *Stats {
	return &Stats{byStatus: make(map[string]int64)}
}

// Record учитывает обработанный заказ.
func (s *Stats) Record(status string) {
	s.mu.Lock()
	s.byStatus[status]++
	s.mu.Unlock()

	s.total++
}

// Total возвращает число учтённых заказов.
func (s *Stats) Total() int64 {
	return s.total
}

// ByStatus возвращает число заказов в статусе status.
func (s *Stats) ByStatus(status string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.byStatus[status]
}
