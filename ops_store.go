package main

import (
	"context"
	"sort"
	"sync"
)

type OpsStore struct {
	mu    sync.RWMutex
	items map[string]OpsRecord
}

func newOpsStore(seed []OpsRecord) *OpsStore {
	s := &OpsStore{items: map[string]OpsRecord{}}
	for _, item := range seed {
		item = normalizeOpsRecord(item)
		s.items[item.ID] = item
	}
	return s
}
func (s *OpsStore) Get(ctx context.Context, id string) (OpsRecord, error) {
	select {
	case <-ctx.Done():
		return OpsRecord{}, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.items[id]
	if !ok {
		return OpsRecord{}, ErrOpsNotFound
	}
	// Return a deep copy so callers cannot observe in-flight writes performed
	// under the write lock (TouchLabels mutates the stored Labels map). The
	// stored record's map is shared by reference when copied by value, so a
	// plain struct copy would race against concurrent TouchLabels.
	return item.Clone(), nil
}
func (s *OpsStore) List(ctx context.Context) ([]OpsRecord, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]OpsRecord, 0, len(s.items))
	for _, item := range s.items {
		// Deep copy each record: see Get. Concurrent TouchLabels updates a
		// stored record's Labels map, so callers must not share it.
		out = append(out, item.Clone())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
func (s *OpsStore) Put(ctx context.Context, item OpsRecord) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[item.ID]; ok {
		return ErrOpsConflict
	}
	// Deep copy so the caller's Labels map is never aliased into the store;
	// later mutation by the caller would otherwise race with reads here.
	s.items[item.ID] = normalizeOpsRecord(item).Clone()
	return nil
}
func (s *OpsStore) Update(ctx context.Context, item OpsRecord, expected int) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.items[item.ID]
	if !ok {
		return ErrOpsNotFound
	}
	if expected > 0 && current.Revision != expected {
		return ErrOpsConflict
	}
	item.Revision = current.Revision + 1
	item.UpdatedAt = timeNowOps()
	s.items[item.ID] = item.Clone()
	return nil
}
func (s *OpsStore) Delete(ctx context.Context, id string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return ErrOpsNotFound
	}
	delete(s.items, id)
	return nil
}

// TouchLabels updates one label on a stored record. The write lock makes the
// in-place map mutation safe; Get/List return deep copies so concurrent readers
// never observe this write mid-flight.
func (s *OpsStore) TouchLabels(ctx context.Context, id, key, value string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[id]
	if !ok {
		return ErrOpsNotFound
	}
	if item.Labels == nil {
		item.Labels = map[string]string{}
	}
	item.Labels[key] = value
	s.items[id] = item
	return nil
}
func (s *OpsStore) Count() int { s.mu.RLock(); defer s.mu.RUnlock(); return len(s.items) }
