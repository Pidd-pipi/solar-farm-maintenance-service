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

// Update applies a status change to the maintenance task with the given id.
// It returns the resulting order on success (including a no-op where the status
// is unchanged), ErrWorkNotFound when no such task exists, or ErrWorkTransition
// when the move is not a legal step in workTransitions. Returning distinct
// errors means callers can never mistake a rejected change for an accepted one,
// so the order echoed back to the client always reflects what is persisted.
func (s *WorkStore) Update(id, status string) (WorkOrder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	order, ok := s.orders[id]
	if !ok {
		return WorkOrder{}, ErrWorkNotFound
	}
	if order.Status == status {
		return order, nil
	}
	if !workTransitions[order.Status][status] {
		return WorkOrder{}, ErrWorkTransition
	}
	order.Status = status
	s.orders[id] = order
	return order, nil
}
