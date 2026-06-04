# Federation Bundle Profiles

Directory supports two federation bundle profiles for secure trust bundle exchange between SPIRE servers. Each profile implements distinct security models, infrastructure requirements, and operational characteristics. This document provides technical guidance to assist in profile selection and implementation.

For step-by-step federation setup with the public Directory network, see [Running a Federated Directory Instance](dir-federation-setup.md).

## Profile Comparison

| Aspect | https_web | https_spiffe |
|--------|-----------|--------------|
| Transport Protocol | Standard HTTPS | SPIFFE mutual TLS |
| Certificate Type | CA-signed X.509 (Let's Encrypt, Enterprise CA) | SPIFFE X.509-SVID |
| Certificate Validation | DNS SANs with CA chain verification | SPIFFE ID URI SAN with Trust Bundle |
| SSL Passthrough | Not required | Required |
| Bootstrap Bundle | Not required | Required |
| cert-manager Dependency | Required | Not required |
| DNS Requirement | Public DNS resolution required | Optional (IP addresses supported) |
| Traffic Inspection | Supported (NGINX terminates TLS) | Not supported (end-to-end encryption) |
| CA Trust Model | Relies on public or enterprise CA | Self-contained SPIFFE trust |
| Implementation Complexity | Medium | Medium-High |
| Estimated Setup Time | Approximately 30 minutes | Approximately 45 minutes |

The profiles are described in more detail below:

=== "https_web profile"

    Federation over standard HTTPS using CA-signed certificates. The SPIRE server presents a certificate validated through traditional DNS Subject Alternative Names (SANs) and CA certificate chain verification. This profile leverages existing PKI infrastructure and standard TLS implementations.

    **Architecture:**

    ```mermaid
    flowchart TB
      Remote[Remote SPIRE Bundle Consumer]
      NGINX[NGINX Ingress Controller<br/>• Terminates TLS Let's Encrypt<br/>• Traffic inspection/logging<br/>• Initiates TLS to backend]
      SPIRE[SPIRE Server Bundle Provider]

      Remote -->|"Standard HTTPS (validates DNS SAN)"| NGINX
      NGINX -->|"Internal HTTPS with SNI"| SPIRE
    ```

    **Advantages:**

    1. **No SSL Passthrough Requirement**

        - Compatible with standard NGINX configurations
        - Aligns with enterprise security policies restricting SSL passthrough
        - No special ingress controller configuration required

    2. **Simplified Initial Trust Establishment**

        - Remote SPIRE server can fetch bundles immediately
        - No manual trust bundle exchange required
        - Eliminates out-of-band coordination overhead

    3. **Traffic Inspection Compatibility**

        - Web Application Firewalls (WAF) can inspect traffic
        - Load balancers maintain request/response visibility
        - DDoS protection operates at Layer 7 (application layer)

    4. **Standardized Certificate Management**

        - Integrates with cert-manager for automation
        - Leverages established CA infrastructure
        - Automatic certificate renewal (Let's Encrypt)

    5. **Simplified Troubleshooting**

        - Standard diagnostic tools functional (curl, openssl s_client)
        - Certificate chain validation is transparent
        - DNS-based validation errors provide clear feedback

    **Limitations:**

    1. **External CA Dependency**

        - Trust anchored in public or enterprise Certificate Authority
        - Certificate revocation depends on CA infrastructure availability
        - Not suitable for architectures requiring complete trust independence

    2. **DNS Infrastructure Requirement**

        - Requires publicly resolvable DNS records
        - DNS must resolve from all federation partner networks
        - Vulnerable to DNS-based attacks (hijacking, poisoning)

    3. **cert-manager Dependency**

        - Requires cert-manager installation and maintenance
        - ClusterIssuer must be properly configured
        - Additional component in the operational stack

    4. **Limited Air-Gap Support**

        - Let's Encrypt requires internet connectivity for ACME challenges
        - Private CA deployments require certificate distribution
        - Increased complexity in isolated network environments

    **Recommended Use Cases:**

    - Cloud-native deployments (AWS, Google Cloud, Azure)
    - Organizations with existing cert-manager infrastructure
    - Environments with traffic inspection/WAF requirements
    - Scenarios prioritizing rapid onboarding
    - Public-facing federation endpoints
    - Organizations restricting SSL passthrough

=== "https_spiffe profile"

    Federation over SPIFFE mutual TLS using X.509-SVIDs issued by SPIRE. The SPIRE server presents a certificate containing a SPIFFE ID in the URI Subject Alternative Name (SAN), validated against a pre-shared trust bundle. This profile implements pure SPIFFE trust without external CA dependencies.

    **Architecture:**

    ```mermaid
    flowchart TB
      Remote[Remote SPIRE Bundle Consumer]
      NGINX[NGINX Ingress Controller<br/>• SSL Passthrough enabled<br/>• Forwards encrypted traffic opaquely<br/>• No inspection or termination]
      SPIRE[SPIRE Server SPIFFE mTLS Bundle Provider]

      Remote -->|"SPIFFE mTLS (URI SAN + bootstrap)"| NGINX
      NGINX -->|"Direct TLS passthrough"| SPIRE
    ```

    **Advantages:**

    1. **Pure SPIFFE Trust Model**

        - No reliance on external Certificate Authorities
        - Self-contained cryptographic identity verification
        - Aligns with zero-trust architecture principles

    2. **Air-Gap Environment Compatibility**

        - No internet connectivity required for operation
        - Functions in completely isolated network environments
        - DNS resolution optional (IP addresses supported)

    3. **Eliminated cert-manager Dependency**

        - SPIRE manages certificate lifecycle internally
        - Reduces operational components
        - Simplified cluster requirements

    4. **End-to-End Encryption Guarantee**

        - Traffic encrypted from source SPIRE to destination SPIRE
        - No intermediate inspection or decryption points
        - Maximum confidentiality assurance

    5. **Strong Cryptographic Identity Binding**

        - Certificate cryptographically bound to specific SPIFFE ID
        - Enhanced authenticity verification
        - Federation partner explicitly identified and validated

    **Limitations:**

    1. **SSL Passthrough Infrastructure Requirement**

        - Many enterprise environments restrict or prohibit SSL passthrough
        - NGINX ingress controller requires special configuration
        - Not universally supported across ingress implementations

    2. **Bootstrap Bundle Coordination Required**

        - Manual exchange of initial trust bundles necessary
        - Out-of-band communication channel required
        - Coordination overhead between federation partners

    3. **Traffic Inspection Incompatibility**

        - Web Application Firewalls cannot inspect traffic
        - Load balancers lose request/response visibility
        - DDoS protection limited to Layer 4 (transport layer)

    4. **Increased Troubleshooting Complexity**

        - Standard diagnostic tools require SPIFFE certificates
        - Error messages (e.g., "certificate contains no URI SAN") require SPIFFE expertise
        - Steeper learning curve for operations teams

    5. **Bootstrap Bundle Staleness Risk**

        - Federation breaks if CA rotates before bundle update
        - Requires monitoring of bundle freshness
        - Manual intervention required for trust bundle synchronization

    **Recommended Use Cases:**

    - Air-gapped or network-isolated environments
    - Zero-trust architectures eliminating CA dependencies
    - Organizations with SSL passthrough capabilities
    - Scenarios requiring maximum security and confidentiality
    - Pure SPIFFE/SPIRE architectural deployments
    - Environments where traffic inspection is explicitly prohibited

## Decision Matrix

### Select the `https_web` profile if:

- cert-manager is installed or can be deployed in your cluster
- NGINX ingress controller does not support or allow SSL passthrough
- Organizational security policies require traffic inspection or Web Application Firewall (WAF)
- Standard CA-based certificates are acceptable for your trust model
- Public DNS resolution is available for federation endpoints
- Operational priority is rapid deployment and simplified troubleshooting
- Deployment targets public cloud infrastructure (AWS, Google Cloud, Azure)

### Select the `https_spiffe` profile if:

- NGINX ingress controller supports and allows SSL passthrough configuration
- Deployment environment is air-gapped or network-isolated
- Zero-trust architecture requires elimination of external CA dependencies
- End-to-end encryption without intermediate inspection is mandatory
- Existing architecture is SPIFFE-centric
- Coordination for bootstrap bundle exchange is operationally feasible
- Organizational security model prioritizes pure cryptographic identity over CA trust

### Default Recommendation

For organizations without specific constraints, `https_web` is recommended as the default profile due to:

- Broader infrastructure compatibility (no SSL passthrough requirement)
- Simplified initial setup (no bootstrap bundle coordination)
- Reduced operational complexity
- Compatibility with standard enterprise security controls

## Bootstrap Bundle Exchange Process (https_spiffe)

The bootstrap bundle exchange is a critical prerequisite for `https_spiffe` federation. This section provides detailed procedures for bundle generation, validation, and configuration.

### Understanding Bootstrap Bundles

A bootstrap bundle contains the public key material for a trust domain's Certificate Authority. The remote SPIRE server uses this bundle to validate the SPIFFE X.509-SVID presented by the federation endpoint during the initial TLS handshake.

Key characteristics of bootstrap bundles:

- Contains CA certificate and public keys (no private keys)
- Safe to transmit over untrusted channels
- Must be kept current as CAs rotate

### Setting Up Bootstrap Bundles

1. Extract Bundle from SPIRE Server

    There are three methods to extract the bootstrap bundle from the SPIRE server:

    === "Using kubectl exec (Recommended)"

        Connect to the SPIRE server pod and extract the bundle:

        ```bash
        # Connect to SPIRE server pod and extract bundle
        kubectl exec -n dir-spire deployment/spire-server -c spire-server -- \
          /opt/spire/bin/spire-server bundle show -format spiffe > my-trust-domain-bundle.json

        # Verify bundle contents
        cat my-trust-domain-bundle.json
        ```

        ??? "Expected output format"

            ```json
            {
              "keys": [
                {
                  "use": "x509-svid",
                  "kty": "RSA",
                  "n": "xGOr-H7A-qw...",
                  "e": "AQAB",
                  "x5c": [
                    "MIIC..."
                  ]
                },
                {
                  "use": "jwt-svid",
                  "kty": "RSA",
                  "kid": "bJV3z...",
                  "n": "yH3s-K9B...",
                  "e": "AQAB"
                }
              ],
              "spiffe_refresh_hint": 450000,
              "spiffe_sequence_number": 1
            }
            ```

    === "Using spire-server CLI Locally"

        If you have the SPIRE server binary installed locally and configured with access to the SPIRE server:

        ```bash
        # Show bundle in SPIFFE format
        spire-server bundle show \
          -format spiffe \
          -socketPath /run/spire/sockets/server.sock \
          > my-trust-domain-bundle.json
        ```

    === "Via Federation Endpoint"

        Once your federation endpoint is operational, bundles can be fetched via HTTPS:

        ```bash
        # Fetch bundle from federation endpoint
        curl -k https://your-spire.example.com > my-trust-domain-bundle.json
        ```

2. Validate the bundle structure to ensure it is well-formed:

    ```bash
    # Verify JSON is well-formed
    jq '.' my-trust-domain-bundle.json

    # Check for required x509-svid key
    jq '.keys[] | select(.use == "x509-svid")' my-trust-domain-bundle.json

    # Verify key count (should have at least one x509-svid key)
    jq '.keys | length' my-trust-domain-bundle.json

    # Optionally decode and inspect the X.509 certificate
    jq -r '.keys[] | select(.use == "x509-svid") | .x5c[0]' my-trust-domain-bundle.json | \
      base64 -d | \
      openssl x509 -inform DER -text -noout
    ```

3. Add the bundle to your ClusterFederatedTrustDomain resource configuration:

    ```yaml
    # onboarding/federation/partner.com.yaml
    className: dir-spire
    trustDomain: partner.com
    bundleEndpointURL: https://spire.partner.com
    bundleEndpointProfile:
      type: https_spiffe
      endpointSPIFFEID: spiffe://partner.com/spire/server
    trustDomainBundle: |-
      {
        "keys": [
          {
            "use": "x509-svid",
            "kty": "RSA",
            "n": "xGOr-H7A-qw...",
            "e": "AQAB",
            "x5c": ["MIIC..."]
          },
          {
            "use": "jwt-svid",
            "kty": "RSA",
            "kid": "bJV3z...",
            "n": "yH3s-K9B...",
            "e": "AQAB"
          }
        ],
        "spiffe_refresh_hint": 450000,
        "spiffe_sequence_number": 1
      }
    ```

!!! note
    Organizations must establish their own secure procedures for exchanging bootstrap bundles with federation partners. The bundle exchange mechanism (email, file transfer, version control, etc.) should align with organizational security policies.
