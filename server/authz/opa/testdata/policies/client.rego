package authz

allow if {
    input.user_id == "client"
    input.request in {"GET"}
}
