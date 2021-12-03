package ring_pkg

import (
	"errors"
	"hash/crc32"
	"sort"
	"strconv"
	"sync"
)

const MinReplicasPerShard = 2
const NumVirtShardsPerShard = 10

var ErrNotEnoughNodes = errors.New("not enough nodes to provide fault tolerance with requested shard count")

type Ring struct {
	*sync.Mutex
	VirtShards VirtShards
	Shards     Shards
}

type Shard struct {
	Replicas map[string]struct{}
}

type Shards []Shard

type VirtShard struct {
	HashId  uint32
	ShardId int
}

type VirtShards []VirtShard

func NewRing(numShards int, nodes map[string]struct{}) Ring {
	newRing := Ring{
		VirtShards: make(VirtShards, numShards*NumVirtShardsPerShard),
		Shards:     make(Shards, numShards),
	}

	for i := 0; i < numShards; i++ {
		// Populate the ring with empty shards
		newRing.Shards[i] = Shard{
			Replicas: make(map[string]struct{}),
		}

		// make virtual shards on the ring for each real shard
		for j := 0; j < NumVirtShardsPerShard; j++ {

			// Note, name can be anything as long as its unique from all other virtual shards
			name := "Shard_" + strconv.Itoa(i) + "_VirtShard_" + strconv.Itoa(j)
			newRing.VirtShards[j+i*NumVirtShardsPerShard] = VirtShard{
				HashId:  crc32.ChecksumIEEE([]byte(name)),
				ShardId: i,
			}
		}
	}

	// sort virt shards so we can use sort.Search() function later
	sort.Sort(newRing.VirtShards)

	// put the nodes into the shards as evenly as we can
	i := 0
	for key, val := range nodes {
		newRing.Shards[i].Replicas[key] = val

		i = numShards % (i + 1)
	}

	// return the new ring
	return newRing
}

// Given a piece of data, this function returns the id of the shard it should go to
func (r Ring) GetShardId(id string) int {
	i := r.search(id)
	if i >= len(r.VirtShards) {
		i = 0
	}

	return r.VirtShards[i].ShardId
}

// Helper function for GetShard
func (r Ring) search(id string) int {
	searchfn := func(i int) bool {
		return r.VirtShards[i].HashId >= crc32.ChecksumIEEE([]byte(id))
	}

	return sort.Search(len(r.VirtShards), searchfn)
}

// Finds the id of the shard a node belongs to, If node not found return -1
func (r Ring) GetShardIdFromNode(node string) int {
	for i := 0; i < len(r.Shards); i++ {
		_, exists := r.Shards[i].Replicas[node]
		if exists {
			return i
		}
	}
	return -1
}

// Returns a new ring based on the given paramiters
func (r Ring) Reshard(numShards int, nodes map[string]struct{}) (Ring, error) {
	if numShards == len(r.Shards) {
		return r, nil
	}
	if (len(nodes) / numShards) < MinReplicasPerShard {
		return r, ErrNotEnoughNodes
	}

	newRing := NewRing(numShards, nodes)

	return newRing, nil
}

// These are hear so we can use the built in sort function on VirtShards structs
func (n VirtShards) Len() int           { return len(n) }
func (n VirtShards) Less(i, j int) bool { return n[i].HashId < n[j].HashId }
func (n VirtShards) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }
