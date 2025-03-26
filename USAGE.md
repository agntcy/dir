# Usage

This document defines a basic overview on how to use the project.
To run the examples below, it is required to have an up-and-running Dir API instance.  
Check [Deployment](DEPLOYMENT.md) guide on how to deploy the neccessary services.

> Although the following example is shown for CLI-based usage scenario,
there is an effort on exposing the same functionality via Golang and Python SDKs.

## Storage API

This API enables interaction with the local storage layer.
It is used as an blob store for Dir objects and serves both
the networking and user-specific purposes.

```bash
  # build
  dir build /path/to/source > model.json

  # push
  dir push model.json

  # pull
  dir pull <digest>

  # lookup
  dir info <digest>
```

## Routing API

This API enables interaction with the networking layer.

### Announce

Broadcast the data to the network (DHT), allowing content discovery.
The data will be republished to the network periodically by the API server.
This is to avoid stale data, as the data across the network has TTL.
This API only works for the data already pushed to local store.

```bash
  # Publish the data to your local routing table.
  dir publish <digest>

  # Publish the data across the network.
  # It is not guaranteed that this will succeed.
  dir publish <digest> --network
```

Use `--network` to publish the data to the network.
If not published to the network, the data cannot be discovered by other peers. 
For published data, peers can reach out and request specific objects.

### Discover

Search for the data across the network.
This API supports both unicast- mode for routing to specific peers/objects,
and multicast mode for attribute-based matching and routing.

```bash
  # Get a list of peer addresses holding specific agents, ie. find the location of data.
  dir list --digest <digest>

  # Get a list of labels that you currently have in your local routing table.
  dir list info

  # Get a list of labels that a given peer can serve, ie. find the type of data.
  # Labels are defined by OASF.
  dir list info --peer <peer-id>

  # Get full data about the location and holders of data across the network that can satisfy our query.
  dir list info /skill/voice /skill/coding --network
```

Use `--max-hops` to limit the number of routing hops when traversing the network.  
Use `--sync` to sync the discovered data into our local routing table.  
Use `--pull` to pull the discovered data into our local storage layer.  
If pulling is used, use `--verify` to verify each received record.  
Use one of `--allowed peerIDs`, `--blocked peerIds` to allow-list or block-list specific peers during network traversal.

Notes:

- It is not guaranteed that the data is available.
- It is not guaranteed that the results are valid.
- It is not guaranteed that the results are up-to-date.
