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
    - Implementation for:
        - ***FILL IN HERE***
2. Chris Sterza
    - Wrote README

## Mechanism Description
#### Tracking Causal Dependencies
  - Causal Dependencies were tracked using vector clocks which consisted of a map of socket addresses and integer clock values. Vector clocks were only incremented on replica puts and deletes (aka only message sends) and were passed via the message's ```'causal-metadata'``` field.
  - We used the causal broadcast algorithm learned in class to compare the vector clocks. Namely we had a function that checked if the sender's vector clock was only one larger than the local clock in the sender's position and equal to or less than the local clock for every other position. If it was, the function returned true, otherwise false. Other functions used this return value to determine the next course of action.
#### Detecting Down Replicas
  - If at any time a replica took more than one second to respond to any http request, that replica would be removed from the sender's view and the sender would broadcast a DELETE message at the /view endpoint to all other replicas 
#### Key-to-Shard Mapping Mechanism
  - The data structures we used for this were our Ring, Shard, and VirtShard structs. 
  - The approach we took to map keys to shards was to use consistent hashing.
    - When putting keys into the kvs, the key is hashed using crc32 and then the corresponding shard id is found. If the node is not in the desired shard, it forwards the put request via proxy to the correct shard. Otherwise it stores the data to its own store.
    - When getting values from the kvs, the key is hashed as above and the corresponding shard id where the data is stored is found. If the node is not in the desired shard, it forwards the get request via proxy to the correct shard. Otherwise it retreives the data from its own store.
  - Our rationale for using consistent hashing for our sharding was because it should result in less data migration when a shard is added or goes down.
#### Resharding Mechanism
  - The data structures we used for this were our Ring, Shard, and VirtShard structs, as well as a mutex for locking the kvs while shuffling the data. 
  - The approach we took to resharding the data was the following
    - First verify the new shard count to ensure that it was properly formatted and that there were enough shards to provide fault tolerance. Then we constructed the new Ring object using the desired number of shards. The node would then retreive its new shard id from the new ring and then respond to the client.
    - Afterwards, the node checks the keys to see if any of its old keys no longer belong to its own store. If they don't they're deleted. The node also broadcasts the keys to their respective new shards.
    - Concurrently, the node also broadcasts the new Ring object to all the other nodes so that they're aware of the new sharding.
  - Our rationale for broadcasting the new Ring object was that not every node needed to construct and compute this new ring when all the information would have been identical. It was easier to just have the first node compute it and then broadcast it to the others.