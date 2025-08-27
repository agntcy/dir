package authz

allow if {
    input.trust_domain == "dir.com"
}
