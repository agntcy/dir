### Authorization Policies

Directory supports fine-grained authorization policies that control which SPIFFE trust domains can access specific API methods. Authorization policies work in conjunction with authentication (either X.509-SVID or JWT-SVID) to provide comprehensive access control.

#### Policy Format

Authorization policies use a simple CSV format based on Casbin:

```
p,<trust_domain>,<api_method>
```

Where:
- `trust_domain`: SPIFFE trust domain extracted from the client's X.509-SVID or JWT-SVID
- `api_method`: gRPC method name in the format `/package.Service/Method`

#### Matching Rules

Policies support multiple matching strategies:

1. **Exact matching**: Match specific trust domain and API method
2. **Wildcard matching**: Use `*` to match any value
3. **Prefix matching**: Use patterns like `/service/*` to match all methods under a path
4. **Regex matching**: Use regular expressions for complex patterns

The authorization system evaluates policies using both `keyMatch` (for prefix patterns) and `regexMatch` (for regular expressions).

#### Common Policy Examples

**Allow full access for your trust domain:**
```
p,example.org,*
```

**Allow read-only access for external trust domains:**
```
p,*,/agntcy.dir.store.v1.StoreService/Pull
p,*,/agntcy.dir.store.v1.StoreService/PullReferrer
p,*,/agntcy.dir.store.v1.StoreService/Lookup
```

**Allow sync operations only for dedicated sync services:**
```
p,sync.example.org,/agntcy.dir.sync.v1.SyncService/*
```

**Allow access for all subdomains using regex:**
```
p,^.*\.example\.org$,*
```

**Mixed policy (internal full access, external read-only):**
```
# Full access for internal services
p,example.org,*

# Read-only access for external partners
p,partner1.com,/agntcy.dir.store.v1.StoreService/Pull
p,partner1.com,/agntcy.dir.store.v1.StoreService/PullReferrer
p,partner1.com,/agntcy.dir.store.v1.StoreService/Lookup

# Another partner with different requirements
p,partner2.com,/agntcy.dir.store.v1.StoreService/Pull
p,partner2.com,/agntcy.dir.sync.v1.SyncService/RequestRegistryCredentials
```

#### Available API Methods

**Store Service:**
- `/agntcy.dir.store.v1.StoreService/Push` - Create or update records
- `/agntcy.dir.store.v1.StoreService/Pull` - Retrieve records
- `/agntcy.dir.store.v1.StoreService/Lookup` - Search for records
- `/agntcy.dir.store.v1.StoreService/Delete` - Remove records
- `/agntcy.dir.store.v1.StoreService/PushReferrer` - Create referrer relationships
- `/agntcy.dir.store.v1.StoreService/PullReferrer` - Retrieve referrer relationships

**Sync Service:**
- `/agntcy.dir.sync.v1.SyncService/CreateSync` - Initiate sync operations
- `/agntcy.dir.sync.v1.SyncService/ListSyncs` - List sync operations
- `/agntcy.dir.sync.v1.SyncService/GetSync` - Get sync status
- `/agntcy.dir.sync.v1.SyncService/DeleteSync` - Cancel sync operations
- `/agntcy.dir.sync.v1.SyncService/RequestRegistryCredentials` - Request credentials for sync

#### Enabling Authorization

To enable authorization policies in your Helm deployment:

```yaml
authz_policies_csv: |
  p,example.org,*
  p,*,/agntcy.dir.store.v1.StoreService/Pull
  p,*,/agntcy.dir.store.v1.StoreService/PullReferrer
  p,*,/agntcy.dir.store.v1.StoreService/Lookup

config:
  # Authentication must be enabled first
  authn:
    enabled: true
    mode: "x509"  # or "jwt"
    socket_path: "unix:///run/spire/agent-sockets/api.sock"

  # Authorization policies
  authz:
    enabled: true
    enforcer_policy_file_path: "/etc/agntcy/dir/authz_policies.csv"
```

**Important:** Authorization requires authentication to be enabled. The authorization system extracts the trust domain from the authenticated SPIFFE ID (either from X.509-SVID or JWT-SVID) and evaluates it against the configured policies.


