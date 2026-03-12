```bash
# Push manifest
oras push localhost:8080/dir:example-agent \
    --artifact-type application/vnd.aaif.ai.manifest.v1+json \
    --annotation "org.aaif.ai.card.id=did:example:agent-finance-001" \
    --annotation "org.aaif.ai.card.specVersion=1.0" \
    --annotation "org.opencontainers.image.created=2026-03-10T15:00:00Z" \
    README.md:application/vnd.oasf.card.v1+json

# Generate sample signing key
notation cert delete --type ca --store ai-example.io --all -y
notation cert generate-test --default "ai-example.io"

# Sign using notation
notation sign localhost:8080/dir:example-agent

# Create and push AI Catalog with example-agent
oras manifest index create \
    localhost:8080/dir:catalog \
    --artifact-type "application/vnd.aaif.ai.catalog.v1+json" \
    example-agent

```