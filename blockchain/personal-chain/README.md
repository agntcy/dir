## Personal chain

Runs a personal chain using Hardhat as a containerised service.
Used for development and testing.

NOTE: Chain state is not preserved between runs.

### Usage

The container exposes (Hardhat-flavor of) Ethereum JSON-RPC API over WebSocket and HTTP on port `8545`.
To build and deploy, run:

```bash
IMAGE_REPO="ghcr.io/agntcy/hardhat-chain"
IMAGE_TAG="latest"

## Build
docker build . -t $IMAGE_REPO:$IMAGE_TAG

## Run in Docker
docker run -it --rm -d -p 8545:8545 $IMAGE_REPO:$IMAGE_TAG

## Run in K8s (using Helm)
helm install agntcy-chain $REPO_ROOT/blockchain/charts/chain \
    --set image.repository=$IMAGE_REPO \
    --set image.tag=$IMAGE_TAG
```
