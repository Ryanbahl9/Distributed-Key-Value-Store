# CSE138_Assignment4

we need a 

PostKvs()
  if wrong shard
    proxy to correct shard
    return
  execute post in local KVS
  broadcast post 


ReShard(x)
  Take the view
  divide it into x shards









# Sharding
## Re-sharding 
  calculate new shards in ring, look through every key in your local store and send it to the correct node 