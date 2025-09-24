# Directory - Public Good Instance

There is a public staging environment with versions of Directory API, 
with its own SPIRE-based roots of trust.

The endpoints are as follows:

- https://api.directory.agntcy.org - Directory API Service
- https://spire.directory.agntcy.org - Directory SPIRE Server API for federation
- https://status.directory.agntcy.org - Status Page for Directory services

**NOTE**: The staging environment provides neither SLO guarantees nor the same protection of data.
This environment is meant for development and testing only. It is not appropriate to use for production purposes.

## Onboarding

To join the staging environment, you need to be added to the SPIRE server as a trusted federation member.
You can request this by opening a PR in the [agntcy/dir](https://github.com/agntcy/dir) repository with the details of your SPIRE server.

You will need to provide the following details:
- **SPIRE Server Endpoint** - The endpoint of your SPIRE server.
- **Trust Domain** - The trust domain for your organization.
- **Root CA** - The root CA certificate of the SPIRE server.

You can find the full example in the [onboarding/spire.template.yaml](onboarding/spire.template.yaml) file.
An example configuration for your SPIRE server might look like this

```yaml
trustDomain: example.com
bundleEndpointURL: https://spire.example.com
bundleEndpointProfile:
  type: https_spiffe
  endpointSPIFFEID: spiffe://example.com/spire/server
trustDomainBundle: |-
    # Your certificate data here
```

In addition to being onboarded, you also need to configure your own SPIRE server to trust the Directory SPIRE server as a federation peer.
You can find the Public Directory SPIRE server details at [spire.directory.yaml](spire.directory.yaml).

## Usage

Once you are onboarded, you can use the public Directory API by configuring your client to interact with it.

```yaml
# Server address of the public Directory API
serverAddress: spire.directory.agntcy.org

# SPIRE Agent Socket Path for Workload API.
# It assumes that your SPIRE setup is already
# configured to federate with Directory SPIRE server.
spiffeSocketPath: /tmp/spire-agent/public.sock
```

## Support

For support, please open an issue in the [agntcy/dir](https://github.com/agntcy/dir) repository.
For urgent issues, you can reach out via email to [support@agntcy.org](mailto:support@agntcy.org).

## Legal

By using the staging environment, you agree to the [Terms of Service](https://docs.agntcy.org/dir/public_instance/terms) and [Privacy Policy](https://docs.agntcy.org/dir/public_instance/privacy).
