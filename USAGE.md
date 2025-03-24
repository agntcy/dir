# Usage

This document defines a basic overview on how to use the Dir tooling.
To run the examples below, it is required to have an up-and-running Dir API instance.

Although the following example is shown for a CLI usage scenario,
there is an effort on exposing the same functionality via Golang and Python SDKs.

## Storage API

This API enables the interaction with a local storage layer.

```bash
  # push
  dir push <digest>

  # pull
  dir pull <digest>
  dir pull <digest> --verify

  # lookup returns basic metadata about the agent
  dir info <digest>
```

## Routing API

This API enables the interaction with the networking layer.

### Announce

Broadcast the data to the network (DHT), allowing content discovery.
The data will be republished to the network periodically by the API server.
This is to avoid stale data, as the data across the network has TTL.
This API only works for the data already pushed to local store.

```bash
  # Publish the data across the network.
  # It is not guaranteed that this will succeed.
  dir publish <digest>

  # Remove the data published to the network.
  # This simply drops the auto-announce for the data from the local API.
  # It may take upto TTL time for the record to be dropped by the network.
  dir unpublish <digest>
```

Use `--local` to publish data only to local routing table.
Local data cannot be pulled by the network layer,
while for published data, peers can try to request the given digests.

### Discover

Search for the data across the network.
This API supports both unicast- mode for routing to specific peers/sources,
as well as multicast mode for tag-based routing.

```bash
  # Get a list of peer addresses holding specific agents, ie. find the location of data.
  dir list <digest> --peers

  # Get a list of keys that a given peer can serve, ie. find the type of data.
  # Keys are OASF (e.g. skills) and Dir-specific models (e.g. repos).
  dir list <peer-id> --keys

  # Get a list of agent digests across the network that can satisfy the query.
  dir list --keys skill=voice,skill=coding
```

Use `--local` to list data available on the local routing table.  
Use `--max-hops` to limit the number of routing hops when traversing the network.  
Use `--sync` to sync the discovered data into our local routing table.  
Use `--pull` to pull the discovered data into our local storage layer.  
If pulling is used, use `--verify` to verify each received record.  
Use one of `--allowed peerIDs`, `--blocked peerIds` to allow-list or block-list specific peers during network traversal.

Notes:

- It is not guaranteed that the data is available.
- It is not guaranteed that the results are valid.
- It is not guaranteed that the results are up-to-date.
