package main

import "sync"

// MutexedBool is a simple boolean, protected against race conditions with a sync.Mutex.
type MutexedBool struct {
	mux   sync.Mutex
	state bool
}

// NewMutexedBool creates a new MutexedBool.
func NewMutexedBool(initialState bool) *MutexedBool {
	a := MutexedBool{}
	a.state = initialState
	return &a
}

// Get returns the bool.
func (m *MutexedBool) Get() bool {
	m.mux.Lock()
	defer m.mux.Unlock()
	return m.state
}

// Set sets the bool.
func (m *MutexedBool) Set(state bool) {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.state = state
}
