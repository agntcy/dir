package authz

allow if {
    input.trust_domain == "service.org"
    input.api_method in {"LookupRequest", "PullRequest"}
}