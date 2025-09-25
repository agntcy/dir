# Directory - Public Staging Instance

Welcome to the **Directory Public Staging Environment** - your gateway to testing and developing
with the decentralized AI agent discovery network! This environment provides a fully functional
Directory instance for development, testing, and exploration purposes.

## Table of Contents
- [What is Directory?](#-what-is-directory)
- [Architecture Overview](#-architecture-overview)
- [Available Endpoints](#-available-endpoints)
- [Quick Start Guide](#-quick-start-guide)
- [Use Cases](#-use-cases)
- [Troubleshooting](#-troubleshooting)
- [Getting Help](#-getting-help)

## 🎯 What is Directory?

Directory is a decentralized peer-to-peer network that enables:
- **AI Agent Discovery**: Find agents by capabilities like skills, domains, and modules
- **Secure Publication**: Publish agent metadata with cryptographic verification
- **Network Federation**: Connect multiple Directory instances securely
- **Capability Matching**: Match agent capabilities to specific requirements

**⚠️ Important Notice:** This is a staging environment for development and testing only.

- ❌ No SLA guarantees
- ❌ No data persistence guarantees  
- ❌ Not suitable for production use
- ✅ Useful for development, testing, and exploration

## 📊 Architecture Overview

```
┌─────────────────────┐    ┌──────────────────────┐    ┌─────────────────────┐
│   Your Application  │    │  Directory Network   │    │  Other Federation   │
│                     │    │                      │    │     Members         │
│  ┌─────────────┐    │    │ ┌──────────────────┐ │    │                     │
│  │ Directory   │◄───┼────┼►│ Directory API    │ │    │ ┌─────────────────┐ │
│  │ Client SDK  │    │    │ │ Service          │ │    │ │   Partner Org   │ │
│  └─────────────┘    │    │ └──────────────────┘ │    │ │   Directory     │ │
│                     │    │                      │    │ │   Instances     │ │
│  ┌─────────────┐    │    │ ┌──────────────────┐ │◄───┼─┤                 │ │
│  │ SPIRE Agent │◄───┼────┼►│ SPIRE Server     │ │    │ └─────────────────┘ │
│  └─────────────┘    │    │ │ (Federation)     │ │    │                     │
└─────────────────────┘    │ └──────────────────┘ │    └─────────────────────┘
                           └──────────────────────┘
```

## 🌐 Available Endpoints

| Service              | URL                                   | Purpose                                     |
| -------------------- | ------------------------------------- | ------------------------------------------- |
| **Directory API**    | `https://api.directory.agntcy.org`    | Main API for agent discovery and management |
| **SPIRE Federation** | `https://spire.directory.agntcy.org`  | SPIRE server for secure identity federation |
| **Status Dashboard** | `https://status.directory.agntcy.org` | Real-time service status and monitoring     |

## 🚀 Quick Start Guide

### Prerequisites

Before you begin, ensure you have:
- [ ] A SPIRE server setup in your organization
- [ ] Basic understanding of SPIFFE/SPIRE concepts
- [ ] Directory client SDK or CLI tools available

### Prepare Your Environment

#### Option 1: Using Directory CLI

1. **Install the CLI**:
   ```bash
   # Using Homebrew (Linux/macOS)
   brew tap agntcy/dir https://github.com/agntcy/dir
   brew install dirctl
   
   # Or download directly from releases
   curl -L https://github.com/agntcy/dir/releases/latest/download/dirctl-linux-amd64 -o dirctl
   chmod +x dirctl
   sudo mv dirctl /usr/local/bin/
   ```

2. **Configure the client**:
   ```bash
   dirctl config set server-address api.directory.agntcy.org
   dirctl config set spiffe-socket-path /tmp/spire-agent/public.sock
   ```

3. **Test the connection**:
   ```bash
   dirctl ping
   # Expected: ✅ Connected to Directory API at api.directory.agntcy.org
   ```

#### Option 2: Using Directory Client SDK

Choose your preferred language:

<details>
<summary><strong>Go SDK</strong></summary>

```go
package main

import (
    "context"
    "log"
    
    "github.com/agntcy/dir/client"
)

func main() {
    // Create client with SPIRE support
    config := &client.Config{
        ServerAddress:     "api.directory.agntcy.org",
        SpiffeSocketPath:  "/tmp/spire-agent/public.sock",
    }
    client, _ := client.New(client.WithConfig(config))

    // Test connection
    _, err := client.Ping(context.Background())
    if err != nil {
        log.Printf("❌ Connection failed: %v", err)
        return
    }

    log.Println("✅ Connected to Directory!")

    // Run workflows...
}
```
</details>

<details>
<summary><strong>Python SDK</strong></summary>

```python
from agntcy.dir_sdk.client import Config, Client

def main():
    # Create client with SPIRE support
    config = Config(
        server_address="api.directory.agntcy.org",
        spiffe_socket_path="/tmp/spire-agent/public.sock"
    )
    client = Client(config)

    # Test connection
    try:
        client.ping()
        print("✅ Connected to Directory!")
    except Exception as e:
        print(f"❌ Connection failed: {e}")

    # Run workflows...

if __name__ == "__main__":
    main()
```
</details>

<details>
<summary><strong>JavaScript/Node.js SDK</strong></summary>

```javascript
import {Config, Client} from 'agntcy-dir';

async function main() {
    // Create client with SPIRE support
    const config = new Config({
        serverAddress: "api.directory.agntcy.org",
        spiffeEndpointSocket: "/tmp/spire-agent/public.sock",
    });
    const transport = await Client.createGRPCTransport(config);
    const client = new Client(config, transport);

    // Test connection
    try {
        await client.ping();
        console.log('✅ Connected to Directory!');
    } catch (error) {
        console.error('❌ Connection failed:', error.message);
    }

    // Run workflows...
}

main();
```

**Note**: This SDK is intended for Node.js applications only and will not work in web browsers.
</details>

### Federation Setup (Required)

To interact with the Directory, you need to establish a trusted federation between your SPIRE server and the Directory SPIRE server.

### Step 1: Prepare Your Federation Request

Create a file with your SPIRE server details using the template below:

```yaml
# onboarding/your-org.com.yaml
trustDomain: your-org.com
bundleEndpointURL: https://spire.your-org.com
bundleEndpointProfile:
  type: https_spiffe
  endpointSPIFFEID: spiffe://your-org.com/spire/server
trustDomainBundle: |-
  {
    "keys": [
      {
        "use": "x509-svid",
        "kty": "RSA",
        "n": "your-public-key-here...",
        "e": "AQAB",
        "x5c": ["your-certificate-chain-here..."]
      }
    ]
  }
```

**💡 How to get your trust bundle:**
```bash
# Export your SPIRE server trust bundle
spire-server bundle show -format spiffe > your-trust-bundle.json
```

### Step 2: Submit Federation Request

1. **Fork the repository**: Go to https://github.com/agntcy/dir and click "Fork"

2. **Create your federation file**:
   ```bash
   git clone https://github.com/your-username/dir.git
   cd dir/deployment/onboarding/
   cp spire.template.yaml your-org.com.yaml
   # Edit your-org.com.yaml with your details
   ```

3. **Submit a Pull Request**:
   - Title: `federation(<Trust Domain>): add [Your Organization]`
   - Description: Brief description of your organization and use case
   - Files: Include your completed federation configuration

### Step 3: Configure Your SPIRE Server

Add the Directory SPIRE server as a federation peer in your SPIRE server configuration
by obtaining the [Directory trust bundle](spire.directory.yaml).

Save the trust bundle to the specified path.

### Step 4: Verify Federation

```bash
# Check federation status
spire-server federation list

# Should show federated trust domain
spire-server federation show --trustDomain dir.agntcy.org
```

## 📚 Use Cases

You can find various usage examples at [docs.agntcy.org](https://docs.agntcy.org/dir/scenarios/).

## 🔧 Troubleshooting

### Connection Issues

**Problem**: Cannot connect to Directory API
```bash
# Check SPIRE agent status
spire-agent api fetch x509-svid

# Verify network connectivity
curl -v https://api.directory.agntcy.org

# Check client configuration
dirctl config list
```

### Federation Issues

**Problem**: SPIRE federation not working
```bash
# Verify trust bundle exchange
spire-server federation show --trustDomain dir.agntcy.org

# Test bundle endpoint connectivity
curl https://spire.directory.agntcy.org/
```

### Common Error Messages

| Error                                           | Solution                                                   |
| ----------------------------------------------- | ---------------------------------------------------------- |
| `connection refused`                            | Check if SPIRE agent is running and socket path is correct |
| `x509: certificate signed by unknown authority` | Verify trust bundle configuration                          |
| `context deadline exceeded`                     | Check network connectivity and firewall settings           |
| `permission denied`                             | Ensure proper SPIFFE ID registration and policies          |

## 🆘 Getting Help

### Community Support
- **GitHub Issues**: [Open an issue](https://github.com/agntcy/dir/issues) for bugs and feature requests
- **Discussions**: [GitHub Discussions](https://github.com/agntcy/dir/discussions) for questions and community help
- **Documentation**: [Full Documentation](https://docs.agntcy.org/dir/overview/)

---

**Ready to get started?** 🎉 Follow the [Quick Start Guide](https://docs.agntcy.org/dir/getting-started/) or
check out our [Usage and Examples](https://docs.agntcy.org/dir/scenarios/) for sample applications!
