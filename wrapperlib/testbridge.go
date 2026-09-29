// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

/*
#include <stdlib.h>
*/
import "C"

import "unsafe"

// This file exists solely to let the _test.go files in this package call the
// cgo-exported functions using plain Go types (string/int64) instead of C
// types directly. The cgo tool refuses to process any file whose name ends
// in "_test.go" ("use of cgo in test ... not supported"), so the C.CString/
// C.GoString/C.free conversions that real callers would perform at the ABI
// boundary have to live in a regular .go file. It contains no business
// logic of its own -- every call is forwarded verbatim to the corresponding
// //export function in this package, and every returned *C.char is freed via
// FreeCString exactly as a real caller is required to do.

// bridgeNoHandle invokes a no-handle, single-JSON-argument exported function
// (e.g. SignWithKey, VerifyWithKey) and returns its JSON response as a Go string.
func bridgeNoHandle(fn func(*C.char) *C.char, reqJSON string) string {
	cReq := C.CString(reqJSON)
	defer C.free(unsafe.Pointer(cReq))

	cResp := fn(cReq)
	defer FreeCString(cResp)

	return C.GoString(cResp)
}

// bridgeHandle invokes a handle-based, single-JSON-argument exported function
// and returns its JSON response as a Go string.
func bridgeHandle(fn func(C.longlong, *C.char) *C.char, handle int64, reqJSON string) string {
	cReq := C.CString(reqJSON)
	defer C.free(unsafe.Pointer(cReq))

	cResp := fn(C.longlong(handle), cReq)
	defer FreeCString(cResp)

	return C.GoString(cResp)
}

// bridgeHandleOnly invokes a handle-only exported function (e.g. CloseClient)
// and returns its JSON response as a Go string.
func bridgeHandleOnly(fn func(C.longlong) *C.char, handle int64) string {
	cResp := fn(C.longlong(handle))
	defer FreeCString(cResp)

	return C.GoString(cResp)
}

// bridgeFreeCStringSmoke exercises FreeCString directly (via a fresh
// allocation, not a function return) for wrapper_test.go's smoke test.
func bridgeFreeCStringSmoke(s string) {
	cStr := C.CString(s)
	FreeCString(cStr)
}
