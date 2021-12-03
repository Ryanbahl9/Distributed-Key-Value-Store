package view_pkg

import "sync"

type View struct {
	*sync.Mutex
	Nodes map[string]struct{} `json:"nodes"`
}

func (v View) Contains(node string) bool {
	_, exists := v.Nodes[node]
	return exists
}
