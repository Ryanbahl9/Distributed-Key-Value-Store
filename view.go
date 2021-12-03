package main

import (
	"errors"
	"sync"
)

var ErrNodeNotFound = errors.New("node not found")

type View struct {
	sync.Mutex
	Nodes map[string]struct{} `json:"nodes"`
}

func NewView() *View {
	return &View{
		Nodes: make(map[string]struct{}),
	}
}

func (v *View) Contains(node string) bool {
	_, exists := v.Nodes[node]
	return exists
}

func (v *View) GetViewAsSlice() []string {
	v.Lock()
	defer v.Unlock()

	// Build View array from the keys of the local vector clock
	viewArr := make([]string, len(v.Nodes))
	i := 0
	for key := range v.Nodes {
		viewArr[i] = key
		i++
	}
	return viewArr
}

func (v *View) PutView(node string) bool {
	v.Lock()
	defer v.Unlock()

	// check if node exists in view
	_, exists := v.Nodes[node]

	// Replica doesn't exist in view. Add it.
	v.Nodes[node] = struct{}{}

	return exists
}

func (v *View) DeleteView(node string) bool {
	v.Lock()
	defer v.Unlock()

	// check if node exists in view
	_, exists := v.Nodes[node]
	if !exists {
		return false
	}

	// Replica exists in the view, delete it from the view
	delete(v.Nodes, node)

	return true
}
