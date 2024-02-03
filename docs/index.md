# CSE138_Assignment4
CSE138 (Distributed Systems) Assignment 4 By Ryan Bahl and Chris Sterza

## Acknowledgments 
N/A

## Citations 
1. Consistent hashing, a guide & Go library: [Link](https://medium.com/@sent0hil/consistent-hashing-a-guide-go-implementation-fe3421ac3e8f)
    - This website was consulted heavily for the structure of our structs in the sharding.go file as well as a few functions that act on those structs. 
      - Our Ring and VirtShard structs are based on their Ring and Node structs.
      - Our GetShardId() and search() functions are based on their Get() function.
      - Part of our NewRing() function is based on their AddNode() function
    - Consulting this site, we learned how to implement the more abstract concepts of the ring method of consistent hashing learned in class, using the tools provided to us in the Golang language.
2. Distributed Systems Notes: [Link](https://github.com/ChrisWhealy/DistributedSystemNotes)
    - This website was consulted lightly as a general refresher of the concepts we learned in class. 
      - Although we didn't use anything in these notes directly, the concepts in them helped us reason about points of failure such as downed nodes.
    - We didn't learn anything new in these notes, but they were a good reminder for some of the points we learned in class.

## Team Contributions
1. Ryan Bahl
    - Sharding functions and routes
    - Broadcasting Functions 
    - Initiation Logic
    - Sending and Parsing related code
2. Chris Sterza
    - Vector Clock Comparison Functionality.
    - Value retrieval.
    - View list management.
    - Pseudocode when designing system
## Mechanism Description
#### Tracking Causal Dependencies
  - Causal Dependencies were tracked using vector clocks which consisted of a map of socket addresses and integer clock values. Vector clocks were only incremented on replica puts and deletes (aka only message sends) and were passed via the message's ```'causal-metadata'``` field.
  - We used the causal broadcast algorithm learned in class to compare the vector clocks. Namely we had a function that checked if the sender's vector clock was only one larger than the local clock in the sender's position and equal to or less than the local clock for every other position. If it was, the function returned true, otherwise false. Other functions used this return value to determine the next course of action.
#### Detecting Down Replicas
  - If at any time a replica took more than one second to respond to any http request, that replica would be removed from the sender's view and the sender would broadcast a DELETE message at the /view endpoint to all other replicas 
#### Key-to-Shard Mapping Mechanism
  - The data structures we used for this were our Ring, Shard, and VirtShard structs. 
  - We used consistent hashing to map keys to shards.
  - The shards are initially set up as a list which the nodes are added two 
    round robin style.
  - Each shard then creates 10 virtual shards. Each Virtual Shard contains a
    unique hash (using crc32) and the id of the corresponding physical shard.
  - All virtual shards are sorted by their hash into a list (called the ring).
  - To find the Shard given a key, we use binary search and the list of virtual
    shards to find the smallest virtual shard hash that is still bigger than 
    key's hash. 
#### Resharding Mechanism
  - The data structures we used for this were our Ring, Shard, and VirtShard structs, as well as a mutex for locking the kvs while shuffling the data.
  - When Resharding is started, the activator node generates a new ring which
    contains new shards, virtual shards, and evenly distributed nodes as 
    replicas
  - The activator node broadcasts the new ring with its shard mappings to all 
    nodes. All nodes that receive this broadcast will copy the same ring 
    object
  - after obtaining the new ring, all nodes including the activator will call 
    shuffleKvsData(). shuffleKvsData loops through every key in the data base and 
    - First sends it to all members of correct shard under the new sharding 
    - Second deletes it from it's own database if it no longer belongs  
