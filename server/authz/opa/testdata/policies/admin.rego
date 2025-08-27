package authz

allow if {
    input.user_id == "admin"
    input.request in {"POST", "GET"}
}
