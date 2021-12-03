package view_pkg

import "sync"

type View struct {
	*sync.Mutex
	Nodes        map[string]struct{} `json:"nodes"`
	LocalAddress string              `json:"localAddress"`
}
