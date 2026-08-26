package main

import "sync"

type WorkStore struct {
	mu     sync.RWMutex
	orders map[string]WorkOrder
}

func newWorkStore() *WorkStore {
	return &WorkStore{orders: map[string]WorkOrder{
		"WO-301": {ID: "WO-301", Array: "A-03", Issue: "inverter temperature", Status: "open"},
		"WO-302": {ID: "WO-302", Array: "A-04", Issue: "panel inspection", Status: "scheduled"},
	}}
}

func (s *WorkStore) List() []WorkOrder {
	s.mu.RLock()
	defer s.mu.RUnlock()
	orders := make([]WorkOrder, 0, len(s.orders))
	for _, order := range s.orders {
		orders = append(orders, order)
	}
	return orders
}

func (s *WorkStore) Update(id, status string) (WorkOrder, bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	order, ok := s.orders[id]
	if !ok {
		return WorkOrder{}, false, false
	}
	if !workTransitions[order.Status][status] {
		return order, true, false
	}
	updated := order
	updated.Status = status
	s.orders[id] = order
	return updated, true, true
}
