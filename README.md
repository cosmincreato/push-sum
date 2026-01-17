# Push-Sum Algorithm

A Go project that implements the Push-Sum gossip algorithm to compute the global average across a decentralized network. The algorithm works by modeling nodes as independent actors in a Ring Topology, where each node communicates only with its immediate neighbors to exchange and aggregate local estimates ($s, w$) until a system-wide consensus is reached.