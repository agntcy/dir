import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { env } from 'node:process';
import { readFileSync, mkdtempSync, writeFileSync, rmSync } from 'node:fs';
import { spawnSync } from 'node:child_process';
import { createClient as createClient$1 } from 'spiffe';
import * as zlib from 'node:zlib';
import { promisify } from 'node:util';
import * as http from 'node:http';
import * as https from 'node:https';
import * as http2 from 'node:http2';

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Connect represents categories of errors as codes, and each code maps to a
 * specific HTTP status code. The codes and their semantics were chosen to
 * match gRPC. Only the codes below are valid — there are no user-defined
 * codes.
 *
 * See the specification at https://connectrpc.com/docs/protocol#error-codes
 * for details.
 */
var Code;
(function (Code) {
    /**
     * Canceled, usually be the user
     */
    Code[Code["Canceled"] = 1] = "Canceled";
    /**
     * Unknown error
     */
    Code[Code["Unknown"] = 2] = "Unknown";
    /**
     * Argument invalid regardless of system state
     */
    Code[Code["InvalidArgument"] = 3] = "InvalidArgument";
    /**
     * Operation expired, may or may not have completed.
     */
    Code[Code["DeadlineExceeded"] = 4] = "DeadlineExceeded";
    /**
     * Entity not found.
     */
    Code[Code["NotFound"] = 5] = "NotFound";
    /**
     * Entity already exists.
     */
    Code[Code["AlreadyExists"] = 6] = "AlreadyExists";
    /**
     * Operation not authorized.
     */
    Code[Code["PermissionDenied"] = 7] = "PermissionDenied";
    /**
     * Quota exhausted.
     */
    Code[Code["ResourceExhausted"] = 8] = "ResourceExhausted";
    /**
     * Argument invalid in current system state.
     */
    Code[Code["FailedPrecondition"] = 9] = "FailedPrecondition";
    /**
     * Operation aborted.
     */
    Code[Code["Aborted"] = 10] = "Aborted";
    /**
     * Out of bounds, use instead of FailedPrecondition.
     */
    Code[Code["OutOfRange"] = 11] = "OutOfRange";
    /**
     * Operation not implemented or disabled.
     */
    Code[Code["Unimplemented"] = 12] = "Unimplemented";
    /**
     * Internal error, reserved for "serious errors".
     */
    Code[Code["Internal"] = 13] = "Internal";
    /**
     * Unavailable, client should back off and retry.
     */
    Code[Code["Unavailable"] = 14] = "Unavailable";
    /**
     * Unrecoverable data loss or corruption.
     */
    Code[Code["DataLoss"] = 15] = "DataLoss";
    /**
     * Request isn't authenticated.
     */
    Code[Code["Unauthenticated"] = 16] = "Unauthenticated";
})(Code || (Code = {}));

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Determine whether the given `arg` is a message.
 * If `desc` is set, determine whether `arg` is this specific message.
 */
function isMessage(arg, schema) {
    const isMessage = arg !== null &&
        typeof arg == "object" &&
        "$typeName" in arg &&
        typeof arg.$typeName == "string";
    if (!isMessage) {
        return false;
    }
    if (schema === undefined) {
        return true;
    }
    return schema.typeName === arg.$typeName;
}

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Scalar value types. This is a subset of field types declared by protobuf
 * enum google.protobuf.FieldDescriptorProto.Type The types GROUP and MESSAGE
 * are omitted, but the numerical values are identical.
 */
var ScalarType;
(function (ScalarType) {
    // 0 is reserved for errors.
    // Order is weird for historical reasons.
    ScalarType[ScalarType["DOUBLE"] = 1] = "DOUBLE";
    ScalarType[ScalarType["FLOAT"] = 2] = "FLOAT";
    // Not ZigZag encoded.  Negative numbers take 10 bytes.  Use TYPE_SINT64 if
    // negative values are likely.
    ScalarType[ScalarType["INT64"] = 3] = "INT64";
    ScalarType[ScalarType["UINT64"] = 4] = "UINT64";
    // Not ZigZag encoded.  Negative numbers take 10 bytes.  Use TYPE_SINT32 if
    // negative values are likely.
    ScalarType[ScalarType["INT32"] = 5] = "INT32";
    ScalarType[ScalarType["FIXED64"] = 6] = "FIXED64";
    ScalarType[ScalarType["FIXED32"] = 7] = "FIXED32";
    ScalarType[ScalarType["BOOL"] = 8] = "BOOL";
    ScalarType[ScalarType["STRING"] = 9] = "STRING";
    // Tag-delimited aggregate.
    // Group type is deprecated and not supported in proto3. However, Proto3
    // implementations should still be able to parse the group wire format and
    // treat group fields as unknown fields.
    // TYPE_GROUP = 10,
    // TYPE_MESSAGE = 11,  // Length-delimited aggregate.
    // New in version 2.
    ScalarType[ScalarType["BYTES"] = 12] = "BYTES";
    ScalarType[ScalarType["UINT32"] = 13] = "UINT32";
    // TYPE_ENUM = 14,
    ScalarType[ScalarType["SFIXED32"] = 15] = "SFIXED32";
    ScalarType[ScalarType["SFIXED64"] = 16] = "SFIXED64";
    ScalarType[ScalarType["SINT32"] = 17] = "SINT32";
    ScalarType[ScalarType["SINT64"] = 18] = "SINT64";
})(ScalarType || (ScalarType = {}));

// Copyright 2008 Google Inc.  All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
// * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
// * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
//
// Code generated by the Protocol Buffer compiler is owned by the owner
// of the input file used when generating it.  This code is not
// standalone and requires a support library to be linked with it.  This
// support library is itself covered by the above license.
/**
 * Read a 64 bit varint as two JS numbers.
 *
 * Returns tuple:
 * [0]: low bits
 * [1]: high bits
 *
 * Copyright 2008 Google Inc.  All rights reserved.
 *
 * See https://github.com/protocolbuffers/protobuf/blob/8a71927d74a4ce34efe2d8769fda198f52d20d12/js/experimental/runtime/kernel/buffer_decoder.js#L175
 */
function varint64read() {
    let lowBits = 0;
    let highBits = 0;
    for (let shift = 0; shift < 28; shift += 7) {
        let b = this.buf[this.pos++];
        lowBits |= (b & 0x7f) << shift;
        if ((b & 0x80) == 0) {
            this.assertBounds();
            return [lowBits, highBits];
        }
    }
    let middleByte = this.buf[this.pos++];
    // last four bits of the first 32 bit number
    lowBits |= (middleByte & 0x0f) << 28;
    // 3 upper bits are part of the next 32 bit number
    highBits = (middleByte & 0x70) >> 4;
    if ((middleByte & 0x80) == 0) {
        this.assertBounds();
        return [lowBits, highBits];
    }
    for (let shift = 3; shift <= 31; shift += 7) {
        let b = this.buf[this.pos++];
        highBits |= (b & 0x7f) << shift;
        if ((b & 0x80) == 0) {
            this.assertBounds();
            return [lowBits, highBits];
        }
    }
    throw new Error("invalid varint");
}
/**
 * Write a 64 bit varint, given as two JS numbers, to the given bytes array.
 *
 * Copyright 2008 Google Inc.  All rights reserved.
 *
 * See https://github.com/protocolbuffers/protobuf/blob/8a71927d74a4ce34efe2d8769fda198f52d20d12/js/experimental/runtime/kernel/writer.js#L344
 */
function varint64write(lo, hi, bytes) {
    for (let i = 0; i < 28; i = i + 7) {
        const shift = lo >>> i;
        const hasNext = !(shift >>> 7 == 0 && hi == 0);
        const byte = (hasNext ? shift | 0x80 : shift) & 0xff;
        bytes.push(byte);
        if (!hasNext) {
            return;
        }
    }
    const splitBits = ((lo >>> 28) & 0x0f) | ((hi & 0x07) << 4);
    const hasMoreBits = !(hi >> 3 == 0);
    bytes.push((hasMoreBits ? splitBits | 0x80 : splitBits) & 0xff);
    if (!hasMoreBits) {
        return;
    }
    for (let i = 3; i < 31; i = i + 7) {
        const shift = hi >>> i;
        const hasNext = !(shift >>> 7 == 0);
        const byte = (hasNext ? shift | 0x80 : shift) & 0xff;
        bytes.push(byte);
        if (!hasNext) {
            return;
        }
    }
    bytes.push((hi >>> 31) & 0x01);
}
// constants for binary math
const TWO_PWR_32_DBL = 0x100000000;
/**
 * Parse decimal string of 64 bit integer value as two JS numbers.
 *
 * Copyright 2008 Google Inc.  All rights reserved.
 *
 * See https://github.com/protocolbuffers/protobuf-javascript/blob/a428c58273abad07c66071d9753bc4d1289de426/experimental/runtime/int64.js#L10
 */
function int64FromString(dec) {
    // Check for minus sign.
    const minus = dec[0] === "-";
    if (minus) {
        dec = dec.slice(1);
    }
    // Work 6 decimal digits at a time, acting like we're converting base 1e6
    // digits to binary. This is safe to do with floating point math because
    // Number.isSafeInteger(ALL_32_BITS * 1e6) == true.
    const base = 1e6;
    let lowBits = 0;
    let highBits = 0;
    function add1e6digit(begin, end) {
        // Note: Number('') is 0.
        const digit1e6 = Number(dec.slice(begin, end));
        highBits *= base;
        lowBits = lowBits * base + digit1e6;
        // Carry bits from lowBits to
        if (lowBits >= TWO_PWR_32_DBL) {
            highBits = highBits + ((lowBits / TWO_PWR_32_DBL) | 0);
            lowBits = lowBits % TWO_PWR_32_DBL;
        }
    }
    add1e6digit(-24, -18);
    add1e6digit(-18, -12);
    add1e6digit(-12, -6);
    add1e6digit(-6);
    return minus ? negate(lowBits, highBits) : newBits(lowBits, highBits);
}
/**
 * Losslessly converts a 64-bit signed integer in 32:32 split representation
 * into a decimal string.
 *
 * Copyright 2008 Google Inc.  All rights reserved.
 *
 * See https://github.com/protocolbuffers/protobuf-javascript/blob/a428c58273abad07c66071d9753bc4d1289de426/experimental/runtime/int64.js#L10
 */
function int64ToString(lo, hi) {
    let bits = newBits(lo, hi);
    // If we're treating the input as a signed value and the high bit is set, do
    // a manual two's complement conversion before the decimal conversion.
    const negative = bits.hi & 0x80000000;
    if (negative) {
        bits = negate(bits.lo, bits.hi);
    }
    const result = uInt64ToString(bits.lo, bits.hi);
    return negative ? "-" + result : result;
}
/**
 * Losslessly converts a 64-bit unsigned integer in 32:32 split representation
 * into a decimal string.
 *
 * Copyright 2008 Google Inc.  All rights reserved.
 *
 * See https://github.com/protocolbuffers/protobuf-javascript/blob/a428c58273abad07c66071d9753bc4d1289de426/experimental/runtime/int64.js#L10
 */
function uInt64ToString(lo, hi) {
    ({ lo, hi } = toUnsigned(lo, hi));
    // Skip the expensive conversion if the number is small enough to use the
    // built-in conversions.
    // Number.MAX_SAFE_INTEGER = 0x001FFFFF FFFFFFFF, thus any number with
    // highBits <= 0x1FFFFF can be safely expressed with a double and retain
    // integer precision.
    // Proven by: Number.isSafeInteger(0x1FFFFF * 2**32 + 0xFFFFFFFF) == true.
    if (hi <= 0x1fffff) {
        return String(TWO_PWR_32_DBL * hi + lo);
    }
    // What this code is doing is essentially converting the input number from
    // base-2 to base-1e7, which allows us to represent the 64-bit range with
    // only 3 (very large) digits. Those digits are then trivial to convert to
    // a base-10 string.
    // The magic numbers used here are -
    // 2^24 = 16777216 = (1,6777216) in base-1e7.
    // 2^48 = 281474976710656 = (2,8147497,6710656) in base-1e7.
    // Split 32:32 representation into 16:24:24 representation so our
    // intermediate digits don't overflow.
    const low = lo & 0xffffff;
    const mid = ((lo >>> 24) | (hi << 8)) & 0xffffff;
    const high = (hi >> 16) & 0xffff;
    // Assemble our three base-1e7 digits, ignoring carries. The maximum
    // value in a digit at this step is representable as a 48-bit integer, which
    // can be stored in a 64-bit floating point number.
    let digitA = low + mid * 6777216 + high * 6710656;
    let digitB = mid + high * 8147497;
    let digitC = high * 2;
    // Apply carries from A to B and from B to C.
    const base = 10000000;
    if (digitA >= base) {
        digitB += Math.floor(digitA / base);
        digitA %= base;
    }
    if (digitB >= base) {
        digitC += Math.floor(digitB / base);
        digitB %= base;
    }
    // If digitC is 0, then we should have returned in the trivial code path
    // at the top for non-safe integers. Given this, we can assume both digitB
    // and digitA need leading zeros.
    return (digitC.toString() +
        decimalFrom1e7WithLeadingZeros(digitB) +
        decimalFrom1e7WithLeadingZeros(digitA));
}
function toUnsigned(lo, hi) {
    return { lo: lo >>> 0, hi: hi >>> 0 };
}
function newBits(lo, hi) {
    return { lo: lo | 0, hi: hi | 0 };
}
/**
 * Returns two's compliment negation of input.
 * @see https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Operators/Bitwise_Operators#Signed_32-bit_integers
 */
function negate(lowBits, highBits) {
    highBits = ~highBits;
    if (lowBits) {
        lowBits = ~lowBits + 1;
    }
    else {
        // If lowBits is 0, then bitwise-not is 0xFFFFFFFF,
        // adding 1 to that, results in 0x100000000, which leaves
        // the low bits 0x0 and simply adds one to the high bits.
        highBits += 1;
    }
    return newBits(lowBits, highBits);
}
/**
 * Returns decimal representation of digit1e7 with leading zeros.
 */
const decimalFrom1e7WithLeadingZeros = (digit1e7) => {
    const partial = String(digit1e7);
    return "0000000".slice(partial.length) + partial;
};
/**
 * Write a 32 bit varint, signed or unsigned. Same as `varint64write(0, value, bytes)`
 *
 * Copyright 2008 Google Inc.  All rights reserved.
 *
 * See https://github.com/protocolbuffers/protobuf/blob/1b18833f4f2a2f681f4e4a25cdf3b0a43115ec26/js/binary/encoder.js#L144
 */
function varint32write(value, bytes) {
    if (value >= 0) {
        // write value as varint 32
        while (value > 0x7f) {
            bytes.push((value & 0x7f) | 0x80);
            value = value >>> 7;
        }
        bytes.push(value);
    }
    else {
        for (let i = 0; i < 9; i++) {
            bytes.push((value & 127) | 128);
            value = value >> 7;
        }
        bytes.push(1);
    }
}
/**
 * Read an unsigned 32 bit varint.
 *
 * See https://github.com/protocolbuffers/protobuf/blob/8a71927d74a4ce34efe2d8769fda198f52d20d12/js/experimental/runtime/kernel/buffer_decoder.js#L220
 */
function varint32read() {
    let b = this.buf[this.pos++];
    let result = b & 0x7f;
    if ((b & 0x80) == 0) {
        this.assertBounds();
        return result;
    }
    b = this.buf[this.pos++];
    result |= (b & 0x7f) << 7;
    if ((b & 0x80) == 0) {
        this.assertBounds();
        return result;
    }
    b = this.buf[this.pos++];
    result |= (b & 0x7f) << 14;
    if ((b & 0x80) == 0) {
        this.assertBounds();
        return result;
    }
    b = this.buf[this.pos++];
    result |= (b & 0x7f) << 21;
    if ((b & 0x80) == 0) {
        this.assertBounds();
        return result;
    }
    // Extract only last 4 bits
    b = this.buf[this.pos++];
    result |= (b & 0x0f) << 28;
    for (let readBytes = 5; (b & 0x80) !== 0 && readBytes < 10; readBytes++)
        b = this.buf[this.pos++];
    if ((b & 0x80) != 0)
        throw new Error("invalid varint");
    this.assertBounds();
    // Result can have 32 bits, convert it to unsigned
    return result >>> 0;
}

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Int64Support for the current environment.
 */
const protoInt64 = /*@__PURE__*/ makeInt64Support();
function makeInt64Support() {
    const dv = new DataView(new ArrayBuffer(8));
    // note that Safari 14 implements BigInt, but not the DataView methods
    const ok = typeof BigInt === "function" &&
        typeof dv.getBigInt64 === "function" &&
        typeof dv.getBigUint64 === "function" &&
        typeof dv.setBigInt64 === "function" &&
        typeof dv.setBigUint64 === "function" &&
        (!!globalThis.Deno ||
            typeof process != "object" ||
            typeof process.env != "object" ||
            process.env.BUF_BIGINT_DISABLE !== "1");
    if (ok) {
        const MIN = BigInt("-9223372036854775808");
        const MAX = BigInt("9223372036854775807");
        const UMIN = BigInt("0");
        const UMAX = BigInt("18446744073709551615");
        return {
            zero: BigInt(0),
            supported: true,
            parse(value) {
                const bi = typeof value == "bigint" ? value : BigInt(value);
                if (bi > MAX || bi < MIN) {
                    throw new Error(`invalid int64: ${value}`);
                }
                return bi;
            },
            uParse(value) {
                const bi = typeof value == "bigint" ? value : BigInt(value);
                if (bi > UMAX || bi < UMIN) {
                    throw new Error(`invalid uint64: ${value}`);
                }
                return bi;
            },
            enc(value) {
                dv.setBigInt64(0, this.parse(value), true);
                return {
                    lo: dv.getInt32(0, true),
                    hi: dv.getInt32(4, true),
                };
            },
            uEnc(value) {
                dv.setBigInt64(0, this.uParse(value), true);
                return {
                    lo: dv.getInt32(0, true),
                    hi: dv.getInt32(4, true),
                };
            },
            dec(lo, hi) {
                dv.setInt32(0, lo, true);
                dv.setInt32(4, hi, true);
                return dv.getBigInt64(0, true);
            },
            uDec(lo, hi) {
                dv.setInt32(0, lo, true);
                dv.setInt32(4, hi, true);
                return dv.getBigUint64(0, true);
            },
        };
    }
    return {
        zero: "0",
        supported: false,
        parse(value) {
            if (typeof value != "string") {
                value = value.toString();
            }
            assertInt64String(value);
            return value;
        },
        uParse(value) {
            if (typeof value != "string") {
                value = value.toString();
            }
            assertUInt64String(value);
            return value;
        },
        enc(value) {
            if (typeof value != "string") {
                value = value.toString();
            }
            assertInt64String(value);
            return int64FromString(value);
        },
        uEnc(value) {
            if (typeof value != "string") {
                value = value.toString();
            }
            assertUInt64String(value);
            return int64FromString(value);
        },
        dec(lo, hi) {
            return int64ToString(lo, hi);
        },
        uDec(lo, hi) {
            return uInt64ToString(lo, hi);
        },
    };
}
function assertInt64String(value) {
    if (!/^-?[0-9]+$/.test(value)) {
        throw new Error("invalid int64: " + value);
    }
}
function assertUInt64String(value) {
    if (!/^[0-9]+$/.test(value)) {
        throw new Error("invalid uint64: " + value);
    }
}

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Returns the zero value for the given scalar type.
 */
function scalarZeroValue(type, longAsString) {
    switch (type) {
        case ScalarType.STRING:
            return "";
        case ScalarType.BOOL:
            return false;
        case ScalarType.DOUBLE:
        case ScalarType.FLOAT:
            return 0.0;
        case ScalarType.INT64:
        case ScalarType.UINT64:
        case ScalarType.SFIXED64:
        case ScalarType.FIXED64:
        case ScalarType.SINT64:
            return (longAsString ? "0" : protoInt64.zero);
        case ScalarType.BYTES:
            return new Uint8Array(0);
        default:
            // Handles INT32, UINT32, SINT32, FIXED32, SFIXED32.
            // We do not use individual cases to save a few bytes code size.
            return 0;
    }
}
/**
 * Returns true for a zero-value. For example, an integer has the zero-value `0`,
 * a boolean is `false`, a string is `""`, and bytes is an empty Uint8Array.
 *
 * In proto3, zero-values are not written to the wire, unless the field is
 * optional or repeated.
 */
function isScalarZeroValue(type, value) {
    switch (type) {
        case ScalarType.BOOL:
            return value === false;
        case ScalarType.STRING:
            return value === "";
        case ScalarType.BYTES:
            return value instanceof Uint8Array && !value.byteLength;
        default:
            return value == 0; // Loose comparison matches 0n, 0 and "0"
    }
}

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// bootstrap-inject google.protobuf.FeatureSet.FieldPresence.IMPLICIT: const $name: FeatureSet_FieldPresence.$localName = $number;
const IMPLICIT$3 = 2;
const unsafeLocal = Symbol.for("reflect unsafe local");
/**
 * Return the selected field of a oneof group.
 *
 * @private
 */
function unsafeOneofCase(
// biome-ignore lint/suspicious/noExplicitAny: `any` is the best choice for dynamic access
target, oneof) {
    const c = target[oneof.localName].case;
    if (c === undefined) {
        return c;
    }
    return oneof.fields.find((f) => f.localName === c);
}
/**
 * Returns true if the field is set.
 *
 * @private
 */
function unsafeIsSet(
// biome-ignore lint/suspicious/noExplicitAny: `any` is the best choice for dynamic access
target, field) {
    const name = field.localName;
    if (field.oneof) {
        return target[field.oneof.localName].case === name;
    }
    if (field.presence != IMPLICIT$3) {
        // Fields with explicit presence have properties on the prototype chain
        // for default / zero values (except for proto3).
        return (target[name] !== undefined &&
            Object.prototype.hasOwnProperty.call(target, name));
    }
    switch (field.fieldKind) {
        case "list":
            return target[name].length > 0;
        case "map":
            return Object.keys(target[name]).length > 0;
        case "scalar":
            return !isScalarZeroValue(field.scalar, target[name]);
        case "enum":
            return target[name] !== field.enum.values[0].number;
    }
    throw new Error("message field with implicit presence");
}
/**
 * Returns true if the field is set, but only for singular fields with explicit
 * presence (proto2).
 *
 * @private
 */
function unsafeIsSetExplicit(target, localName) {
    return (Object.prototype.hasOwnProperty.call(target, localName) &&
        target[localName] !== undefined);
}
/**
 * Return a field value, respecting oneof groups.
 *
 * @private
 */
function unsafeGet(target, field) {
    if (field.oneof) {
        const oneof = target[field.oneof.localName];
        if (oneof.case === field.localName) {
            return oneof.value;
        }
        return undefined;
    }
    return target[field.localName];
}
/**
 * Set a field value, respecting oneof groups.
 *
 * @private
 */
function unsafeSet(target, field, value) {
    if (field.oneof) {
        target[field.oneof.localName] = {
            case: field.localName,
            value: value,
        };
    }
    else {
        target[field.localName] = value;
    }
}
/**
 * Resets the field, so that unsafeIsSet() will return false.
 *
 * @private
 */
function unsafeClear(
// biome-ignore lint/suspicious/noExplicitAny: `any` is the best choice for dynamic access
target, field) {
    const name = field.localName;
    if (field.oneof) {
        const oneofLocalName = field.oneof.localName;
        if (target[oneofLocalName].case === name) {
            target[oneofLocalName] = { case: undefined };
        }
    }
    else if (field.presence != IMPLICIT$3) {
        // Fields with explicit presence have properties on the prototype chain
        // for default / zero values (except for proto3). By deleting their own
        // property, the field is reset.
        delete target[name];
    }
    else {
        switch (field.fieldKind) {
            case "map":
                target[name] = {};
                break;
            case "list":
                target[name] = [];
                break;
            case "enum":
                target[name] = field.enum.values[0].number;
                break;
            case "scalar":
                target[name] = scalarZeroValue(field.scalar, field.longAsString);
                break;
        }
    }
}

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
function isObject(arg) {
    return arg !== null && typeof arg == "object" && !Array.isArray(arg);
}
function isReflectList(arg, field) {
    var _a, _b, _c, _d;
    if (isObject(arg) &&
        unsafeLocal in arg &&
        "add" in arg &&
        "field" in arg &&
        typeof arg.field == "function") {
        if (field !== undefined) {
            const a = field;
            const b = arg.field();
            return (a.listKind == b.listKind &&
                a.scalar === b.scalar &&
                ((_a = a.message) === null || _a === void 0 ? void 0 : _a.typeName) === ((_b = b.message) === null || _b === void 0 ? void 0 : _b.typeName) &&
                ((_c = a.enum) === null || _c === void 0 ? void 0 : _c.typeName) === ((_d = b.enum) === null || _d === void 0 ? void 0 : _d.typeName));
        }
        return true;
    }
    return false;
}
function isReflectMap(arg, field) {
    var _a, _b, _c, _d;
    if (isObject(arg) &&
        unsafeLocal in arg &&
        "has" in arg &&
        "field" in arg &&
        typeof arg.field == "function") {
        if (field !== undefined) {
            const a = field, b = arg.field();
            return (a.mapKey === b.mapKey &&
                a.mapKind == b.mapKind &&
                a.scalar === b.scalar &&
                ((_a = a.message) === null || _a === void 0 ? void 0 : _a.typeName) === ((_b = b.message) === null || _b === void 0 ? void 0 : _b.typeName) &&
                ((_c = a.enum) === null || _c === void 0 ? void 0 : _c.typeName) === ((_d = b.enum) === null || _d === void 0 ? void 0 : _d.typeName));
        }
        return true;
    }
    return false;
}
function isReflectMessage(arg, messageDesc) {
    return (isObject(arg) &&
        unsafeLocal in arg &&
        "desc" in arg &&
        isObject(arg.desc) &&
        arg.desc.kind === "message" &&
        (messageDesc === undefined || arg.desc.typeName == messageDesc.typeName));
}

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
function isWrapper(arg) {
    return isWrapperTypeName(arg.$typeName);
}
function isWrapperDesc(messageDesc) {
    const f = messageDesc.fields[0];
    return (isWrapperTypeName(messageDesc.typeName) &&
        f !== undefined &&
        f.fieldKind == "scalar" &&
        f.name == "value" &&
        f.number == 1);
}
function isWrapperTypeName(name) {
    return (name.startsWith("google.protobuf.") &&
        [
            "DoubleValue",
            "FloatValue",
            "Int64Value",
            "UInt64Value",
            "Int32Value",
            "UInt32Value",
            "BoolValue",
            "StringValue",
            "BytesValue",
        ].includes(name.substring(16)));
}

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// bootstrap-inject google.protobuf.Edition.EDITION_PROTO3: const $name: Edition.$localName = $number;
const EDITION_PROTO3$1 = 999;
// bootstrap-inject google.protobuf.Edition.EDITION_PROTO2: const $name: Edition.$localName = $number;
const EDITION_PROTO2$1 = 998;
// bootstrap-inject google.protobuf.FeatureSet.FieldPresence.IMPLICIT: const $name: FeatureSet_FieldPresence.$localName = $number;
const IMPLICIT$2 = 2;
/**
 * Create a new message instance.
 *
 * The second argument is an optional initializer object, where all fields are
 * optional.
 */
function create(schema, init) {
    if (isMessage(init, schema)) {
        return init;
    }
    const message = createZeroMessage(schema);
    if (init !== undefined) {
        initMessage(schema, message, init);
    }
    return message;
}
/**
 * Sets field values from a MessageInitShape on a zero message.
 */
function initMessage(messageDesc, message, init) {
    for (const member of messageDesc.members) {
        let value = init[member.localName];
        if (value == null) {
            // intentionally ignore undefined and null
            continue;
        }
        let field;
        if (member.kind == "oneof") {
            const oneofField = unsafeOneofCase(init, member);
            if (!oneofField) {
                continue;
            }
            field = oneofField;
            value = unsafeGet(init, oneofField);
        }
        else {
            field = member;
        }
        switch (field.fieldKind) {
            case "message":
                value = toMessage(field, value);
                break;
            case "scalar":
                value = initScalar(field, value);
                break;
            case "list":
                value = initList(field, value);
                break;
            case "map":
                value = initMap(field, value);
                break;
        }
        unsafeSet(message, field, value);
    }
    return message;
}
function initScalar(field, value) {
    if (field.scalar == ScalarType.BYTES) {
        return toU8Arr(value);
    }
    return value;
}
function initMap(field, value) {
    if (isObject(value)) {
        if (field.scalar == ScalarType.BYTES) {
            return convertObjectValues(value, toU8Arr);
        }
        if (field.mapKind == "message") {
            return convertObjectValues(value, (val) => toMessage(field, val));
        }
    }
    return value;
}
function initList(field, value) {
    if (Array.isArray(value)) {
        if (field.scalar == ScalarType.BYTES) {
            return value.map(toU8Arr);
        }
        if (field.listKind == "message") {
            return value.map((item) => toMessage(field, item));
        }
    }
    return value;
}
function toMessage(field, value) {
    if (field.fieldKind == "message" &&
        !field.oneof &&
        isWrapperDesc(field.message)) {
        // Types from google/protobuf/wrappers.proto are unwrapped when used in
        // a singular field that is not part of a oneof group.
        return initScalar(field.message.fields[0], value);
    }
    if (isObject(value)) {
        if (field.message.typeName == "google.protobuf.Struct" &&
            field.parent.typeName !== "google.protobuf.Value") {
            // google.protobuf.Struct is represented with JsonObject when used in a
            // field, except when used in google.protobuf.Value.
            return value;
        }
        if (!isMessage(value, field.message)) {
            return create(field.message, value);
        }
    }
    return value;
}
// converts any ArrayLike<number> to Uint8Array if necessary.
function toU8Arr(value) {
    return Array.isArray(value) ? new Uint8Array(value) : value;
}
function convertObjectValues(obj, fn) {
    const ret = {};
    for (const entry of Object.entries(obj)) {
        ret[entry[0]] = fn(entry[1]);
    }
    return ret;
}
const tokenZeroMessageField = Symbol();
const messagePrototypes = new WeakMap();
/**
 * Create a zero message.
 */
function createZeroMessage(desc) {
    let msg;
    if (!needsPrototypeChain(desc)) {
        msg = {
            $typeName: desc.typeName,
        };
        for (const member of desc.members) {
            if (member.kind == "oneof" || member.presence == IMPLICIT$2) {
                msg[member.localName] = createZeroField(member);
            }
        }
    }
    else {
        // Support default values and track presence via the prototype chain
        const cached = messagePrototypes.get(desc);
        let prototype;
        let members;
        if (cached) {
            ({ prototype, members } = cached);
        }
        else {
            prototype = {};
            members = new Set();
            for (const member of desc.members) {
                if (member.kind == "oneof") {
                    // we can only put immutable values on the prototype,
                    // oneof ADTs are mutable
                    continue;
                }
                if (member.fieldKind != "scalar" && member.fieldKind != "enum") {
                    // only scalar and enum values are immutable, map, list, and message
                    // are not
                    continue;
                }
                if (member.presence == IMPLICIT$2) {
                    // implicit presence tracks field presence by zero values - e.g. 0, false, "", are unset, 1, true, "x" are set.
                    // message, map, list fields are mutable, and also have IMPLICIT presence.
                    continue;
                }
                members.add(member);
                prototype[member.localName] = createZeroField(member);
            }
            messagePrototypes.set(desc, { prototype, members });
        }
        msg = Object.create(prototype);
        msg.$typeName = desc.typeName;
        for (const member of desc.members) {
            if (members.has(member)) {
                continue;
            }
            if (member.kind == "field") {
                if (member.fieldKind == "message") {
                    continue;
                }
                if (member.fieldKind == "scalar" || member.fieldKind == "enum") {
                    if (member.presence != IMPLICIT$2) {
                        continue;
                    }
                }
            }
            msg[member.localName] = createZeroField(member);
        }
    }
    return msg;
}
/**
 * Do we need the prototype chain to track field presence?
 */
function needsPrototypeChain(desc) {
    switch (desc.file.edition) {
        case EDITION_PROTO3$1:
            // proto3 always uses implicit presence, we never need the prototype chain.
            return false;
        case EDITION_PROTO2$1:
            // proto2 never uses implicit presence, we always need the prototype chain.
            return true;
        default:
            // If a message uses scalar or enum fields with explicit presence, we need
            // the prototype chain to track presence. This rule does not apply to fields
            // in a oneof group - they use a different mechanism to track presence.
            return desc.fields.some((f) => f.presence != IMPLICIT$2 && f.fieldKind != "message" && !f.oneof);
    }
}
/**
 * Returns a zero value for oneof groups, and for every field kind except
 * messages. Scalar and enum fields can have default values.
 */
function createZeroField(field) {
    if (field.kind == "oneof") {
        return { case: undefined };
    }
    if (field.fieldKind == "list") {
        return [];
    }
    if (field.fieldKind == "map") {
        return {}; // Object.create(null) would be desirable here, but is unsupported by react https://react.dev/reference/react/use-server#serializable-parameters-and-return-values
    }
    if (field.fieldKind == "message") {
        return tokenZeroMessageField;
    }
    const defaultValue = field.getDefaultValue();
    if (defaultValue !== undefined) {
        return field.fieldKind == "scalar" && field.longAsString
            ? defaultValue.toString()
            : defaultValue;
    }
    return field.fieldKind == "scalar"
        ? scalarZeroValue(field.scalar, field.longAsString)
        : field.enum.values[0].number;
}

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
const errorNames = [
    "FieldValueInvalidError",
    "FieldListRangeError",
    "ForeignFieldError",
];
class FieldError extends Error {
    constructor(fieldOrOneof, message, name = "FieldValueInvalidError") {
        super(message);
        this.name = name;
        this.field = () => fieldOrOneof;
    }
}
function isFieldError(arg) {
    return (arg instanceof Error &&
        errorNames.includes(arg.name) &&
        "field" in arg &&
        typeof arg.field == "function");
}

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
const symbol = Symbol.for("@bufbuild/protobuf/text-encoding");
function getTextEncoding() {
    if (globalThis[symbol] == undefined) {
        const te = new globalThis.TextEncoder();
        const td = new globalThis.TextDecoder();
        globalThis[symbol] = {
            encodeUtf8(text) {
                return te.encode(text);
            },
            decodeUtf8(bytes) {
                return td.decode(bytes);
            },
            checkUtf8(text) {
                try {
                    encodeURIComponent(text);
                    return true;
                }
                catch (_) {
                    return false;
                }
            },
        };
    }
    return globalThis[symbol];
}

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Protobuf binary format wire types.
 *
 * A wire type provides just enough information to find the length of the
 * following value.
 *
 * See https://developers.google.com/protocol-buffers/docs/encoding#structure
 */
var WireType;
(function (WireType) {
    /**
     * Used for int32, int64, uint32, uint64, sint32, sint64, bool, enum
     */
    WireType[WireType["Varint"] = 0] = "Varint";
    /**
     * Used for fixed64, sfixed64, double.
     * Always 8 bytes with little-endian byte order.
     */
    WireType[WireType["Bit64"] = 1] = "Bit64";
    /**
     * Used for string, bytes, embedded messages, packed repeated fields
     *
     * Only repeated numeric types (types which use the varint, 32-bit,
     * or 64-bit wire types) can be packed. In proto3, such fields are
     * packed by default.
     */
    WireType[WireType["LengthDelimited"] = 2] = "LengthDelimited";
    /**
     * Start of a tag-delimited aggregate, such as a proto2 group, or a message
     * in editions with message_encoding = DELIMITED.
     */
    WireType[WireType["StartGroup"] = 3] = "StartGroup";
    /**
     * End of a tag-delimited aggregate.
     */
    WireType[WireType["EndGroup"] = 4] = "EndGroup";
    /**
     * Used for fixed32, sfixed32, float.
     * Always 4 bytes with little-endian byte order.
     */
    WireType[WireType["Bit32"] = 5] = "Bit32";
})(WireType || (WireType = {}));
/**
 * Maximum value for a 32-bit floating point value (Protobuf FLOAT).
 */
const FLOAT32_MAX = 3.4028234663852886e38;
/**
 * Minimum value for a 32-bit floating point value (Protobuf FLOAT).
 */
const FLOAT32_MIN = -34028234663852886e22;
/**
 * Maximum value for an unsigned 32-bit integer (Protobuf UINT32, FIXED32).
 */
const UINT32_MAX = 0xffffffff;
/**
 * Maximum value for a signed 32-bit integer (Protobuf INT32, SFIXED32, SINT32).
 */
const INT32_MAX = 0x7fffffff;
/**
 * Minimum value for a signed 32-bit integer (Protobuf INT32, SFIXED32, SINT32).
 */
const INT32_MIN = -2147483648;
class BinaryWriter {
    constructor(encodeUtf8 = getTextEncoding().encodeUtf8) {
        this.encodeUtf8 = encodeUtf8;
        /**
         * Previous fork states.
         */
        this.stack = [];
        this.chunks = [];
        this.buf = [];
    }
    /**
     * Return all bytes written and reset this writer.
     */
    finish() {
        if (this.buf.length) {
            this.chunks.push(new Uint8Array(this.buf)); // flush the buffer
            this.buf = [];
        }
        let len = 0;
        for (let i = 0; i < this.chunks.length; i++)
            len += this.chunks[i].length;
        let bytes = new Uint8Array(len);
        let offset = 0;
        for (let i = 0; i < this.chunks.length; i++) {
            bytes.set(this.chunks[i], offset);
            offset += this.chunks[i].length;
        }
        this.chunks = [];
        return bytes;
    }
    /**
     * Start a new fork for length-delimited data like a message
     * or a packed repeated field.
     *
     * Must be joined later with `join()`.
     */
    fork() {
        this.stack.push({ chunks: this.chunks, buf: this.buf });
        this.chunks = [];
        this.buf = [];
        return this;
    }
    /**
     * Join the last fork. Write its length and bytes, then
     * return to the previous state.
     */
    join() {
        // get chunk of fork
        let chunk = this.finish();
        // restore previous state
        let prev = this.stack.pop();
        if (!prev)
            throw new Error("invalid state, fork stack empty");
        this.chunks = prev.chunks;
        this.buf = prev.buf;
        // write length of chunk as varint
        this.uint32(chunk.byteLength);
        return this.raw(chunk);
    }
    /**
     * Writes a tag (field number and wire type).
     *
     * Equivalent to `uint32( (fieldNo << 3 | type) >>> 0 )`.
     *
     * Generated code should compute the tag ahead of time and call `uint32()`.
     */
    tag(fieldNo, type) {
        return this.uint32(((fieldNo << 3) | type) >>> 0);
    }
    /**
     * Write a chunk of raw bytes.
     */
    raw(chunk) {
        if (this.buf.length) {
            this.chunks.push(new Uint8Array(this.buf));
            this.buf = [];
        }
        this.chunks.push(chunk);
        return this;
    }
    /**
     * Write a `uint32` value, an unsigned 32 bit varint.
     */
    uint32(value) {
        assertUInt32(value);
        // write value as varint 32, inlined for speed
        while (value > 0x7f) {
            this.buf.push((value & 0x7f) | 0x80);
            value = value >>> 7;
        }
        this.buf.push(value);
        return this;
    }
    /**
     * Write a `int32` value, a signed 32 bit varint.
     */
    int32(value) {
        assertInt32(value);
        varint32write(value, this.buf);
        return this;
    }
    /**
     * Write a `bool` value, a variant.
     */
    bool(value) {
        this.buf.push(value ? 1 : 0);
        return this;
    }
    /**
     * Write a `bytes` value, length-delimited arbitrary data.
     */
    bytes(value) {
        this.uint32(value.byteLength); // write length of chunk as varint
        return this.raw(value);
    }
    /**
     * Write a `string` value, length-delimited data converted to UTF-8 text.
     */
    string(value) {
        let chunk = this.encodeUtf8(value);
        this.uint32(chunk.byteLength); // write length of chunk as varint
        return this.raw(chunk);
    }
    /**
     * Write a `float` value, 32-bit floating point number.
     */
    float(value) {
        assertFloat32(value);
        let chunk = new Uint8Array(4);
        new DataView(chunk.buffer).setFloat32(0, value, true);
        return this.raw(chunk);
    }
    /**
     * Write a `double` value, a 64-bit floating point number.
     */
    double(value) {
        let chunk = new Uint8Array(8);
        new DataView(chunk.buffer).setFloat64(0, value, true);
        return this.raw(chunk);
    }
    /**
     * Write a `fixed32` value, an unsigned, fixed-length 32-bit integer.
     */
    fixed32(value) {
        assertUInt32(value);
        let chunk = new Uint8Array(4);
        new DataView(chunk.buffer).setUint32(0, value, true);
        return this.raw(chunk);
    }
    /**
     * Write a `sfixed32` value, a signed, fixed-length 32-bit integer.
     */
    sfixed32(value) {
        assertInt32(value);
        let chunk = new Uint8Array(4);
        new DataView(chunk.buffer).setInt32(0, value, true);
        return this.raw(chunk);
    }
    /**
     * Write a `sint32` value, a signed, zigzag-encoded 32-bit varint.
     */
    sint32(value) {
        assertInt32(value);
        // zigzag encode
        value = ((value << 1) ^ (value >> 31)) >>> 0;
        varint32write(value, this.buf);
        return this;
    }
    /**
     * Write a `fixed64` value, a signed, fixed-length 64-bit integer.
     */
    sfixed64(value) {
        let chunk = new Uint8Array(8), view = new DataView(chunk.buffer), tc = protoInt64.enc(value);
        view.setInt32(0, tc.lo, true);
        view.setInt32(4, tc.hi, true);
        return this.raw(chunk);
    }
    /**
     * Write a `fixed64` value, an unsigned, fixed-length 64 bit integer.
     */
    fixed64(value) {
        let chunk = new Uint8Array(8), view = new DataView(chunk.buffer), tc = protoInt64.uEnc(value);
        view.setInt32(0, tc.lo, true);
        view.setInt32(4, tc.hi, true);
        return this.raw(chunk);
    }
    /**
     * Write a `int64` value, a signed 64-bit varint.
     */
    int64(value) {
        let tc = protoInt64.enc(value);
        varint64write(tc.lo, tc.hi, this.buf);
        return this;
    }
    /**
     * Write a `sint64` value, a signed, zig-zag-encoded 64-bit varint.
     */
    sint64(value) {
        const tc = protoInt64.enc(value), 
        // zigzag encode
        sign = tc.hi >> 31, lo = (tc.lo << 1) ^ sign, hi = ((tc.hi << 1) | (tc.lo >>> 31)) ^ sign;
        varint64write(lo, hi, this.buf);
        return this;
    }
    /**
     * Write a `uint64` value, an unsigned 64-bit varint.
     */
    uint64(value) {
        const tc = protoInt64.uEnc(value);
        varint64write(tc.lo, tc.hi, this.buf);
        return this;
    }
}
class BinaryReader {
    constructor(buf, decodeUtf8 = getTextEncoding().decodeUtf8) {
        this.decodeUtf8 = decodeUtf8;
        this.varint64 = varint64read; // dirty cast for `this`
        /**
         * Read a `uint32` field, an unsigned 32 bit varint.
         */
        this.uint32 = varint32read;
        this.buf = buf;
        this.len = buf.length;
        this.pos = 0;
        this.view = new DataView(buf.buffer, buf.byteOffset, buf.byteLength);
    }
    /**
     * Reads a tag - field number and wire type.
     */
    tag() {
        let tag = this.uint32(), fieldNo = tag >>> 3, wireType = tag & 7;
        if (fieldNo <= 0 || wireType < 0 || wireType > 5)
            throw new Error("illegal tag: field no " + fieldNo + " wire type " + wireType);
        return [fieldNo, wireType];
    }
    /**
     * Skip one element and return the skipped data.
     *
     * When skipping StartGroup, provide the tags field number to check for
     * matching field number in the EndGroup tag.
     */
    skip(wireType, fieldNo) {
        let start = this.pos;
        switch (wireType) {
            case WireType.Varint:
                while (this.buf[this.pos++] & 0x80) {
                    // ignore
                }
                break;
            // @ts-ignore TS7029: Fallthrough case in switch -- ignore instead of expect-error for compiler settings without noFallthroughCasesInSwitch: true
            case WireType.Bit64:
                this.pos += 4;
            case WireType.Bit32:
                this.pos += 4;
                break;
            case WireType.LengthDelimited:
                let len = this.uint32();
                this.pos += len;
                break;
            case WireType.StartGroup:
                for (;;) {
                    const [fn, wt] = this.tag();
                    if (wt === WireType.EndGroup) {
                        if (fieldNo !== undefined && fn !== fieldNo) {
                            throw new Error("invalid end group tag");
                        }
                        break;
                    }
                    this.skip(wt, fn);
                }
                break;
            default:
                throw new Error("cant skip wire type " + wireType);
        }
        this.assertBounds();
        return this.buf.subarray(start, this.pos);
    }
    /**
     * Throws error if position in byte array is out of range.
     */
    assertBounds() {
        if (this.pos > this.len)
            throw new RangeError("premature EOF");
    }
    /**
     * Read a `int32` field, a signed 32 bit varint.
     */
    int32() {
        return this.uint32() | 0;
    }
    /**
     * Read a `sint32` field, a signed, zigzag-encoded 32-bit varint.
     */
    sint32() {
        let zze = this.uint32();
        // decode zigzag
        return (zze >>> 1) ^ -(zze & 1);
    }
    /**
     * Read a `int64` field, a signed 64-bit varint.
     */
    int64() {
        return protoInt64.dec(...this.varint64());
    }
    /**
     * Read a `uint64` field, an unsigned 64-bit varint.
     */
    uint64() {
        return protoInt64.uDec(...this.varint64());
    }
    /**
     * Read a `sint64` field, a signed, zig-zag-encoded 64-bit varint.
     */
    sint64() {
        let [lo, hi] = this.varint64();
        // decode zig zag
        let s = -(lo & 1);
        lo = ((lo >>> 1) | ((hi & 1) << 31)) ^ s;
        hi = (hi >>> 1) ^ s;
        return protoInt64.dec(lo, hi);
    }
    /**
     * Read a `bool` field, a variant.
     */
    bool() {
        let [lo, hi] = this.varint64();
        return lo !== 0 || hi !== 0;
    }
    /**
     * Read a `fixed32` field, an unsigned, fixed-length 32-bit integer.
     */
    fixed32() {
        // biome-ignore lint/suspicious/noAssignInExpressions: no
        return this.view.getUint32((this.pos += 4) - 4, true);
    }
    /**
     * Read a `sfixed32` field, a signed, fixed-length 32-bit integer.
     */
    sfixed32() {
        // biome-ignore lint/suspicious/noAssignInExpressions: no
        return this.view.getInt32((this.pos += 4) - 4, true);
    }
    /**
     * Read a `fixed64` field, an unsigned, fixed-length 64 bit integer.
     */
    fixed64() {
        return protoInt64.uDec(this.sfixed32(), this.sfixed32());
    }
    /**
     * Read a `fixed64` field, a signed, fixed-length 64-bit integer.
     */
    sfixed64() {
        return protoInt64.dec(this.sfixed32(), this.sfixed32());
    }
    /**
     * Read a `float` field, 32-bit floating point number.
     */
    float() {
        // biome-ignore lint/suspicious/noAssignInExpressions: no
        return this.view.getFloat32((this.pos += 4) - 4, true);
    }
    /**
     * Read a `double` field, a 64-bit floating point number.
     */
    double() {
        // biome-ignore lint/suspicious/noAssignInExpressions: no
        return this.view.getFloat64((this.pos += 8) - 8, true);
    }
    /**
     * Read a `bytes` field, length-delimited arbitrary data.
     */
    bytes() {
        let len = this.uint32(), start = this.pos;
        this.pos += len;
        this.assertBounds();
        return this.buf.subarray(start, start + len);
    }
    /**
     * Read a `string` field, length-delimited data converted to UTF-8 text.
     */
    string() {
        return this.decodeUtf8(this.bytes());
    }
}
/**
 * Assert a valid signed protobuf 32-bit integer as a number or string.
 */
function assertInt32(arg) {
    if (typeof arg == "string") {
        arg = Number(arg);
    }
    else if (typeof arg != "number") {
        throw new Error("invalid int32: " + typeof arg);
    }
    if (!Number.isInteger(arg) ||
        arg > INT32_MAX ||
        arg < INT32_MIN)
        throw new Error("invalid int32: " + arg);
}
/**
 * Assert a valid unsigned protobuf 32-bit integer as a number or string.
 */
function assertUInt32(arg) {
    if (typeof arg == "string") {
        arg = Number(arg);
    }
    else if (typeof arg != "number") {
        throw new Error("invalid uint32: " + typeof arg);
    }
    if (!Number.isInteger(arg) ||
        arg > UINT32_MAX ||
        arg < 0)
        throw new Error("invalid uint32: " + arg);
}
/**
 * Assert a valid protobuf float value as a number or string.
 */
function assertFloat32(arg) {
    if (typeof arg == "string") {
        const o = arg;
        arg = Number(arg);
        if (Number.isNaN(arg) && o !== "NaN") {
            throw new Error("invalid float32: " + o);
        }
    }
    else if (typeof arg != "number") {
        throw new Error("invalid float32: " + typeof arg);
    }
    if (Number.isFinite(arg) &&
        (arg > FLOAT32_MAX || arg < FLOAT32_MIN))
        throw new Error("invalid float32: " + arg);
}

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Check whether the given field value is valid for the reflect API.
 */
function checkField(field, value) {
    const check = field.fieldKind == "list"
        ? isReflectList(value, field)
        : field.fieldKind == "map"
            ? isReflectMap(value, field)
            : checkSingular(field, value);
    if (check === true) {
        return undefined;
    }
    let reason;
    switch (field.fieldKind) {
        case "list":
            reason = `expected ${formatReflectList(field)}, got ${formatVal(value)}`;
            break;
        case "map":
            reason = `expected ${formatReflectMap(field)}, got ${formatVal(value)}`;
            break;
        default: {
            reason = reasonSingular(field, value, check);
        }
    }
    return new FieldError(field, reason);
}
/**
 * Check whether the given list item is valid for the reflect API.
 */
function checkListItem(field, index, value) {
    const check = checkSingular(field, value);
    if (check !== true) {
        return new FieldError(field, `list item #${index + 1}: ${reasonSingular(field, value, check)}`);
    }
    return undefined;
}
/**
 * Check whether the given map key and value are valid for the reflect API.
 */
function checkMapEntry(field, key, value) {
    const checkKey = checkScalarValue(key, field.mapKey);
    if (checkKey !== true) {
        return new FieldError(field, `invalid map key: ${reasonSingular({ scalar: field.mapKey }, key, checkKey)}`);
    }
    const checkVal = checkSingular(field, value);
    if (checkVal !== true) {
        return new FieldError(field, `map entry ${formatVal(key)}: ${reasonSingular(field, value, checkVal)}`);
    }
    return undefined;
}
function checkSingular(field, value) {
    if (field.scalar !== undefined) {
        return checkScalarValue(value, field.scalar);
    }
    if (field.enum !== undefined) {
        if (field.enum.open) {
            return Number.isInteger(value);
        }
        return field.enum.values.some((v) => v.number === value);
    }
    return isReflectMessage(value, field.message);
}
function checkScalarValue(value, scalar) {
    switch (scalar) {
        case ScalarType.DOUBLE:
            return typeof value == "number";
        case ScalarType.FLOAT:
            if (typeof value != "number") {
                return false;
            }
            if (Number.isNaN(value) || !Number.isFinite(value)) {
                return true;
            }
            if (value > FLOAT32_MAX || value < FLOAT32_MIN) {
                return `${value.toFixed()} out of range`;
            }
            return true;
        case ScalarType.INT32:
        case ScalarType.SFIXED32:
        case ScalarType.SINT32:
            // signed
            if (typeof value !== "number" || !Number.isInteger(value)) {
                return false;
            }
            if (value > INT32_MAX || value < INT32_MIN) {
                return `${value.toFixed()} out of range`;
            }
            return true;
        case ScalarType.FIXED32:
        case ScalarType.UINT32:
            // unsigned
            if (typeof value !== "number" || !Number.isInteger(value)) {
                return false;
            }
            if (value > UINT32_MAX || value < 0) {
                return `${value.toFixed()} out of range`;
            }
            return true;
        case ScalarType.BOOL:
            return typeof value == "boolean";
        case ScalarType.STRING:
            if (typeof value != "string") {
                return false;
            }
            return getTextEncoding().checkUtf8(value) || "invalid UTF8";
        case ScalarType.BYTES:
            return value instanceof Uint8Array;
        case ScalarType.INT64:
        case ScalarType.SFIXED64:
        case ScalarType.SINT64:
            // signed
            if (typeof value == "bigint" ||
                typeof value == "number" ||
                (typeof value == "string" && value.length > 0)) {
                try {
                    protoInt64.parse(value);
                    return true;
                }
                catch (_) {
                    return `${value} out of range`;
                }
            }
            return false;
        case ScalarType.FIXED64:
        case ScalarType.UINT64:
            // unsigned
            if (typeof value == "bigint" ||
                typeof value == "number" ||
                (typeof value == "string" && value.length > 0)) {
                try {
                    protoInt64.uParse(value);
                    return true;
                }
                catch (_) {
                    return `${value} out of range`;
                }
            }
            return false;
    }
}
function reasonSingular(field, val, details) {
    details =
        typeof details == "string" ? `: ${details}` : `, got ${formatVal(val)}`;
    if (field.scalar !== undefined) {
        return `expected ${scalarTypeDescription(field.scalar)}` + details;
    }
    if (field.enum !== undefined) {
        return `expected ${field.enum.toString()}` + details;
    }
    return `expected ${formatReflectMessage(field.message)}` + details;
}
function formatVal(val) {
    switch (typeof val) {
        case "object":
            if (val === null) {
                return "null";
            }
            if (val instanceof Uint8Array) {
                return `Uint8Array(${val.length})`;
            }
            if (Array.isArray(val)) {
                return `Array(${val.length})`;
            }
            if (isReflectList(val)) {
                return formatReflectList(val.field());
            }
            if (isReflectMap(val)) {
                return formatReflectMap(val.field());
            }
            if (isReflectMessage(val)) {
                return formatReflectMessage(val.desc);
            }
            if (isMessage(val)) {
                return `message ${val.$typeName}`;
            }
            return "object";
        case "string":
            return val.length > 30 ? "string" : `"${val.split('"').join('\\"')}"`;
        case "boolean":
            return String(val);
        case "number":
            return String(val);
        case "bigint":
            return String(val) + "n";
        default:
            // "symbol" | "undefined" | "object" | "function"
            return typeof val;
    }
}
function formatReflectMessage(desc) {
    return `ReflectMessage (${desc.typeName})`;
}
function formatReflectList(field) {
    switch (field.listKind) {
        case "message":
            return `ReflectList (${field.message.toString()})`;
        case "enum":
            return `ReflectList (${field.enum.toString()})`;
        case "scalar":
            return `ReflectList (${ScalarType[field.scalar]})`;
    }
}
function formatReflectMap(field) {
    switch (field.mapKind) {
        case "message":
            return `ReflectMap (${ScalarType[field.mapKey]}, ${field.message.toString()})`;
        case "enum":
            return `ReflectMap (${ScalarType[field.mapKey]}, ${field.enum.toString()})`;
        case "scalar":
            return `ReflectMap (${ScalarType[field.mapKey]}, ${ScalarType[field.scalar]})`;
    }
}
function scalarTypeDescription(scalar) {
    switch (scalar) {
        case ScalarType.STRING:
            return "string";
        case ScalarType.BOOL:
            return "boolean";
        case ScalarType.INT64:
        case ScalarType.SINT64:
        case ScalarType.SFIXED64:
            return "bigint (int64)";
        case ScalarType.UINT64:
        case ScalarType.FIXED64:
            return "bigint (uint64)";
        case ScalarType.BYTES:
            return "Uint8Array";
        case ScalarType.DOUBLE:
            return "number (float64)";
        case ScalarType.FLOAT:
            return "number (float32)";
        case ScalarType.FIXED32:
        case ScalarType.UINT32:
            return "number (uint32)";
        case ScalarType.INT32:
        case ScalarType.SFIXED32:
        case ScalarType.SINT32:
            return "number (int32)";
    }
}

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Create a ReflectMessage.
 */
function reflect(messageDesc, message, 
/**
 * By default, field values are validated when setting them. For example,
 * a value for an uint32 field must be a ECMAScript Number >= 0.
 *
 * When field values are trusted, performance can be improved by disabling
 * checks.
 */
check = true) {
    return new ReflectMessageImpl(messageDesc, message, check);
}
class ReflectMessageImpl {
    get sortedFields() {
        var _a;
        return ((_a = this._sortedFields) !== null && _a !== void 0 ? _a : 
        // biome-ignore lint/suspicious/noAssignInExpressions: no
        (this._sortedFields = this.desc.fields
            .concat()
            .sort((a, b) => a.number - b.number)));
    }
    constructor(messageDesc, message, check = true) {
        this.lists = new Map();
        this.maps = new Map();
        this.check = check;
        this.desc = messageDesc;
        this.message = this[unsafeLocal] = message !== null && message !== void 0 ? message : create(messageDesc);
        this.fields = messageDesc.fields;
        this.oneofs = messageDesc.oneofs;
        this.members = messageDesc.members;
    }
    findNumber(number) {
        if (!this._fieldsByNumber) {
            this._fieldsByNumber = new Map(this.desc.fields.map((f) => [f.number, f]));
        }
        return this._fieldsByNumber.get(number);
    }
    oneofCase(oneof) {
        assertOwn(this.message, oneof);
        return unsafeOneofCase(this.message, oneof);
    }
    isSet(field) {
        assertOwn(this.message, field);
        return unsafeIsSet(this.message, field);
    }
    clear(field) {
        assertOwn(this.message, field);
        unsafeClear(this.message, field);
    }
    get(field) {
        assertOwn(this.message, field);
        const value = unsafeGet(this.message, field);
        switch (field.fieldKind) {
            case "list":
                // eslint-disable-next-line no-case-declarations
                let list = this.lists.get(field);
                if (!list || list[unsafeLocal] !== value) {
                    this.lists.set(field, 
                    // biome-ignore lint/suspicious/noAssignInExpressions: no
                    (list = new ReflectListImpl(field, value, this.check)));
                }
                return list;
            case "map":
                let map = this.maps.get(field);
                if (!map || map[unsafeLocal] !== value) {
                    this.maps.set(field, 
                    // biome-ignore lint/suspicious/noAssignInExpressions: no
                    (map = new ReflectMapImpl(field, value, this.check)));
                }
                return map;
            case "message":
                return messageToReflect(field, value, this.check);
            case "scalar":
                return (value === undefined
                    ? scalarZeroValue(field.scalar, false)
                    : longToReflect(field, value));
            case "enum":
                return (value !== null && value !== void 0 ? value : field.enum.values[0].number);
        }
    }
    set(field, value) {
        assertOwn(this.message, field);
        if (this.check) {
            const err = checkField(field, value);
            if (err) {
                throw err;
            }
        }
        let local;
        if (field.fieldKind == "message") {
            local = messageToLocal(field, value);
        }
        else if (isReflectMap(value) || isReflectList(value)) {
            local = value[unsafeLocal];
        }
        else {
            local = longToLocal(field, value);
        }
        unsafeSet(this.message, field, local);
    }
    getUnknown() {
        return this.message.$unknown;
    }
    setUnknown(value) {
        this.message.$unknown = value;
    }
}
function assertOwn(owner, member) {
    if (member.parent.typeName !== owner.$typeName) {
        throw new FieldError(member, `cannot use ${member.toString()} with message ${owner.$typeName}`, "ForeignFieldError");
    }
}
class ReflectListImpl {
    field() {
        return this._field;
    }
    get size() {
        return this._arr.length;
    }
    constructor(field, unsafeInput, check) {
        this._field = field;
        this._arr = this[unsafeLocal] = unsafeInput;
        this.check = check;
    }
    get(index) {
        const item = this._arr[index];
        return item === undefined
            ? undefined
            : listItemToReflect(this._field, item, this.check);
    }
    set(index, item) {
        if (index < 0 || index >= this._arr.length) {
            throw new FieldError(this._field, `list item #${index + 1}: out of range`);
        }
        if (this.check) {
            const err = checkListItem(this._field, index, item);
            if (err) {
                throw err;
            }
        }
        this._arr[index] = listItemToLocal(this._field, item);
    }
    add(item) {
        if (this.check) {
            const err = checkListItem(this._field, this._arr.length, item);
            if (err) {
                throw err;
            }
        }
        this._arr.push(listItemToLocal(this._field, item));
        return undefined;
    }
    clear() {
        this._arr.splice(0, this._arr.length);
    }
    [Symbol.iterator]() {
        return this.values();
    }
    keys() {
        return this._arr.keys();
    }
    *values() {
        for (const item of this._arr) {
            yield listItemToReflect(this._field, item, this.check);
        }
    }
    *entries() {
        for (let i = 0; i < this._arr.length; i++) {
            yield [i, listItemToReflect(this._field, this._arr[i], this.check)];
        }
    }
}
class ReflectMapImpl {
    constructor(field, unsafeInput, check = true) {
        this.obj = this[unsafeLocal] = unsafeInput !== null && unsafeInput !== void 0 ? unsafeInput : {};
        this.check = check;
        this._field = field;
    }
    field() {
        return this._field;
    }
    set(key, value) {
        if (this.check) {
            const err = checkMapEntry(this._field, key, value);
            if (err) {
                throw err;
            }
        }
        this.obj[mapKeyToLocal(key)] = mapValueToLocal(this._field, value);
        return this;
    }
    delete(key) {
        const k = mapKeyToLocal(key);
        const has = Object.prototype.hasOwnProperty.call(this.obj, k);
        if (has) {
            delete this.obj[k];
        }
        return has;
    }
    clear() {
        for (const key of Object.keys(this.obj)) {
            delete this.obj[key];
        }
    }
    get(key) {
        let val = this.obj[mapKeyToLocal(key)];
        if (val !== undefined) {
            val = mapValueToReflect(this._field, val, this.check);
        }
        return val;
    }
    has(key) {
        return Object.prototype.hasOwnProperty.call(this.obj, mapKeyToLocal(key));
    }
    *keys() {
        for (const objKey of Object.keys(this.obj)) {
            yield mapKeyToReflect(objKey, this._field.mapKey);
        }
    }
    *entries() {
        for (const objEntry of Object.entries(this.obj)) {
            yield [
                mapKeyToReflect(objEntry[0], this._field.mapKey),
                mapValueToReflect(this._field, objEntry[1], this.check),
            ];
        }
    }
    [Symbol.iterator]() {
        return this.entries();
    }
    get size() {
        return Object.keys(this.obj).length;
    }
    *values() {
        for (const val of Object.values(this.obj)) {
            yield mapValueToReflect(this._field, val, this.check);
        }
    }
    forEach(callbackfn, thisArg) {
        for (const mapEntry of this.entries()) {
            callbackfn.call(thisArg, mapEntry[1], mapEntry[0], this);
        }
    }
}
function messageToLocal(field, value) {
    if (!isReflectMessage(value)) {
        return value;
    }
    if (isWrapper(value.message) &&
        !field.oneof &&
        field.fieldKind == "message") {
        // Types from google/protobuf/wrappers.proto are unwrapped when used in
        // a singular field that is not part of a oneof group.
        return value.message.value;
    }
    if (value.desc.typeName == "google.protobuf.Struct" &&
        field.parent.typeName != "google.protobuf.Value") {
        // google.protobuf.Struct is represented with JsonObject when used in a
        // field, except when used in google.protobuf.Value.
        return wktStructToLocal(value.message);
    }
    return value.message;
}
function messageToReflect(field, value, check) {
    if (value !== undefined) {
        if (isWrapperDesc(field.message) &&
            !field.oneof &&
            field.fieldKind == "message") {
            // Types from google/protobuf/wrappers.proto are unwrapped when used in
            // a singular field that is not part of a oneof group.
            value = {
                $typeName: field.message.typeName,
                value: longToReflect(field.message.fields[0], value),
            };
        }
        else if (field.message.typeName == "google.protobuf.Struct" &&
            field.parent.typeName != "google.protobuf.Value" &&
            isObject(value)) {
            // google.protobuf.Struct is represented with JsonObject when used in a
            // field, except when used in google.protobuf.Value.
            value = wktStructToReflect(value);
        }
    }
    return new ReflectMessageImpl(field.message, value, check);
}
function listItemToLocal(field, value) {
    if (field.listKind == "message") {
        return messageToLocal(field, value);
    }
    return longToLocal(field, value);
}
function listItemToReflect(field, value, check) {
    if (field.listKind == "message") {
        return messageToReflect(field, value, check);
    }
    return longToReflect(field, value);
}
function mapValueToLocal(field, value) {
    if (field.mapKind == "message") {
        return messageToLocal(field, value);
    }
    return longToLocal(field, value);
}
function mapValueToReflect(field, value, check) {
    if (field.mapKind == "message") {
        return messageToReflect(field, value, check);
    }
    return value;
}
function mapKeyToLocal(key) {
    return typeof key == "string" || typeof key == "number" ? key : String(key);
}
/**
 * Converts a map key (any scalar value except float, double, or bytes) from its
 * representation in a message (string or number, the only possible object key
 * types) to the closest possible type in ECMAScript.
 */
function mapKeyToReflect(key, type) {
    switch (type) {
        case ScalarType.STRING:
            return key;
        case ScalarType.INT32:
        case ScalarType.FIXED32:
        case ScalarType.UINT32:
        case ScalarType.SFIXED32:
        case ScalarType.SINT32: {
            const n = Number.parseInt(key);
            if (Number.isFinite(n)) {
                return n;
            }
            break;
        }
        case ScalarType.BOOL:
            switch (key) {
                case "true":
                    return true;
                case "false":
                    return false;
            }
            break;
        case ScalarType.UINT64:
        case ScalarType.FIXED64:
            try {
                return protoInt64.uParse(key);
            }
            catch (_a) {
                //
            }
            break;
        default:
            // INT64, SFIXED64, SINT64
            try {
                return protoInt64.parse(key);
            }
            catch (_b) {
                //
            }
            break;
    }
    return key;
}
function longToReflect(field, value) {
    switch (field.scalar) {
        case ScalarType.INT64:
        case ScalarType.SFIXED64:
        case ScalarType.SINT64:
            if ("longAsString" in field &&
                field.longAsString &&
                typeof value == "string") {
                value = protoInt64.parse(value);
            }
            break;
        case ScalarType.FIXED64:
        case ScalarType.UINT64:
            if ("longAsString" in field &&
                field.longAsString &&
                typeof value == "string") {
                value = protoInt64.uParse(value);
            }
            break;
    }
    return value;
}
function longToLocal(field, value) {
    switch (field.scalar) {
        case ScalarType.INT64:
        case ScalarType.SFIXED64:
        case ScalarType.SINT64:
            if ("longAsString" in field && field.longAsString) {
                value = String(value);
            }
            else if (typeof value == "string" || typeof value == "number") {
                value = protoInt64.parse(value);
            }
            break;
        case ScalarType.FIXED64:
        case ScalarType.UINT64:
            if ("longAsString" in field && field.longAsString) {
                value = String(value);
            }
            else if (typeof value == "string" || typeof value == "number") {
                value = protoInt64.uParse(value);
            }
            break;
    }
    return value;
}
function wktStructToReflect(json) {
    const struct = {
        $typeName: "google.protobuf.Struct",
        fields: {},
    };
    if (isObject(json)) {
        for (const [k, v] of Object.entries(json)) {
            struct.fields[k] = wktValueToReflect(v);
        }
    }
    return struct;
}
function wktStructToLocal(val) {
    const json = {};
    for (const [k, v] of Object.entries(val.fields)) {
        json[k] = wktValueToLocal(v);
    }
    return json;
}
function wktValueToLocal(val) {
    switch (val.kind.case) {
        case "structValue":
            return wktStructToLocal(val.kind.value);
        case "listValue":
            return val.kind.value.values.map(wktValueToLocal);
        case "nullValue":
        case undefined:
            return null;
        default:
            return val.kind.value;
    }
}
function wktValueToReflect(json) {
    const value = {
        $typeName: "google.protobuf.Value",
        kind: { case: undefined },
    };
    switch (typeof json) {
        case "number":
            value.kind = { case: "numberValue", value: json };
            break;
        case "string":
            value.kind = { case: "stringValue", value: json };
            break;
        case "boolean":
            value.kind = { case: "boolValue", value: json };
            break;
        case "object":
            if (json === null) {
                const nullValue = 0;
                value.kind = { case: "nullValue", value: nullValue };
            }
            else if (Array.isArray(json)) {
                const listValue = {
                    $typeName: "google.protobuf.ListValue",
                    values: [],
                };
                if (Array.isArray(json)) {
                    for (const e of json) {
                        listValue.values.push(wktValueToReflect(e));
                    }
                }
                value.kind = {
                    case: "listValue",
                    value: listValue,
                };
            }
            else {
                value.kind = {
                    case: "structValue",
                    value: wktStructToReflect(json),
                };
            }
            break;
    }
    return value;
}

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Decodes a base64 string to a byte array.
 *
 * - ignores white-space, including line breaks and tabs
 * - allows inner padding (can decode concatenated base64 strings)
 * - does not require padding
 * - understands base64url encoding:
 *   "-" instead of "+",
 *   "_" instead of "/",
 *   no padding
 */
function base64Decode(base64Str) {
    const table = getDecodeTable();
    // estimate byte size, not accounting for inner padding and whitespace
    let es = (base64Str.length * 3) / 4;
    if (base64Str[base64Str.length - 2] == "=")
        es -= 2;
    else if (base64Str[base64Str.length - 1] == "=")
        es -= 1;
    let bytes = new Uint8Array(es), bytePos = 0, // position in byte array
    groupPos = 0, // position in base64 group
    b, // current byte
    p = 0; // previous byte
    for (let i = 0; i < base64Str.length; i++) {
        b = table[base64Str.charCodeAt(i)];
        if (b === undefined) {
            switch (base64Str[i]) {
                // @ts-ignore TS7029: Fallthrough case in switch -- ignore instead of expect-error for compiler settings without noFallthroughCasesInSwitch: true
                case "=":
                    groupPos = 0; // reset state when padding found
                case "\n":
                case "\r":
                case "\t":
                case " ":
                    continue; // skip white-space, and padding
                default:
                    throw Error("invalid base64 string");
            }
        }
        switch (groupPos) {
            case 0:
                p = b;
                groupPos = 1;
                break;
            case 1:
                bytes[bytePos++] = (p << 2) | ((b & 48) >> 4);
                p = b;
                groupPos = 2;
                break;
            case 2:
                bytes[bytePos++] = ((p & 15) << 4) | ((b & 60) >> 2);
                p = b;
                groupPos = 3;
                break;
            case 3:
                bytes[bytePos++] = ((p & 3) << 6) | b;
                groupPos = 0;
                break;
        }
    }
    if (groupPos == 1)
        throw Error("invalid base64 string");
    return bytes.subarray(0, bytePos);
}
/**
 * Encode a byte array to a base64 string.
 *
 * By default, this function uses the standard base64 encoding with padding.
 *
 * To encode without padding, use encoding = "std_raw".
 *
 * To encode with the URL encoding, use encoding = "url", which replaces the
 * characters +/ by their URL-safe counterparts -_, and omits padding.
 */
function base64Encode(bytes, encoding = "std") {
    const table = getEncodeTable(encoding);
    const pad = encoding == "std";
    let base64 = "", groupPos = 0, // position in base64 group
    b, // current byte
    p = 0; // carry over from previous byte
    for (let i = 0; i < bytes.length; i++) {
        b = bytes[i];
        switch (groupPos) {
            case 0:
                base64 += table[b >> 2];
                p = (b & 3) << 4;
                groupPos = 1;
                break;
            case 1:
                base64 += table[p | (b >> 4)];
                p = (b & 15) << 2;
                groupPos = 2;
                break;
            case 2:
                base64 += table[p | (b >> 6)];
                base64 += table[b & 63];
                groupPos = 0;
                break;
        }
    }
    // add output padding
    if (groupPos) {
        base64 += table[p];
        if (pad) {
            base64 += "=";
            if (groupPos == 1)
                base64 += "=";
        }
    }
    return base64;
}
// lookup table from base64 character to byte
let encodeTableStd;
let encodeTableUrl;
// lookup table from base64 character *code* to byte because lookup by number is fast
let decodeTable;
function getEncodeTable(encoding) {
    if (!encodeTableStd) {
        encodeTableStd =
            "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/".split("");
        encodeTableUrl = encodeTableStd.slice(0, -2).concat("-", "_");
    }
    return encoding == "url"
        ? // biome-ignore lint/style/noNonNullAssertion: TS fails to narrow down
            encodeTableUrl
        : encodeTableStd;
}
function getDecodeTable() {
    if (!decodeTable) {
        decodeTable = [];
        const encodeTable = getEncodeTable("std");
        for (let i = 0; i < encodeTable.length; i++)
            decodeTable[encodeTable[i].charCodeAt(0)] = i;
        // support base64url variants
        decodeTable["-".charCodeAt(0)] = encodeTable.indexOf("+");
        decodeTable["_".charCodeAt(0)] = encodeTable.indexOf("/");
    }
    return decodeTable;
}

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Return a fully-qualified name for a Protobuf descriptor.
 * For a file descriptor, return the original file path.
 *
 * See https://protobuf.com/docs/language-spec#fully-qualified-names
 */
/**
 * Converts snake_case to protoCamelCase according to the convention
 * used by protoc to convert a field name to a JSON name.
 */
function protoCamelCase(snakeCase) {
    let capNext = false;
    const b = [];
    for (let i = 0; i < snakeCase.length; i++) {
        let c = snakeCase.charAt(i);
        switch (c) {
            case "_":
                capNext = true;
                break;
            case "0":
            case "1":
            case "2":
            case "3":
            case "4":
            case "5":
            case "6":
            case "7":
            case "8":
            case "9":
                b.push(c);
                capNext = false;
                break;
            default:
                if (capNext) {
                    capNext = false;
                    c = c.toUpperCase();
                }
                b.push(c);
                break;
        }
    }
    return b.join("");
}
/**
 * Names that cannot be used for object properties because they are reserved
 * by built-in JavaScript properties.
 */
const reservedObjectProperties = new Set([
    // names reserved by JavaScript
    "constructor",
    "toString",
    "toJSON",
    "valueOf",
]);
/**
 * Escapes names that are reserved for ECMAScript built-in object properties.
 *
 * Also see safeIdentifier() from @bufbuild/protoplugin.
 */
function safeObjectProperty(name) {
    return reservedObjectProperties.has(name) ? name + "$" : name;
}

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * @private
 */
function restoreJsonNames(message) {
    for (const f of message.field) {
        if (!unsafeIsSetExplicit(f, "jsonName")) {
            f.jsonName = protoCamelCase(f.name);
        }
    }
    message.nestedType.forEach(restoreJsonNames);
}

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Parse an enum value from the Protobuf text format.
 *
 * @private
 */
function parseTextFormatEnumValue(descEnum, value) {
    const enumValue = descEnum.values.find((v) => v.name === value);
    if (!enumValue) {
        throw new Error(`cannot parse ${descEnum} default value: ${value}`);
    }
    return enumValue.number;
}
/**
 * Parse a scalar value from the Protobuf text format.
 *
 * @private
 */
function parseTextFormatScalarValue(type, value) {
    switch (type) {
        case ScalarType.STRING:
            return value;
        case ScalarType.BYTES: {
            const u = unescapeBytesDefaultValue(value);
            if (u === false) {
                throw new Error(`cannot parse ${ScalarType[type]} default value: ${value}`);
            }
            return u;
        }
        case ScalarType.INT64:
        case ScalarType.SFIXED64:
        case ScalarType.SINT64:
            return protoInt64.parse(value);
        case ScalarType.UINT64:
        case ScalarType.FIXED64:
            return protoInt64.uParse(value);
        case ScalarType.DOUBLE:
        case ScalarType.FLOAT:
            switch (value) {
                case "inf":
                    return Number.POSITIVE_INFINITY;
                case "-inf":
                    return Number.NEGATIVE_INFINITY;
                case "nan":
                    return Number.NaN;
                default:
                    return parseFloat(value);
            }
        case ScalarType.BOOL:
            return value === "true";
        case ScalarType.INT32:
        case ScalarType.UINT32:
        case ScalarType.SINT32:
        case ScalarType.FIXED32:
        case ScalarType.SFIXED32:
            return parseInt(value, 10);
    }
}
/**
 * Parses a text-encoded default value (proto2) of a BYTES field.
 */
function unescapeBytesDefaultValue(str) {
    const b = [];
    const input = {
        tail: str,
        c: "",
        next() {
            if (this.tail.length == 0) {
                return false;
            }
            this.c = this.tail[0];
            this.tail = this.tail.substring(1);
            return true;
        },
        take(n) {
            if (this.tail.length >= n) {
                const r = this.tail.substring(0, n);
                this.tail = this.tail.substring(n);
                return r;
            }
            return false;
        },
    };
    while (input.next()) {
        switch (input.c) {
            case "\\":
                if (input.next()) {
                    switch (input.c) {
                        case "\\":
                            b.push(input.c.charCodeAt(0));
                            break;
                        case "b":
                            b.push(0x08);
                            break;
                        case "f":
                            b.push(0x0c);
                            break;
                        case "n":
                            b.push(0x0a);
                            break;
                        case "r":
                            b.push(0x0d);
                            break;
                        case "t":
                            b.push(0x09);
                            break;
                        case "v":
                            b.push(0x0b);
                            break;
                        case "0":
                        case "1":
                        case "2":
                        case "3":
                        case "4":
                        case "5":
                        case "6":
                        case "7": {
                            const s = input.c;
                            const t = input.take(2);
                            if (t === false) {
                                return false;
                            }
                            const n = parseInt(s + t, 8);
                            if (Number.isNaN(n)) {
                                return false;
                            }
                            b.push(n);
                            break;
                        }
                        case "x": {
                            const s = input.c;
                            const t = input.take(2);
                            if (t === false) {
                                return false;
                            }
                            const n = parseInt(s + t, 16);
                            if (Number.isNaN(n)) {
                                return false;
                            }
                            b.push(n);
                            break;
                        }
                        case "u": {
                            const s = input.c;
                            const t = input.take(4);
                            if (t === false) {
                                return false;
                            }
                            const n = parseInt(s + t, 16);
                            if (Number.isNaN(n)) {
                                return false;
                            }
                            const chunk = new Uint8Array(4);
                            const view = new DataView(chunk.buffer);
                            view.setInt32(0, n, true);
                            b.push(chunk[0], chunk[1], chunk[2], chunk[3]);
                            break;
                        }
                        case "U": {
                            const s = input.c;
                            const t = input.take(8);
                            if (t === false) {
                                return false;
                            }
                            const tc = protoInt64.uEnc(s + t);
                            const chunk = new Uint8Array(8);
                            const view = new DataView(chunk.buffer);
                            view.setInt32(0, tc.lo, true);
                            view.setInt32(4, tc.hi, true);
                            b.push(chunk[0], chunk[1], chunk[2], chunk[3], chunk[4], chunk[5], chunk[6], chunk[7]);
                            break;
                        }
                    }
                }
                break;
            default:
                b.push(input.c.charCodeAt(0));
        }
    }
    return new Uint8Array(b);
}

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Iterate over all types - enumerations, extensions, services, messages -
 * and enumerations, extensions and messages nested in messages.
 */
function* nestedTypes(desc) {
    switch (desc.kind) {
        case "file":
            for (const message of desc.messages) {
                yield message;
                yield* nestedTypes(message);
            }
            yield* desc.enums;
            yield* desc.services;
            yield* desc.extensions;
            break;
        case "message":
            for (const message of desc.nestedMessages) {
                yield message;
                yield* nestedTypes(message);
            }
            yield* desc.nestedEnums;
            yield* desc.nestedExtensions;
            break;
    }
}

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
function createFileRegistry(...args) {
    const registry = createBaseRegistry();
    if (!args.length) {
        return registry;
    }
    if ("$typeName" in args[0] &&
        args[0].$typeName == "google.protobuf.FileDescriptorSet") {
        for (const file of args[0].file) {
            addFile(file, registry);
        }
        return registry;
    }
    if ("$typeName" in args[0]) {
        const input = args[0];
        const resolve = args[1];
        const seen = new Set();
        function recurseDeps(file) {
            const deps = [];
            for (const protoFileName of file.dependency) {
                if (registry.getFile(protoFileName) != undefined) {
                    continue;
                }
                if (seen.has(protoFileName)) {
                    continue;
                }
                const dep = resolve(protoFileName);
                if (!dep) {
                    throw new Error(`Unable to resolve ${protoFileName}, imported by ${file.name}`);
                }
                if ("kind" in dep) {
                    registry.addFile(dep, false, true);
                }
                else {
                    seen.add(dep.name);
                    deps.push(dep);
                }
            }
            return deps.concat(...deps.map(recurseDeps));
        }
        for (const file of [input, ...recurseDeps(input)].reverse()) {
            addFile(file, registry);
        }
    }
    else {
        for (const fileReg of args) {
            for (const file of fileReg.files) {
                registry.addFile(file);
            }
        }
    }
    return registry;
}
/**
 * @private
 */
function createBaseRegistry() {
    const types = new Map();
    const extendees = new Map();
    const files = new Map();
    return {
        kind: "registry",
        types,
        extendees,
        [Symbol.iterator]() {
            return types.values();
        },
        get files() {
            return files.values();
        },
        addFile(file, skipTypes, withDeps) {
            files.set(file.proto.name, file);
            if (!skipTypes) {
                for (const type of nestedTypes(file)) {
                    this.add(type);
                }
            }
            if (withDeps) {
                for (const f of file.dependencies) {
                    this.addFile(f, skipTypes, withDeps);
                }
            }
        },
        add(desc) {
            if (desc.kind == "extension") {
                let numberToExt = extendees.get(desc.extendee.typeName);
                if (!numberToExt) {
                    extendees.set(desc.extendee.typeName, 
                    // biome-ignore lint/suspicious/noAssignInExpressions: no
                    (numberToExt = new Map()));
                }
                numberToExt.set(desc.number, desc);
            }
            types.set(desc.typeName, desc);
        },
        get(typeName) {
            return types.get(typeName);
        },
        getFile(fileName) {
            return files.get(fileName);
        },
        getMessage(typeName) {
            const t = types.get(typeName);
            return (t === null || t === void 0 ? void 0 : t.kind) == "message" ? t : undefined;
        },
        getEnum(typeName) {
            const t = types.get(typeName);
            return (t === null || t === void 0 ? void 0 : t.kind) == "enum" ? t : undefined;
        },
        getExtension(typeName) {
            const t = types.get(typeName);
            return (t === null || t === void 0 ? void 0 : t.kind) == "extension" ? t : undefined;
        },
        getExtensionFor(extendee, no) {
            var _a;
            return (_a = extendees.get(extendee.typeName)) === null || _a === void 0 ? void 0 : _a.get(no);
        },
        getService(typeName) {
            const t = types.get(typeName);
            return (t === null || t === void 0 ? void 0 : t.kind) == "service" ? t : undefined;
        },
    };
}
// bootstrap-inject google.protobuf.Edition.EDITION_PROTO2: const $name: Edition.$localName = $number;
const EDITION_PROTO2 = 998;
// bootstrap-inject google.protobuf.Edition.EDITION_PROTO3: const $name: Edition.$localName = $number;
const EDITION_PROTO3 = 999;
// bootstrap-inject google.protobuf.FieldDescriptorProto.Type.TYPE_STRING: const $name: FieldDescriptorProto_Type.$localName = $number;
const TYPE_STRING = 9;
// bootstrap-inject google.protobuf.FieldDescriptorProto.Type.TYPE_GROUP: const $name: FieldDescriptorProto_Type.$localName = $number;
const TYPE_GROUP = 10;
// bootstrap-inject google.protobuf.FieldDescriptorProto.Type.TYPE_MESSAGE: const $name: FieldDescriptorProto_Type.$localName = $number;
const TYPE_MESSAGE = 11;
// bootstrap-inject google.protobuf.FieldDescriptorProto.Type.TYPE_BYTES: const $name: FieldDescriptorProto_Type.$localName = $number;
const TYPE_BYTES = 12;
// bootstrap-inject google.protobuf.FieldDescriptorProto.Type.TYPE_ENUM: const $name: FieldDescriptorProto_Type.$localName = $number;
const TYPE_ENUM = 14;
// bootstrap-inject google.protobuf.FieldDescriptorProto.Label.LABEL_REPEATED: const $name: FieldDescriptorProto_Label.$localName = $number;
const LABEL_REPEATED = 3;
// bootstrap-inject google.protobuf.FieldDescriptorProto.Label.LABEL_REQUIRED: const $name: FieldDescriptorProto_Label.$localName = $number;
const LABEL_REQUIRED = 2;
// bootstrap-inject google.protobuf.FieldOptions.JSType.JS_STRING: const $name: FieldOptions_JSType.$localName = $number;
const JS_STRING = 1;
// bootstrap-inject google.protobuf.MethodOptions.IdempotencyLevel.IDEMPOTENCY_UNKNOWN: const $name: MethodOptions_IdempotencyLevel.$localName = $number;
const IDEMPOTENCY_UNKNOWN = 0;
// bootstrap-inject google.protobuf.FeatureSet.FieldPresence.EXPLICIT: const $name: FeatureSet_FieldPresence.$localName = $number;
const EXPLICIT = 1;
// bootstrap-inject google.protobuf.FeatureSet.FieldPresence.IMPLICIT: const $name: FeatureSet_FieldPresence.$localName = $number;
const IMPLICIT$1 = 2;
// bootstrap-inject google.protobuf.FeatureSet.FieldPresence.LEGACY_REQUIRED: const $name: FeatureSet_FieldPresence.$localName = $number;
const LEGACY_REQUIRED$2 = 3;
// bootstrap-inject google.protobuf.FeatureSet.RepeatedFieldEncoding.PACKED: const $name: FeatureSet_RepeatedFieldEncoding.$localName = $number;
const PACKED = 1;
// bootstrap-inject google.protobuf.FeatureSet.MessageEncoding.DELIMITED: const $name: FeatureSet_MessageEncoding.$localName = $number;
const DELIMITED = 2;
// bootstrap-inject google.protobuf.FeatureSet.EnumType.OPEN: const $name: FeatureSet_EnumType.$localName = $number;
const OPEN = 1;
const featureDefaults = {
    // EDITION_PROTO2
    998: {
        fieldPresence: 1, // EXPLICIT,
        enumType: 2, // CLOSED,
        repeatedFieldEncoding: 2, // EXPANDED,
        utf8Validation: 3, // NONE,
        messageEncoding: 1, // LENGTH_PREFIXED,
        jsonFormat: 2, // LEGACY_BEST_EFFORT,
        enforceNamingStyle: 2, // STYLE_LEGACY,
        defaultSymbolVisibility: 1, // EXPORT_ALL,
    },
    // EDITION_PROTO3
    999: {
        fieldPresence: 2, // IMPLICIT,
        enumType: 1, // OPEN,
        repeatedFieldEncoding: 1, // PACKED,
        utf8Validation: 2, // VERIFY,
        messageEncoding: 1, // LENGTH_PREFIXED,
        jsonFormat: 1, // ALLOW,
        enforceNamingStyle: 2, // STYLE_LEGACY,
        defaultSymbolVisibility: 1, // EXPORT_ALL,
    },
    // EDITION_2023
    1000: {
        fieldPresence: 1, // EXPLICIT,
        enumType: 1, // OPEN,
        repeatedFieldEncoding: 1, // PACKED,
        utf8Validation: 2, // VERIFY,
        messageEncoding: 1, // LENGTH_PREFIXED,
        jsonFormat: 1, // ALLOW,
        enforceNamingStyle: 2, // STYLE_LEGACY,
        defaultSymbolVisibility: 1, // EXPORT_ALL,
    },
    // EDITION_2024
    1001: {
        fieldPresence: 1, // EXPLICIT,
        enumType: 1, // OPEN,
        repeatedFieldEncoding: 1, // PACKED,
        utf8Validation: 2, // VERIFY,
        messageEncoding: 1, // LENGTH_PREFIXED,
        jsonFormat: 1, // ALLOW,
        enforceNamingStyle: 1, // STYLE2024,
        defaultSymbolVisibility: 2, // EXPORT_TOP_LEVEL,
    },
};
/**
 * Create a descriptor for a file, add it to the registry.
 */
function addFile(proto, reg) {
    var _a, _b;
    const file = {
        kind: "file",
        proto,
        deprecated: (_b = (_a = proto.options) === null || _a === void 0 ? void 0 : _a.deprecated) !== null && _b !== void 0 ? _b : false,
        edition: getFileEdition(proto),
        name: proto.name.replace(/\.proto$/, ""),
        dependencies: findFileDependencies(proto, reg),
        enums: [],
        messages: [],
        extensions: [],
        services: [],
        toString() {
            // eslint-disable-next-line @typescript-eslint/restrict-template-expressions -- we asserted above
            return `file ${proto.name}`;
        },
    };
    const mapEntriesStore = new Map();
    const mapEntries = {
        get(typeName) {
            return mapEntriesStore.get(typeName);
        },
        add(desc) {
            var _a;
            assert(((_a = desc.proto.options) === null || _a === void 0 ? void 0 : _a.mapEntry) === true);
            mapEntriesStore.set(desc.typeName, desc);
        },
    };
    for (const enumProto of proto.enumType) {
        addEnum(enumProto, file, undefined, reg);
    }
    for (const messageProto of proto.messageType) {
        addMessage(messageProto, file, undefined, reg, mapEntries);
    }
    for (const serviceProto of proto.service) {
        addService(serviceProto, file, reg);
    }
    addExtensions(file, reg);
    for (const mapEntry of mapEntriesStore.values()) {
        // to create a map field, we need access to the map entry's fields
        addFields(mapEntry, reg, mapEntries);
    }
    for (const message of file.messages) {
        addFields(message, reg, mapEntries);
        addExtensions(message, reg);
    }
    reg.addFile(file, true);
}
/**
 * Create descriptors for extensions, and add them to the message / file,
 * and to our cart.
 * Recurses into nested types.
 */
function addExtensions(desc, reg) {
    switch (desc.kind) {
        case "file":
            for (const proto of desc.proto.extension) {
                const ext = newField(proto, desc, reg);
                desc.extensions.push(ext);
                reg.add(ext);
            }
            break;
        case "message":
            for (const proto of desc.proto.extension) {
                const ext = newField(proto, desc, reg);
                desc.nestedExtensions.push(ext);
                reg.add(ext);
            }
            for (const message of desc.nestedMessages) {
                addExtensions(message, reg);
            }
            break;
    }
}
/**
 * Create descriptors for fields and oneof groups, and add them to the message.
 * Recurses into nested types.
 */
function addFields(message, reg, mapEntries) {
    const allOneofs = message.proto.oneofDecl.map((proto) => newOneof(proto, message));
    const oneofsSeen = new Set();
    for (const proto of message.proto.field) {
        const oneof = findOneof(proto, allOneofs);
        const field = newField(proto, message, reg, oneof, mapEntries);
        message.fields.push(field);
        message.field[field.localName] = field;
        if (oneof === undefined) {
            message.members.push(field);
        }
        else {
            oneof.fields.push(field);
            if (!oneofsSeen.has(oneof)) {
                oneofsSeen.add(oneof);
                message.members.push(oneof);
            }
        }
    }
    for (const oneof of allOneofs.filter((o) => oneofsSeen.has(o))) {
        message.oneofs.push(oneof);
    }
    for (const child of message.nestedMessages) {
        addFields(child, reg, mapEntries);
    }
}
/**
 * Create a descriptor for an enumeration, and add it our cart and to the
 * parent type, if any.
 */
function addEnum(proto, file, parent, reg) {
    var _a, _b, _c, _d, _e;
    const sharedPrefix = findEnumSharedPrefix(proto.name, proto.value);
    const desc = {
        kind: "enum",
        proto,
        deprecated: (_b = (_a = proto.options) === null || _a === void 0 ? void 0 : _a.deprecated) !== null && _b !== void 0 ? _b : false,
        file,
        parent,
        open: true,
        name: proto.name,
        typeName: makeTypeName(proto, parent, file),
        value: {},
        values: [],
        sharedPrefix,
        toString() {
            return `enum ${this.typeName}`;
        },
    };
    desc.open = isEnumOpen(desc);
    reg.add(desc);
    for (const p of proto.value) {
        const name = p.name;
        desc.values.push(
        // biome-ignore lint/suspicious/noAssignInExpressions: no
        (desc.value[p.number] = {
            kind: "enum_value",
            proto: p,
            deprecated: (_d = (_c = p.options) === null || _c === void 0 ? void 0 : _c.deprecated) !== null && _d !== void 0 ? _d : false,
            parent: desc,
            name,
            localName: safeObjectProperty(sharedPrefix == undefined
                ? name
                : name.substring(sharedPrefix.length)),
            number: p.number,
            toString() {
                return `enum value ${desc.typeName}.${name}`;
            },
        }));
    }
    ((_e = parent === null || parent === void 0 ? void 0 : parent.nestedEnums) !== null && _e !== void 0 ? _e : file.enums).push(desc);
}
/**
 * Create a descriptor for a message, including nested types, and add it to our
 * cart. Note that this does not create descriptors fields.
 */
function addMessage(proto, file, parent, reg, mapEntries) {
    var _a, _b, _c, _d;
    const desc = {
        kind: "message",
        proto,
        deprecated: (_b = (_a = proto.options) === null || _a === void 0 ? void 0 : _a.deprecated) !== null && _b !== void 0 ? _b : false,
        file,
        parent,
        name: proto.name,
        typeName: makeTypeName(proto, parent, file),
        fields: [],
        field: {},
        oneofs: [],
        members: [],
        nestedEnums: [],
        nestedMessages: [],
        nestedExtensions: [],
        toString() {
            return `message ${this.typeName}`;
        },
    };
    if (((_c = proto.options) === null || _c === void 0 ? void 0 : _c.mapEntry) === true) {
        mapEntries.add(desc);
    }
    else {
        ((_d = parent === null || parent === void 0 ? void 0 : parent.nestedMessages) !== null && _d !== void 0 ? _d : file.messages).push(desc);
        reg.add(desc);
    }
    for (const enumProto of proto.enumType) {
        addEnum(enumProto, file, desc, reg);
    }
    for (const messageProto of proto.nestedType) {
        addMessage(messageProto, file, desc, reg, mapEntries);
    }
}
/**
 * Create a descriptor for a service, including methods, and add it to our
 * cart.
 */
function addService(proto, file, reg) {
    var _a, _b;
    const desc = {
        kind: "service",
        proto,
        deprecated: (_b = (_a = proto.options) === null || _a === void 0 ? void 0 : _a.deprecated) !== null && _b !== void 0 ? _b : false,
        file,
        name: proto.name,
        typeName: makeTypeName(proto, undefined, file),
        methods: [],
        method: {},
        toString() {
            return `service ${this.typeName}`;
        },
    };
    file.services.push(desc);
    reg.add(desc);
    for (const methodProto of proto.method) {
        const method = newMethod(methodProto, desc, reg);
        desc.methods.push(method);
        desc.method[method.localName] = method;
    }
}
/**
 * Create a descriptor for a method.
 */
function newMethod(proto, parent, reg) {
    var _a, _b, _c, _d;
    let methodKind;
    if (proto.clientStreaming && proto.serverStreaming) {
        methodKind = "bidi_streaming";
    }
    else if (proto.clientStreaming) {
        methodKind = "client_streaming";
    }
    else if (proto.serverStreaming) {
        methodKind = "server_streaming";
    }
    else {
        methodKind = "unary";
    }
    const input = reg.getMessage(trimLeadingDot(proto.inputType));
    const output = reg.getMessage(trimLeadingDot(proto.outputType));
    assert(input, `invalid MethodDescriptorProto: input_type ${proto.inputType} not found`);
    assert(output, `invalid MethodDescriptorProto: output_type ${proto.inputType} not found`);
    const name = proto.name;
    return {
        kind: "rpc",
        proto,
        deprecated: (_b = (_a = proto.options) === null || _a === void 0 ? void 0 : _a.deprecated) !== null && _b !== void 0 ? _b : false,
        parent,
        name,
        localName: safeObjectProperty(name.length
            ? safeObjectProperty(name[0].toLowerCase() + name.substring(1))
            : name),
        methodKind,
        input,
        output,
        idempotency: (_d = (_c = proto.options) === null || _c === void 0 ? void 0 : _c.idempotencyLevel) !== null && _d !== void 0 ? _d : IDEMPOTENCY_UNKNOWN,
        toString() {
            return `rpc ${parent.typeName}.${name}`;
        },
    };
}
/**
 * Create a descriptor for a oneof group.
 */
function newOneof(proto, parent) {
    return {
        kind: "oneof",
        proto,
        deprecated: false,
        parent,
        fields: [],
        name: proto.name,
        localName: safeObjectProperty(protoCamelCase(proto.name)),
        toString() {
            return `oneof ${parent.typeName}.${this.name}`;
        },
    };
}
function newField(proto, parentOrFile, reg, oneof, mapEntries) {
    var _a, _b, _c;
    const isExtension = mapEntries === undefined;
    const field = {
        kind: "field",
        proto,
        deprecated: (_b = (_a = proto.options) === null || _a === void 0 ? void 0 : _a.deprecated) !== null && _b !== void 0 ? _b : false,
        name: proto.name,
        number: proto.number,
        scalar: undefined,
        message: undefined,
        enum: undefined,
        presence: getFieldPresence(proto, oneof, isExtension, parentOrFile),
        listKind: undefined,
        mapKind: undefined,
        mapKey: undefined,
        delimitedEncoding: undefined,
        packed: undefined,
        longAsString: false,
        getDefaultValue: undefined,
    };
    if (isExtension) {
        // extension field
        const file = parentOrFile.kind == "file" ? parentOrFile : parentOrFile.file;
        const parent = parentOrFile.kind == "file" ? undefined : parentOrFile;
        const typeName = makeTypeName(proto, parent, file);
        field.kind = "extension";
        field.file = file;
        field.parent = parent;
        field.oneof = undefined;
        field.typeName = typeName;
        field.jsonName = `[${typeName}]`; // option json_name is not allowed on extension fields
        field.toString = () => `extension ${typeName}`;
        const extendee = reg.getMessage(trimLeadingDot(proto.extendee));
        assert(extendee, `invalid FieldDescriptorProto: extendee ${proto.extendee} not found`);
        field.extendee = extendee;
    }
    else {
        // regular field
        const parent = parentOrFile;
        assert(parent.kind == "message");
        field.parent = parent;
        field.oneof = oneof;
        field.localName = oneof
            ? protoCamelCase(proto.name)
            : safeObjectProperty(protoCamelCase(proto.name));
        field.jsonName = proto.jsonName;
        field.toString = () => `field ${parent.typeName}.${proto.name}`;
    }
    const label = proto.label;
    const type = proto.type;
    const jstype = (_c = proto.options) === null || _c === void 0 ? void 0 : _c.jstype;
    if (label === LABEL_REPEATED) {
        // list or map field
        const mapEntry = type == TYPE_MESSAGE
            ? mapEntries === null || mapEntries === void 0 ? void 0 : mapEntries.get(trimLeadingDot(proto.typeName))
            : undefined;
        if (mapEntry) {
            // map field
            field.fieldKind = "map";
            const { key, value } = findMapEntryFields(mapEntry);
            field.mapKey = key.scalar;
            field.mapKind = value.fieldKind;
            field.message = value.message;
            field.delimitedEncoding = false; // map fields are always LENGTH_PREFIXED
            field.enum = value.enum;
            field.scalar = value.scalar;
            return field;
        }
        // list field
        field.fieldKind = "list";
        switch (type) {
            case TYPE_MESSAGE:
            case TYPE_GROUP:
                field.listKind = "message";
                field.message = reg.getMessage(trimLeadingDot(proto.typeName));
                assert(field.message);
                field.delimitedEncoding = isDelimitedEncoding(proto, parentOrFile);
                break;
            case TYPE_ENUM:
                field.listKind = "enum";
                field.enum = reg.getEnum(trimLeadingDot(proto.typeName));
                assert(field.enum);
                break;
            default:
                field.listKind = "scalar";
                field.scalar = type;
                field.longAsString = jstype == JS_STRING;
                break;
        }
        field.packed = isPackedField(proto, parentOrFile);
        return field;
    }
    // singular
    switch (type) {
        case TYPE_MESSAGE:
        case TYPE_GROUP:
            field.fieldKind = "message";
            field.message = reg.getMessage(trimLeadingDot(proto.typeName));
            assert(field.message, `invalid FieldDescriptorProto: type_name ${proto.typeName} not found`);
            field.delimitedEncoding = isDelimitedEncoding(proto, parentOrFile);
            field.getDefaultValue = () => undefined;
            break;
        case TYPE_ENUM: {
            const enumeration = reg.getEnum(trimLeadingDot(proto.typeName));
            assert(enumeration !== undefined, `invalid FieldDescriptorProto: type_name ${proto.typeName} not found`);
            field.fieldKind = "enum";
            field.enum = reg.getEnum(trimLeadingDot(proto.typeName));
            field.getDefaultValue = () => {
                return unsafeIsSetExplicit(proto, "defaultValue")
                    ? parseTextFormatEnumValue(enumeration, proto.defaultValue)
                    : undefined;
            };
            break;
        }
        default: {
            field.fieldKind = "scalar";
            field.scalar = type;
            field.longAsString = jstype == JS_STRING;
            field.getDefaultValue = () => {
                return unsafeIsSetExplicit(proto, "defaultValue")
                    ? parseTextFormatScalarValue(type, proto.defaultValue)
                    : undefined;
            };
            break;
        }
    }
    return field;
}
/**
 * Parse the "syntax" and "edition" fields, returning one of the supported
 * editions.
 */
function getFileEdition(proto) {
    switch (proto.syntax) {
        case "":
        case "proto2":
            return EDITION_PROTO2;
        case "proto3":
            return EDITION_PROTO3;
        case "editions":
            if (proto.edition in featureDefaults) {
                return proto.edition;
            }
            throw new Error(`${proto.name}: unsupported edition`);
        default:
            throw new Error(`${proto.name}: unsupported syntax "${proto.syntax}"`);
    }
}
/**
 * Resolve dependencies of FileDescriptorProto to DescFile.
 */
function findFileDependencies(proto, reg) {
    return proto.dependency.map((wantName) => {
        const dep = reg.getFile(wantName);
        if (!dep) {
            throw new Error(`Cannot find ${wantName}, imported by ${proto.name}`);
        }
        return dep;
    });
}
/**
 * Finds a prefix shared by enum values, for example `my_enum_` for
 * `enum MyEnum {MY_ENUM_A=0; MY_ENUM_B=1;}`.
 */
function findEnumSharedPrefix(enumName, values) {
    const prefix = camelToSnakeCase(enumName) + "_";
    for (const value of values) {
        if (!value.name.toLowerCase().startsWith(prefix)) {
            return undefined;
        }
        const shortName = value.name.substring(prefix.length);
        if (shortName.length == 0) {
            return undefined;
        }
        if (/^\d/.test(shortName)) {
            // identifiers must not start with numbers
            return undefined;
        }
    }
    return prefix;
}
/**
 * Converts lowerCamelCase or UpperCamelCase into lower_snake_case.
 * This is used to find shared prefixes in an enum.
 */
function camelToSnakeCase(camel) {
    return (camel.substring(0, 1) + camel.substring(1).replace(/[A-Z]/g, (c) => "_" + c)).toLowerCase();
}
/**
 * Create a fully qualified name for a protobuf type or extension field.
 *
 * The fully qualified name for messages, enumerations, and services is
 * constructed by concatenating the package name (if present), parent
 * message names (for nested types), and the type name. We omit the leading
 * dot added by protobuf compilers. Examples:
 * - mypackage.MyMessage
 * - mypackage.MyMessage.NestedMessage
 *
 * The fully qualified name for extension fields is constructed by
 * concatenating the package name (if present), parent message names (for
 * extensions declared within a message), and the field name. Examples:
 * - mypackage.extfield
 * - mypackage.MyMessage.extfield
 */
function makeTypeName(proto, parent, file) {
    let typeName;
    if (parent) {
        typeName = `${parent.typeName}.${proto.name}`;
    }
    else if (file.proto.package.length > 0) {
        typeName = `${file.proto.package}.${proto.name}`;
    }
    else {
        typeName = `${proto.name}`;
    }
    return typeName;
}
/**
 * Remove the leading dot from a fully qualified type name.
 */
function trimLeadingDot(typeName) {
    return typeName.startsWith(".") ? typeName.substring(1) : typeName;
}
/**
 * Did the user put the field in a oneof group?
 * Synthetic oneofs for proto3 optionals are ignored.
 */
function findOneof(proto, allOneofs) {
    if (!unsafeIsSetExplicit(proto, "oneofIndex")) {
        return undefined;
    }
    if (proto.proto3Optional) {
        return undefined;
    }
    const oneof = allOneofs[proto.oneofIndex];
    assert(oneof, `invalid FieldDescriptorProto: oneof #${proto.oneofIndex} for field #${proto.number} not found`);
    return oneof;
}
/**
 * Presence of the field.
 * See https://protobuf.dev/programming-guides/field_presence/
 */
function getFieldPresence(proto, oneof, isExtension, parent) {
    if (proto.label == LABEL_REQUIRED) {
        // proto2 required is LEGACY_REQUIRED
        return LEGACY_REQUIRED$2;
    }
    if (proto.label == LABEL_REPEATED) {
        // repeated fields (including maps) do not track presence
        return IMPLICIT$1;
    }
    if (!!oneof || proto.proto3Optional) {
        // oneof is always explicit
        return EXPLICIT;
    }
    if (isExtension) {
        // extensions always track presence
        return EXPLICIT;
    }
    const resolved = resolveFeature("fieldPresence", { proto, parent });
    if (resolved == IMPLICIT$1 &&
        (proto.type == TYPE_MESSAGE || proto.type == TYPE_GROUP)) {
        // singular message field cannot be implicit
        return EXPLICIT;
    }
    return resolved;
}
/**
 * Pack this repeated field?
 */
function isPackedField(proto, parent) {
    if (proto.label != LABEL_REPEATED) {
        return false;
    }
    switch (proto.type) {
        case TYPE_STRING:
        case TYPE_BYTES:
        case TYPE_GROUP:
        case TYPE_MESSAGE:
            // length-delimited types cannot be packed
            return false;
    }
    const o = proto.options;
    if (o && unsafeIsSetExplicit(o, "packed")) {
        // prefer the field option over edition features
        return o.packed;
    }
    return (PACKED ==
        resolveFeature("repeatedFieldEncoding", {
            proto,
            parent,
        }));
}
/**
 * Find the key and value fields of a synthetic map entry message.
 */
function findMapEntryFields(mapEntry) {
    const key = mapEntry.fields.find((f) => f.number === 1);
    const value = mapEntry.fields.find((f) => f.number === 2);
    assert(key &&
        key.fieldKind == "scalar" &&
        key.scalar != ScalarType.BYTES &&
        key.scalar != ScalarType.FLOAT &&
        key.scalar != ScalarType.DOUBLE &&
        value &&
        value.fieldKind != "list" &&
        value.fieldKind != "map");
    return { key, value };
}
/**
 * Enumerations can be open or closed.
 * See https://protobuf.dev/programming-guides/enum/
 */
function isEnumOpen(desc) {
    var _a;
    return (OPEN ==
        resolveFeature("enumType", {
            proto: desc.proto,
            parent: (_a = desc.parent) !== null && _a !== void 0 ? _a : desc.file,
        }));
}
/**
 * Encode the message delimited (a.k.a. proto2 group encoding), or
 * length-prefixed?
 */
function isDelimitedEncoding(proto, parent) {
    if (proto.type == TYPE_GROUP) {
        return true;
    }
    return (DELIMITED ==
        resolveFeature("messageEncoding", {
            proto,
            parent,
        }));
}
function resolveFeature(name, ref) {
    var _a, _b;
    const featureSet = (_a = ref.proto.options) === null || _a === void 0 ? void 0 : _a.features;
    if (featureSet) {
        const val = featureSet[name];
        if (val != 0) {
            return val;
        }
    }
    if ("kind" in ref) {
        if (ref.kind == "message") {
            return resolveFeature(name, (_b = ref.parent) !== null && _b !== void 0 ? _b : ref.file);
        }
        const editionDefaults = featureDefaults[ref.edition];
        if (!editionDefaults) {
            throw new Error(`feature default for edition ${ref.edition} not found`);
        }
        return editionDefaults[name];
    }
    return resolveFeature(name, ref.parent);
}
/**
 * Assert that condition is truthy or throw error (with message)
 */
function assert(condition, msg) {
    if (!condition) {
        throw new Error(msg);
    }
}

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Hydrate a file descriptor for google/protobuf/descriptor.proto from a plain
 * object.
 *
 * See createFileDescriptorProtoBoot() for details.
 *
 * @private
 */
function boot(boot) {
    const root = bootFileDescriptorProto(boot);
    root.messageType.forEach(restoreJsonNames);
    const reg = createFileRegistry(root, () => undefined);
    // biome-ignore lint/style/noNonNullAssertion: non-null assertion because we just created the registry from the file we look up
    return reg.getFile(root.name);
}
/**
 * Creates the message google.protobuf.FileDescriptorProto from an object literal.
 *
 * See createFileDescriptorProtoBoot() for details.
 *
 * @private
 */
function bootFileDescriptorProto(init) {
    const proto = Object.create({
        syntax: "",
        edition: 0,
    });
    return Object.assign(proto, Object.assign(Object.assign({ $typeName: "google.protobuf.FileDescriptorProto", dependency: [], publicDependency: [], weakDependency: [], optionDependency: [], service: [], extension: [] }, init), { messageType: init.messageType.map(bootDescriptorProto), enumType: init.enumType.map(bootEnumDescriptorProto) }));
}
function bootDescriptorProto(init) {
    var _a, _b, _c, _d, _e, _f, _g, _h;
    const proto = Object.create({
        visibility: 0,
    });
    return Object.assign(proto, {
        $typeName: "google.protobuf.DescriptorProto",
        name: init.name,
        field: (_b = (_a = init.field) === null || _a === void 0 ? void 0 : _a.map(bootFieldDescriptorProto)) !== null && _b !== void 0 ? _b : [],
        extension: [],
        nestedType: (_d = (_c = init.nestedType) === null || _c === void 0 ? void 0 : _c.map(bootDescriptorProto)) !== null && _d !== void 0 ? _d : [],
        enumType: (_f = (_e = init.enumType) === null || _e === void 0 ? void 0 : _e.map(bootEnumDescriptorProto)) !== null && _f !== void 0 ? _f : [],
        extensionRange: (_h = (_g = init.extensionRange) === null || _g === void 0 ? void 0 : _g.map((e) => (Object.assign({ $typeName: "google.protobuf.DescriptorProto.ExtensionRange" }, e)))) !== null && _h !== void 0 ? _h : [],
        oneofDecl: [],
        reservedRange: [],
        reservedName: [],
    });
}
function bootFieldDescriptorProto(init) {
    const proto = Object.create({
        label: 1,
        typeName: "",
        extendee: "",
        defaultValue: "",
        oneofIndex: 0,
        jsonName: "",
        proto3Optional: false,
    });
    return Object.assign(proto, Object.assign(Object.assign({ $typeName: "google.protobuf.FieldDescriptorProto" }, init), { options: init.options ? bootFieldOptions(init.options) : undefined }));
}
function bootFieldOptions(init) {
    var _a, _b, _c;
    const proto = Object.create({
        ctype: 0,
        packed: false,
        jstype: 0,
        lazy: false,
        unverifiedLazy: false,
        deprecated: false,
        weak: false,
        debugRedact: false,
        retention: 0,
    });
    return Object.assign(proto, Object.assign(Object.assign({ $typeName: "google.protobuf.FieldOptions" }, init), { targets: (_a = init.targets) !== null && _a !== void 0 ? _a : [], editionDefaults: (_c = (_b = init.editionDefaults) === null || _b === void 0 ? void 0 : _b.map((e) => (Object.assign({ $typeName: "google.protobuf.FieldOptions.EditionDefault" }, e)))) !== null && _c !== void 0 ? _c : [], uninterpretedOption: [] }));
}
function bootEnumDescriptorProto(init) {
    const proto = Object.create({
        visibility: 0,
    });
    return Object.assign(proto, {
        $typeName: "google.protobuf.EnumDescriptorProto",
        name: init.name,
        reservedName: [],
        reservedRange: [],
        value: init.value.map((e) => (Object.assign({ $typeName: "google.protobuf.EnumValueDescriptorProto" }, e))),
    });
}

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Hydrate a message descriptor.
 *
 * @private
 */
function messageDesc(file, path, ...paths) {
    return paths.reduce((acc, cur) => acc.nestedMessages[cur], file.messages[path]);
}

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Hydrate an enum descriptor.
 *
 * @private
 */
function enumDesc(file, path, ...paths) {
    if (paths.length == 0) {
        return file.enums[path];
    }
    const e = paths.pop(); // we checked length above
    return paths.reduce((acc, cur) => acc.nestedMessages[cur], file.messages[path]).nestedEnums[e];
}
/**
 * Construct a TypeScript enum object at runtime from a descriptor.
 */
function tsEnum(desc) {
    const enumObject = {};
    for (const value of desc.values) {
        enumObject[value.localName] = value.number;
        enumObject[value.number] = value.localName;
    }
    return enumObject;
}

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Describes the file google/protobuf/descriptor.proto.
 */
const file_google_protobuf_descriptor = /*@__PURE__*/ boot({ "name": "google/protobuf/descriptor.proto", "package": "google.protobuf", "messageType": [{ "name": "FileDescriptorSet", "field": [{ "name": "file", "number": 1, "type": 11, "label": 3, "typeName": ".google.protobuf.FileDescriptorProto" }], "extensionRange": [{ "start": 536000000, "end": 536000001 }] }, { "name": "FileDescriptorProto", "field": [{ "name": "name", "number": 1, "type": 9, "label": 1 }, { "name": "package", "number": 2, "type": 9, "label": 1 }, { "name": "dependency", "number": 3, "type": 9, "label": 3 }, { "name": "public_dependency", "number": 10, "type": 5, "label": 3 }, { "name": "weak_dependency", "number": 11, "type": 5, "label": 3 }, { "name": "option_dependency", "number": 15, "type": 9, "label": 3 }, { "name": "message_type", "number": 4, "type": 11, "label": 3, "typeName": ".google.protobuf.DescriptorProto" }, { "name": "enum_type", "number": 5, "type": 11, "label": 3, "typeName": ".google.protobuf.EnumDescriptorProto" }, { "name": "service", "number": 6, "type": 11, "label": 3, "typeName": ".google.protobuf.ServiceDescriptorProto" }, { "name": "extension", "number": 7, "type": 11, "label": 3, "typeName": ".google.protobuf.FieldDescriptorProto" }, { "name": "options", "number": 8, "type": 11, "label": 1, "typeName": ".google.protobuf.FileOptions" }, { "name": "source_code_info", "number": 9, "type": 11, "label": 1, "typeName": ".google.protobuf.SourceCodeInfo" }, { "name": "syntax", "number": 12, "type": 9, "label": 1 }, { "name": "edition", "number": 14, "type": 14, "label": 1, "typeName": ".google.protobuf.Edition" }] }, { "name": "DescriptorProto", "field": [{ "name": "name", "number": 1, "type": 9, "label": 1 }, { "name": "field", "number": 2, "type": 11, "label": 3, "typeName": ".google.protobuf.FieldDescriptorProto" }, { "name": "extension", "number": 6, "type": 11, "label": 3, "typeName": ".google.protobuf.FieldDescriptorProto" }, { "name": "nested_type", "number": 3, "type": 11, "label": 3, "typeName": ".google.protobuf.DescriptorProto" }, { "name": "enum_type", "number": 4, "type": 11, "label": 3, "typeName": ".google.protobuf.EnumDescriptorProto" }, { "name": "extension_range", "number": 5, "type": 11, "label": 3, "typeName": ".google.protobuf.DescriptorProto.ExtensionRange" }, { "name": "oneof_decl", "number": 8, "type": 11, "label": 3, "typeName": ".google.protobuf.OneofDescriptorProto" }, { "name": "options", "number": 7, "type": 11, "label": 1, "typeName": ".google.protobuf.MessageOptions" }, { "name": "reserved_range", "number": 9, "type": 11, "label": 3, "typeName": ".google.protobuf.DescriptorProto.ReservedRange" }, { "name": "reserved_name", "number": 10, "type": 9, "label": 3 }, { "name": "visibility", "number": 11, "type": 14, "label": 1, "typeName": ".google.protobuf.SymbolVisibility" }], "nestedType": [{ "name": "ExtensionRange", "field": [{ "name": "start", "number": 1, "type": 5, "label": 1 }, { "name": "end", "number": 2, "type": 5, "label": 1 }, { "name": "options", "number": 3, "type": 11, "label": 1, "typeName": ".google.protobuf.ExtensionRangeOptions" }] }, { "name": "ReservedRange", "field": [{ "name": "start", "number": 1, "type": 5, "label": 1 }, { "name": "end", "number": 2, "type": 5, "label": 1 }] }] }, { "name": "ExtensionRangeOptions", "field": [{ "name": "uninterpreted_option", "number": 999, "type": 11, "label": 3, "typeName": ".google.protobuf.UninterpretedOption" }, { "name": "declaration", "number": 2, "type": 11, "label": 3, "typeName": ".google.protobuf.ExtensionRangeOptions.Declaration", "options": { "retention": 2 } }, { "name": "features", "number": 50, "type": 11, "label": 1, "typeName": ".google.protobuf.FeatureSet" }, { "name": "verification", "number": 3, "type": 14, "label": 1, "typeName": ".google.protobuf.ExtensionRangeOptions.VerificationState", "defaultValue": "UNVERIFIED", "options": { "retention": 2 } }], "nestedType": [{ "name": "Declaration", "field": [{ "name": "number", "number": 1, "type": 5, "label": 1 }, { "name": "full_name", "number": 2, "type": 9, "label": 1 }, { "name": "type", "number": 3, "type": 9, "label": 1 }, { "name": "reserved", "number": 5, "type": 8, "label": 1 }, { "name": "repeated", "number": 6, "type": 8, "label": 1 }] }], "enumType": [{ "name": "VerificationState", "value": [{ "name": "DECLARATION", "number": 0 }, { "name": "UNVERIFIED", "number": 1 }] }], "extensionRange": [{ "start": 1000, "end": 536870912 }] }, { "name": "FieldDescriptorProto", "field": [{ "name": "name", "number": 1, "type": 9, "label": 1 }, { "name": "number", "number": 3, "type": 5, "label": 1 }, { "name": "label", "number": 4, "type": 14, "label": 1, "typeName": ".google.protobuf.FieldDescriptorProto.Label" }, { "name": "type", "number": 5, "type": 14, "label": 1, "typeName": ".google.protobuf.FieldDescriptorProto.Type" }, { "name": "type_name", "number": 6, "type": 9, "label": 1 }, { "name": "extendee", "number": 2, "type": 9, "label": 1 }, { "name": "default_value", "number": 7, "type": 9, "label": 1 }, { "name": "oneof_index", "number": 9, "type": 5, "label": 1 }, { "name": "json_name", "number": 10, "type": 9, "label": 1 }, { "name": "options", "number": 8, "type": 11, "label": 1, "typeName": ".google.protobuf.FieldOptions" }, { "name": "proto3_optional", "number": 17, "type": 8, "label": 1 }], "enumType": [{ "name": "Type", "value": [{ "name": "TYPE_DOUBLE", "number": 1 }, { "name": "TYPE_FLOAT", "number": 2 }, { "name": "TYPE_INT64", "number": 3 }, { "name": "TYPE_UINT64", "number": 4 }, { "name": "TYPE_INT32", "number": 5 }, { "name": "TYPE_FIXED64", "number": 6 }, { "name": "TYPE_FIXED32", "number": 7 }, { "name": "TYPE_BOOL", "number": 8 }, { "name": "TYPE_STRING", "number": 9 }, { "name": "TYPE_GROUP", "number": 10 }, { "name": "TYPE_MESSAGE", "number": 11 }, { "name": "TYPE_BYTES", "number": 12 }, { "name": "TYPE_UINT32", "number": 13 }, { "name": "TYPE_ENUM", "number": 14 }, { "name": "TYPE_SFIXED32", "number": 15 }, { "name": "TYPE_SFIXED64", "number": 16 }, { "name": "TYPE_SINT32", "number": 17 }, { "name": "TYPE_SINT64", "number": 18 }] }, { "name": "Label", "value": [{ "name": "LABEL_OPTIONAL", "number": 1 }, { "name": "LABEL_REPEATED", "number": 3 }, { "name": "LABEL_REQUIRED", "number": 2 }] }] }, { "name": "OneofDescriptorProto", "field": [{ "name": "name", "number": 1, "type": 9, "label": 1 }, { "name": "options", "number": 2, "type": 11, "label": 1, "typeName": ".google.protobuf.OneofOptions" }] }, { "name": "EnumDescriptorProto", "field": [{ "name": "name", "number": 1, "type": 9, "label": 1 }, { "name": "value", "number": 2, "type": 11, "label": 3, "typeName": ".google.protobuf.EnumValueDescriptorProto" }, { "name": "options", "number": 3, "type": 11, "label": 1, "typeName": ".google.protobuf.EnumOptions" }, { "name": "reserved_range", "number": 4, "type": 11, "label": 3, "typeName": ".google.protobuf.EnumDescriptorProto.EnumReservedRange" }, { "name": "reserved_name", "number": 5, "type": 9, "label": 3 }, { "name": "visibility", "number": 6, "type": 14, "label": 1, "typeName": ".google.protobuf.SymbolVisibility" }], "nestedType": [{ "name": "EnumReservedRange", "field": [{ "name": "start", "number": 1, "type": 5, "label": 1 }, { "name": "end", "number": 2, "type": 5, "label": 1 }] }] }, { "name": "EnumValueDescriptorProto", "field": [{ "name": "name", "number": 1, "type": 9, "label": 1 }, { "name": "number", "number": 2, "type": 5, "label": 1 }, { "name": "options", "number": 3, "type": 11, "label": 1, "typeName": ".google.protobuf.EnumValueOptions" }] }, { "name": "ServiceDescriptorProto", "field": [{ "name": "name", "number": 1, "type": 9, "label": 1 }, { "name": "method", "number": 2, "type": 11, "label": 3, "typeName": ".google.protobuf.MethodDescriptorProto" }, { "name": "options", "number": 3, "type": 11, "label": 1, "typeName": ".google.protobuf.ServiceOptions" }] }, { "name": "MethodDescriptorProto", "field": [{ "name": "name", "number": 1, "type": 9, "label": 1 }, { "name": "input_type", "number": 2, "type": 9, "label": 1 }, { "name": "output_type", "number": 3, "type": 9, "label": 1 }, { "name": "options", "number": 4, "type": 11, "label": 1, "typeName": ".google.protobuf.MethodOptions" }, { "name": "client_streaming", "number": 5, "type": 8, "label": 1, "defaultValue": "false" }, { "name": "server_streaming", "number": 6, "type": 8, "label": 1, "defaultValue": "false" }] }, { "name": "FileOptions", "field": [{ "name": "java_package", "number": 1, "type": 9, "label": 1 }, { "name": "java_outer_classname", "number": 8, "type": 9, "label": 1 }, { "name": "java_multiple_files", "number": 10, "type": 8, "label": 1, "defaultValue": "false" }, { "name": "java_generate_equals_and_hash", "number": 20, "type": 8, "label": 1, "options": { "deprecated": true } }, { "name": "java_string_check_utf8", "number": 27, "type": 8, "label": 1, "defaultValue": "false" }, { "name": "optimize_for", "number": 9, "type": 14, "label": 1, "typeName": ".google.protobuf.FileOptions.OptimizeMode", "defaultValue": "SPEED" }, { "name": "go_package", "number": 11, "type": 9, "label": 1 }, { "name": "cc_generic_services", "number": 16, "type": 8, "label": 1, "defaultValue": "false" }, { "name": "java_generic_services", "number": 17, "type": 8, "label": 1, "defaultValue": "false" }, { "name": "py_generic_services", "number": 18, "type": 8, "label": 1, "defaultValue": "false" }, { "name": "deprecated", "number": 23, "type": 8, "label": 1, "defaultValue": "false" }, { "name": "cc_enable_arenas", "number": 31, "type": 8, "label": 1, "defaultValue": "true" }, { "name": "objc_class_prefix", "number": 36, "type": 9, "label": 1 }, { "name": "csharp_namespace", "number": 37, "type": 9, "label": 1 }, { "name": "swift_prefix", "number": 39, "type": 9, "label": 1 }, { "name": "php_class_prefix", "number": 40, "type": 9, "label": 1 }, { "name": "php_namespace", "number": 41, "type": 9, "label": 1 }, { "name": "php_metadata_namespace", "number": 44, "type": 9, "label": 1 }, { "name": "ruby_package", "number": 45, "type": 9, "label": 1 }, { "name": "features", "number": 50, "type": 11, "label": 1, "typeName": ".google.protobuf.FeatureSet" }, { "name": "uninterpreted_option", "number": 999, "type": 11, "label": 3, "typeName": ".google.protobuf.UninterpretedOption" }], "enumType": [{ "name": "OptimizeMode", "value": [{ "name": "SPEED", "number": 1 }, { "name": "CODE_SIZE", "number": 2 }, { "name": "LITE_RUNTIME", "number": 3 }] }], "extensionRange": [{ "start": 1000, "end": 536870912 }] }, { "name": "MessageOptions", "field": [{ "name": "message_set_wire_format", "number": 1, "type": 8, "label": 1, "defaultValue": "false" }, { "name": "no_standard_descriptor_accessor", "number": 2, "type": 8, "label": 1, "defaultValue": "false" }, { "name": "deprecated", "number": 3, "type": 8, "label": 1, "defaultValue": "false" }, { "name": "map_entry", "number": 7, "type": 8, "label": 1 }, { "name": "deprecated_legacy_json_field_conflicts", "number": 11, "type": 8, "label": 1, "options": { "deprecated": true } }, { "name": "features", "number": 12, "type": 11, "label": 1, "typeName": ".google.protobuf.FeatureSet" }, { "name": "uninterpreted_option", "number": 999, "type": 11, "label": 3, "typeName": ".google.protobuf.UninterpretedOption" }], "extensionRange": [{ "start": 1000, "end": 536870912 }] }, { "name": "FieldOptions", "field": [{ "name": "ctype", "number": 1, "type": 14, "label": 1, "typeName": ".google.protobuf.FieldOptions.CType", "defaultValue": "STRING" }, { "name": "packed", "number": 2, "type": 8, "label": 1 }, { "name": "jstype", "number": 6, "type": 14, "label": 1, "typeName": ".google.protobuf.FieldOptions.JSType", "defaultValue": "JS_NORMAL" }, { "name": "lazy", "number": 5, "type": 8, "label": 1, "defaultValue": "false" }, { "name": "unverified_lazy", "number": 15, "type": 8, "label": 1, "defaultValue": "false" }, { "name": "deprecated", "number": 3, "type": 8, "label": 1, "defaultValue": "false" }, { "name": "weak", "number": 10, "type": 8, "label": 1, "defaultValue": "false", "options": { "deprecated": true } }, { "name": "debug_redact", "number": 16, "type": 8, "label": 1, "defaultValue": "false" }, { "name": "retention", "number": 17, "type": 14, "label": 1, "typeName": ".google.protobuf.FieldOptions.OptionRetention" }, { "name": "targets", "number": 19, "type": 14, "label": 3, "typeName": ".google.protobuf.FieldOptions.OptionTargetType" }, { "name": "edition_defaults", "number": 20, "type": 11, "label": 3, "typeName": ".google.protobuf.FieldOptions.EditionDefault" }, { "name": "features", "number": 21, "type": 11, "label": 1, "typeName": ".google.protobuf.FeatureSet" }, { "name": "feature_support", "number": 22, "type": 11, "label": 1, "typeName": ".google.protobuf.FieldOptions.FeatureSupport" }, { "name": "uninterpreted_option", "number": 999, "type": 11, "label": 3, "typeName": ".google.protobuf.UninterpretedOption" }], "nestedType": [{ "name": "EditionDefault", "field": [{ "name": "edition", "number": 3, "type": 14, "label": 1, "typeName": ".google.protobuf.Edition" }, { "name": "value", "number": 2, "type": 9, "label": 1 }] }, { "name": "FeatureSupport", "field": [{ "name": "edition_introduced", "number": 1, "type": 14, "label": 1, "typeName": ".google.protobuf.Edition" }, { "name": "edition_deprecated", "number": 2, "type": 14, "label": 1, "typeName": ".google.protobuf.Edition" }, { "name": "deprecation_warning", "number": 3, "type": 9, "label": 1 }, { "name": "edition_removed", "number": 4, "type": 14, "label": 1, "typeName": ".google.protobuf.Edition" }] }], "enumType": [{ "name": "CType", "value": [{ "name": "STRING", "number": 0 }, { "name": "CORD", "number": 1 }, { "name": "STRING_PIECE", "number": 2 }] }, { "name": "JSType", "value": [{ "name": "JS_NORMAL", "number": 0 }, { "name": "JS_STRING", "number": 1 }, { "name": "JS_NUMBER", "number": 2 }] }, { "name": "OptionRetention", "value": [{ "name": "RETENTION_UNKNOWN", "number": 0 }, { "name": "RETENTION_RUNTIME", "number": 1 }, { "name": "RETENTION_SOURCE", "number": 2 }] }, { "name": "OptionTargetType", "value": [{ "name": "TARGET_TYPE_UNKNOWN", "number": 0 }, { "name": "TARGET_TYPE_FILE", "number": 1 }, { "name": "TARGET_TYPE_EXTENSION_RANGE", "number": 2 }, { "name": "TARGET_TYPE_MESSAGE", "number": 3 }, { "name": "TARGET_TYPE_FIELD", "number": 4 }, { "name": "TARGET_TYPE_ONEOF", "number": 5 }, { "name": "TARGET_TYPE_ENUM", "number": 6 }, { "name": "TARGET_TYPE_ENUM_ENTRY", "number": 7 }, { "name": "TARGET_TYPE_SERVICE", "number": 8 }, { "name": "TARGET_TYPE_METHOD", "number": 9 }] }], "extensionRange": [{ "start": 1000, "end": 536870912 }] }, { "name": "OneofOptions", "field": [{ "name": "features", "number": 1, "type": 11, "label": 1, "typeName": ".google.protobuf.FeatureSet" }, { "name": "uninterpreted_option", "number": 999, "type": 11, "label": 3, "typeName": ".google.protobuf.UninterpretedOption" }], "extensionRange": [{ "start": 1000, "end": 536870912 }] }, { "name": "EnumOptions", "field": [{ "name": "allow_alias", "number": 2, "type": 8, "label": 1 }, { "name": "deprecated", "number": 3, "type": 8, "label": 1, "defaultValue": "false" }, { "name": "deprecated_legacy_json_field_conflicts", "number": 6, "type": 8, "label": 1, "options": { "deprecated": true } }, { "name": "features", "number": 7, "type": 11, "label": 1, "typeName": ".google.protobuf.FeatureSet" }, { "name": "uninterpreted_option", "number": 999, "type": 11, "label": 3, "typeName": ".google.protobuf.UninterpretedOption" }], "extensionRange": [{ "start": 1000, "end": 536870912 }] }, { "name": "EnumValueOptions", "field": [{ "name": "deprecated", "number": 1, "type": 8, "label": 1, "defaultValue": "false" }, { "name": "features", "number": 2, "type": 11, "label": 1, "typeName": ".google.protobuf.FeatureSet" }, { "name": "debug_redact", "number": 3, "type": 8, "label": 1, "defaultValue": "false" }, { "name": "feature_support", "number": 4, "type": 11, "label": 1, "typeName": ".google.protobuf.FieldOptions.FeatureSupport" }, { "name": "uninterpreted_option", "number": 999, "type": 11, "label": 3, "typeName": ".google.protobuf.UninterpretedOption" }], "extensionRange": [{ "start": 1000, "end": 536870912 }] }, { "name": "ServiceOptions", "field": [{ "name": "features", "number": 34, "type": 11, "label": 1, "typeName": ".google.protobuf.FeatureSet" }, { "name": "deprecated", "number": 33, "type": 8, "label": 1, "defaultValue": "false" }, { "name": "uninterpreted_option", "number": 999, "type": 11, "label": 3, "typeName": ".google.protobuf.UninterpretedOption" }], "extensionRange": [{ "start": 1000, "end": 536870912 }] }, { "name": "MethodOptions", "field": [{ "name": "deprecated", "number": 33, "type": 8, "label": 1, "defaultValue": "false" }, { "name": "idempotency_level", "number": 34, "type": 14, "label": 1, "typeName": ".google.protobuf.MethodOptions.IdempotencyLevel", "defaultValue": "IDEMPOTENCY_UNKNOWN" }, { "name": "features", "number": 35, "type": 11, "label": 1, "typeName": ".google.protobuf.FeatureSet" }, { "name": "uninterpreted_option", "number": 999, "type": 11, "label": 3, "typeName": ".google.protobuf.UninterpretedOption" }], "enumType": [{ "name": "IdempotencyLevel", "value": [{ "name": "IDEMPOTENCY_UNKNOWN", "number": 0 }, { "name": "NO_SIDE_EFFECTS", "number": 1 }, { "name": "IDEMPOTENT", "number": 2 }] }], "extensionRange": [{ "start": 1000, "end": 536870912 }] }, { "name": "UninterpretedOption", "field": [{ "name": "name", "number": 2, "type": 11, "label": 3, "typeName": ".google.protobuf.UninterpretedOption.NamePart" }, { "name": "identifier_value", "number": 3, "type": 9, "label": 1 }, { "name": "positive_int_value", "number": 4, "type": 4, "label": 1 }, { "name": "negative_int_value", "number": 5, "type": 3, "label": 1 }, { "name": "double_value", "number": 6, "type": 1, "label": 1 }, { "name": "string_value", "number": 7, "type": 12, "label": 1 }, { "name": "aggregate_value", "number": 8, "type": 9, "label": 1 }], "nestedType": [{ "name": "NamePart", "field": [{ "name": "name_part", "number": 1, "type": 9, "label": 2 }, { "name": "is_extension", "number": 2, "type": 8, "label": 2 }] }] }, { "name": "FeatureSet", "field": [{ "name": "field_presence", "number": 1, "type": 14, "label": 1, "typeName": ".google.protobuf.FeatureSet.FieldPresence", "options": { "retention": 1, "targets": [4, 1], "editionDefaults": [{ "value": "EXPLICIT", "edition": 900 }, { "value": "IMPLICIT", "edition": 999 }, { "value": "EXPLICIT", "edition": 1000 }] } }, { "name": "enum_type", "number": 2, "type": 14, "label": 1, "typeName": ".google.protobuf.FeatureSet.EnumType", "options": { "retention": 1, "targets": [6, 1], "editionDefaults": [{ "value": "CLOSED", "edition": 900 }, { "value": "OPEN", "edition": 999 }] } }, { "name": "repeated_field_encoding", "number": 3, "type": 14, "label": 1, "typeName": ".google.protobuf.FeatureSet.RepeatedFieldEncoding", "options": { "retention": 1, "targets": [4, 1], "editionDefaults": [{ "value": "EXPANDED", "edition": 900 }, { "value": "PACKED", "edition": 999 }] } }, { "name": "utf8_validation", "number": 4, "type": 14, "label": 1, "typeName": ".google.protobuf.FeatureSet.Utf8Validation", "options": { "retention": 1, "targets": [4, 1], "editionDefaults": [{ "value": "NONE", "edition": 900 }, { "value": "VERIFY", "edition": 999 }] } }, { "name": "message_encoding", "number": 5, "type": 14, "label": 1, "typeName": ".google.protobuf.FeatureSet.MessageEncoding", "options": { "retention": 1, "targets": [4, 1], "editionDefaults": [{ "value": "LENGTH_PREFIXED", "edition": 900 }] } }, { "name": "json_format", "number": 6, "type": 14, "label": 1, "typeName": ".google.protobuf.FeatureSet.JsonFormat", "options": { "retention": 1, "targets": [3, 6, 1], "editionDefaults": [{ "value": "LEGACY_BEST_EFFORT", "edition": 900 }, { "value": "ALLOW", "edition": 999 }] } }, { "name": "enforce_naming_style", "number": 7, "type": 14, "label": 1, "typeName": ".google.protobuf.FeatureSet.EnforceNamingStyle", "options": { "retention": 2, "targets": [1, 2, 3, 4, 5, 6, 7, 8, 9], "editionDefaults": [{ "value": "STYLE_LEGACY", "edition": 900 }, { "value": "STYLE2024", "edition": 1001 }] } }, { "name": "default_symbol_visibility", "number": 8, "type": 14, "label": 1, "typeName": ".google.protobuf.FeatureSet.VisibilityFeature.DefaultSymbolVisibility", "options": { "retention": 2, "targets": [1], "editionDefaults": [{ "value": "EXPORT_ALL", "edition": 900 }, { "value": "EXPORT_TOP_LEVEL", "edition": 1001 }] } }], "nestedType": [{ "name": "VisibilityFeature", "enumType": [{ "name": "DefaultSymbolVisibility", "value": [{ "name": "DEFAULT_SYMBOL_VISIBILITY_UNKNOWN", "number": 0 }, { "name": "EXPORT_ALL", "number": 1 }, { "name": "EXPORT_TOP_LEVEL", "number": 2 }, { "name": "LOCAL_ALL", "number": 3 }, { "name": "STRICT", "number": 4 }] }] }], "enumType": [{ "name": "FieldPresence", "value": [{ "name": "FIELD_PRESENCE_UNKNOWN", "number": 0 }, { "name": "EXPLICIT", "number": 1 }, { "name": "IMPLICIT", "number": 2 }, { "name": "LEGACY_REQUIRED", "number": 3 }] }, { "name": "EnumType", "value": [{ "name": "ENUM_TYPE_UNKNOWN", "number": 0 }, { "name": "OPEN", "number": 1 }, { "name": "CLOSED", "number": 2 }] }, { "name": "RepeatedFieldEncoding", "value": [{ "name": "REPEATED_FIELD_ENCODING_UNKNOWN", "number": 0 }, { "name": "PACKED", "number": 1 }, { "name": "EXPANDED", "number": 2 }] }, { "name": "Utf8Validation", "value": [{ "name": "UTF8_VALIDATION_UNKNOWN", "number": 0 }, { "name": "VERIFY", "number": 2 }, { "name": "NONE", "number": 3 }] }, { "name": "MessageEncoding", "value": [{ "name": "MESSAGE_ENCODING_UNKNOWN", "number": 0 }, { "name": "LENGTH_PREFIXED", "number": 1 }, { "name": "DELIMITED", "number": 2 }] }, { "name": "JsonFormat", "value": [{ "name": "JSON_FORMAT_UNKNOWN", "number": 0 }, { "name": "ALLOW", "number": 1 }, { "name": "LEGACY_BEST_EFFORT", "number": 2 }] }, { "name": "EnforceNamingStyle", "value": [{ "name": "ENFORCE_NAMING_STYLE_UNKNOWN", "number": 0 }, { "name": "STYLE2024", "number": 1 }, { "name": "STYLE_LEGACY", "number": 2 }] }], "extensionRange": [{ "start": 1000, "end": 9995 }, { "start": 9995, "end": 10000 }, { "start": 10000, "end": 10001 }] }, { "name": "FeatureSetDefaults", "field": [{ "name": "defaults", "number": 1, "type": 11, "label": 3, "typeName": ".google.protobuf.FeatureSetDefaults.FeatureSetEditionDefault" }, { "name": "minimum_edition", "number": 4, "type": 14, "label": 1, "typeName": ".google.protobuf.Edition" }, { "name": "maximum_edition", "number": 5, "type": 14, "label": 1, "typeName": ".google.protobuf.Edition" }], "nestedType": [{ "name": "FeatureSetEditionDefault", "field": [{ "name": "edition", "number": 3, "type": 14, "label": 1, "typeName": ".google.protobuf.Edition" }, { "name": "overridable_features", "number": 4, "type": 11, "label": 1, "typeName": ".google.protobuf.FeatureSet" }, { "name": "fixed_features", "number": 5, "type": 11, "label": 1, "typeName": ".google.protobuf.FeatureSet" }] }] }, { "name": "SourceCodeInfo", "field": [{ "name": "location", "number": 1, "type": 11, "label": 3, "typeName": ".google.protobuf.SourceCodeInfo.Location" }], "nestedType": [{ "name": "Location", "field": [{ "name": "path", "number": 1, "type": 5, "label": 3, "options": { "packed": true } }, { "name": "span", "number": 2, "type": 5, "label": 3, "options": { "packed": true } }, { "name": "leading_comments", "number": 3, "type": 9, "label": 1 }, { "name": "trailing_comments", "number": 4, "type": 9, "label": 1 }, { "name": "leading_detached_comments", "number": 6, "type": 9, "label": 3 }] }], "extensionRange": [{ "start": 536000000, "end": 536000001 }] }, { "name": "GeneratedCodeInfo", "field": [{ "name": "annotation", "number": 1, "type": 11, "label": 3, "typeName": ".google.protobuf.GeneratedCodeInfo.Annotation" }], "nestedType": [{ "name": "Annotation", "field": [{ "name": "path", "number": 1, "type": 5, "label": 3, "options": { "packed": true } }, { "name": "source_file", "number": 2, "type": 9, "label": 1 }, { "name": "begin", "number": 3, "type": 5, "label": 1 }, { "name": "end", "number": 4, "type": 5, "label": 1 }, { "name": "semantic", "number": 5, "type": 14, "label": 1, "typeName": ".google.protobuf.GeneratedCodeInfo.Annotation.Semantic" }], "enumType": [{ "name": "Semantic", "value": [{ "name": "NONE", "number": 0 }, { "name": "SET", "number": 1 }, { "name": "ALIAS", "number": 2 }] }] }] }], "enumType": [{ "name": "Edition", "value": [{ "name": "EDITION_UNKNOWN", "number": 0 }, { "name": "EDITION_LEGACY", "number": 900 }, { "name": "EDITION_PROTO2", "number": 998 }, { "name": "EDITION_PROTO3", "number": 999 }, { "name": "EDITION_2023", "number": 1000 }, { "name": "EDITION_2024", "number": 1001 }, { "name": "EDITION_1_TEST_ONLY", "number": 1 }, { "name": "EDITION_2_TEST_ONLY", "number": 2 }, { "name": "EDITION_99997_TEST_ONLY", "number": 99997 }, { "name": "EDITION_99998_TEST_ONLY", "number": 99998 }, { "name": "EDITION_99999_TEST_ONLY", "number": 99999 }, { "name": "EDITION_MAX", "number": 2147483647 }] }, { "name": "SymbolVisibility", "value": [{ "name": "VISIBILITY_UNSET", "number": 0 }, { "name": "VISIBILITY_LOCAL", "number": 1 }, { "name": "VISIBILITY_EXPORT", "number": 2 }] }] });
/**
 * Describes the message google.protobuf.FileDescriptorProto.
 * Use `create(FileDescriptorProtoSchema)` to create a new message.
 */
const FileDescriptorProtoSchema = /*@__PURE__*/ messageDesc(file_google_protobuf_descriptor, 1);
/**
 * The verification state of the extension range.
 *
 * @generated from enum google.protobuf.ExtensionRangeOptions.VerificationState
 */
var ExtensionRangeOptions_VerificationState;
(function (ExtensionRangeOptions_VerificationState) {
    /**
     * All the extensions of the range must be declared.
     *
     * @generated from enum value: DECLARATION = 0;
     */
    ExtensionRangeOptions_VerificationState[ExtensionRangeOptions_VerificationState["DECLARATION"] = 0] = "DECLARATION";
    /**
     * @generated from enum value: UNVERIFIED = 1;
     */
    ExtensionRangeOptions_VerificationState[ExtensionRangeOptions_VerificationState["UNVERIFIED"] = 1] = "UNVERIFIED";
})(ExtensionRangeOptions_VerificationState || (ExtensionRangeOptions_VerificationState = {}));
/**
 * @generated from enum google.protobuf.FieldDescriptorProto.Type
 */
var FieldDescriptorProto_Type;
(function (FieldDescriptorProto_Type) {
    /**
     * 0 is reserved for errors.
     * Order is weird for historical reasons.
     *
     * @generated from enum value: TYPE_DOUBLE = 1;
     */
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["DOUBLE"] = 1] = "DOUBLE";
    /**
     * @generated from enum value: TYPE_FLOAT = 2;
     */
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["FLOAT"] = 2] = "FLOAT";
    /**
     * Not ZigZag encoded.  Negative numbers take 10 bytes.  Use TYPE_SINT64 if
     * negative values are likely.
     *
     * @generated from enum value: TYPE_INT64 = 3;
     */
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["INT64"] = 3] = "INT64";
    /**
     * @generated from enum value: TYPE_UINT64 = 4;
     */
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["UINT64"] = 4] = "UINT64";
    /**
     * Not ZigZag encoded.  Negative numbers take 10 bytes.  Use TYPE_SINT32 if
     * negative values are likely.
     *
     * @generated from enum value: TYPE_INT32 = 5;
     */
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["INT32"] = 5] = "INT32";
    /**
     * @generated from enum value: TYPE_FIXED64 = 6;
     */
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["FIXED64"] = 6] = "FIXED64";
    /**
     * @generated from enum value: TYPE_FIXED32 = 7;
     */
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["FIXED32"] = 7] = "FIXED32";
    /**
     * @generated from enum value: TYPE_BOOL = 8;
     */
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["BOOL"] = 8] = "BOOL";
    /**
     * @generated from enum value: TYPE_STRING = 9;
     */
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["STRING"] = 9] = "STRING";
    /**
     * Tag-delimited aggregate.
     * Group type is deprecated and not supported after google.protobuf. However, Proto3
     * implementations should still be able to parse the group wire format and
     * treat group fields as unknown fields.  In Editions, the group wire format
     * can be enabled via the `message_encoding` feature.
     *
     * @generated from enum value: TYPE_GROUP = 10;
     */
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["GROUP"] = 10] = "GROUP";
    /**
     * Length-delimited aggregate.
     *
     * @generated from enum value: TYPE_MESSAGE = 11;
     */
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["MESSAGE"] = 11] = "MESSAGE";
    /**
     * New in version 2.
     *
     * @generated from enum value: TYPE_BYTES = 12;
     */
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["BYTES"] = 12] = "BYTES";
    /**
     * @generated from enum value: TYPE_UINT32 = 13;
     */
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["UINT32"] = 13] = "UINT32";
    /**
     * @generated from enum value: TYPE_ENUM = 14;
     */
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["ENUM"] = 14] = "ENUM";
    /**
     * @generated from enum value: TYPE_SFIXED32 = 15;
     */
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["SFIXED32"] = 15] = "SFIXED32";
    /**
     * @generated from enum value: TYPE_SFIXED64 = 16;
     */
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["SFIXED64"] = 16] = "SFIXED64";
    /**
     * Uses ZigZag encoding.
     *
     * @generated from enum value: TYPE_SINT32 = 17;
     */
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["SINT32"] = 17] = "SINT32";
    /**
     * Uses ZigZag encoding.
     *
     * @generated from enum value: TYPE_SINT64 = 18;
     */
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["SINT64"] = 18] = "SINT64";
})(FieldDescriptorProto_Type || (FieldDescriptorProto_Type = {}));
/**
 * @generated from enum google.protobuf.FieldDescriptorProto.Label
 */
var FieldDescriptorProto_Label;
(function (FieldDescriptorProto_Label) {
    /**
     * 0 is reserved for errors
     *
     * @generated from enum value: LABEL_OPTIONAL = 1;
     */
    FieldDescriptorProto_Label[FieldDescriptorProto_Label["OPTIONAL"] = 1] = "OPTIONAL";
    /**
     * @generated from enum value: LABEL_REPEATED = 3;
     */
    FieldDescriptorProto_Label[FieldDescriptorProto_Label["REPEATED"] = 3] = "REPEATED";
    /**
     * The required label is only allowed in google.protobuf.  In proto3 and Editions
     * it's explicitly prohibited.  In Editions, the `field_presence` feature
     * can be used to get this behavior.
     *
     * @generated from enum value: LABEL_REQUIRED = 2;
     */
    FieldDescriptorProto_Label[FieldDescriptorProto_Label["REQUIRED"] = 2] = "REQUIRED";
})(FieldDescriptorProto_Label || (FieldDescriptorProto_Label = {}));
/**
 * Generated classes can be optimized for speed or code size.
 *
 * @generated from enum google.protobuf.FileOptions.OptimizeMode
 */
var FileOptions_OptimizeMode;
(function (FileOptions_OptimizeMode) {
    /**
     * Generate complete code for parsing, serialization,
     *
     * @generated from enum value: SPEED = 1;
     */
    FileOptions_OptimizeMode[FileOptions_OptimizeMode["SPEED"] = 1] = "SPEED";
    /**
     * etc.
     *
     * Use ReflectionOps to implement these methods.
     *
     * @generated from enum value: CODE_SIZE = 2;
     */
    FileOptions_OptimizeMode[FileOptions_OptimizeMode["CODE_SIZE"] = 2] = "CODE_SIZE";
    /**
     * Generate code using MessageLite and the lite runtime.
     *
     * @generated from enum value: LITE_RUNTIME = 3;
     */
    FileOptions_OptimizeMode[FileOptions_OptimizeMode["LITE_RUNTIME"] = 3] = "LITE_RUNTIME";
})(FileOptions_OptimizeMode || (FileOptions_OptimizeMode = {}));
/**
 * @generated from enum google.protobuf.FieldOptions.CType
 */
var FieldOptions_CType;
(function (FieldOptions_CType) {
    /**
     * Default mode.
     *
     * @generated from enum value: STRING = 0;
     */
    FieldOptions_CType[FieldOptions_CType["STRING"] = 0] = "STRING";
    /**
     * The option [ctype=CORD] may be applied to a non-repeated field of type
     * "bytes". It indicates that in C++, the data should be stored in a Cord
     * instead of a string.  For very large strings, this may reduce memory
     * fragmentation. It may also allow better performance when parsing from a
     * Cord, or when parsing with aliasing enabled, as the parsed Cord may then
     * alias the original buffer.
     *
     * @generated from enum value: CORD = 1;
     */
    FieldOptions_CType[FieldOptions_CType["CORD"] = 1] = "CORD";
    /**
     * @generated from enum value: STRING_PIECE = 2;
     */
    FieldOptions_CType[FieldOptions_CType["STRING_PIECE"] = 2] = "STRING_PIECE";
})(FieldOptions_CType || (FieldOptions_CType = {}));
/**
 * @generated from enum google.protobuf.FieldOptions.JSType
 */
var FieldOptions_JSType;
(function (FieldOptions_JSType) {
    /**
     * Use the default type.
     *
     * @generated from enum value: JS_NORMAL = 0;
     */
    FieldOptions_JSType[FieldOptions_JSType["JS_NORMAL"] = 0] = "JS_NORMAL";
    /**
     * Use JavaScript strings.
     *
     * @generated from enum value: JS_STRING = 1;
     */
    FieldOptions_JSType[FieldOptions_JSType["JS_STRING"] = 1] = "JS_STRING";
    /**
     * Use JavaScript numbers.
     *
     * @generated from enum value: JS_NUMBER = 2;
     */
    FieldOptions_JSType[FieldOptions_JSType["JS_NUMBER"] = 2] = "JS_NUMBER";
})(FieldOptions_JSType || (FieldOptions_JSType = {}));
/**
 * If set to RETENTION_SOURCE, the option will be omitted from the binary.
 *
 * @generated from enum google.protobuf.FieldOptions.OptionRetention
 */
var FieldOptions_OptionRetention;
(function (FieldOptions_OptionRetention) {
    /**
     * @generated from enum value: RETENTION_UNKNOWN = 0;
     */
    FieldOptions_OptionRetention[FieldOptions_OptionRetention["RETENTION_UNKNOWN"] = 0] = "RETENTION_UNKNOWN";
    /**
     * @generated from enum value: RETENTION_RUNTIME = 1;
     */
    FieldOptions_OptionRetention[FieldOptions_OptionRetention["RETENTION_RUNTIME"] = 1] = "RETENTION_RUNTIME";
    /**
     * @generated from enum value: RETENTION_SOURCE = 2;
     */
    FieldOptions_OptionRetention[FieldOptions_OptionRetention["RETENTION_SOURCE"] = 2] = "RETENTION_SOURCE";
})(FieldOptions_OptionRetention || (FieldOptions_OptionRetention = {}));
/**
 * This indicates the types of entities that the field may apply to when used
 * as an option. If it is unset, then the field may be freely used as an
 * option on any kind of entity.
 *
 * @generated from enum google.protobuf.FieldOptions.OptionTargetType
 */
var FieldOptions_OptionTargetType;
(function (FieldOptions_OptionTargetType) {
    /**
     * @generated from enum value: TARGET_TYPE_UNKNOWN = 0;
     */
    FieldOptions_OptionTargetType[FieldOptions_OptionTargetType["TARGET_TYPE_UNKNOWN"] = 0] = "TARGET_TYPE_UNKNOWN";
    /**
     * @generated from enum value: TARGET_TYPE_FILE = 1;
     */
    FieldOptions_OptionTargetType[FieldOptions_OptionTargetType["TARGET_TYPE_FILE"] = 1] = "TARGET_TYPE_FILE";
    /**
     * @generated from enum value: TARGET_TYPE_EXTENSION_RANGE = 2;
     */
    FieldOptions_OptionTargetType[FieldOptions_OptionTargetType["TARGET_TYPE_EXTENSION_RANGE"] = 2] = "TARGET_TYPE_EXTENSION_RANGE";
    /**
     * @generated from enum value: TARGET_TYPE_MESSAGE = 3;
     */
    FieldOptions_OptionTargetType[FieldOptions_OptionTargetType["TARGET_TYPE_MESSAGE"] = 3] = "TARGET_TYPE_MESSAGE";
    /**
     * @generated from enum value: TARGET_TYPE_FIELD = 4;
     */
    FieldOptions_OptionTargetType[FieldOptions_OptionTargetType["TARGET_TYPE_FIELD"] = 4] = "TARGET_TYPE_FIELD";
    /**
     * @generated from enum value: TARGET_TYPE_ONEOF = 5;
     */
    FieldOptions_OptionTargetType[FieldOptions_OptionTargetType["TARGET_TYPE_ONEOF"] = 5] = "TARGET_TYPE_ONEOF";
    /**
     * @generated from enum value: TARGET_TYPE_ENUM = 6;
     */
    FieldOptions_OptionTargetType[FieldOptions_OptionTargetType["TARGET_TYPE_ENUM"] = 6] = "TARGET_TYPE_ENUM";
    /**
     * @generated from enum value: TARGET_TYPE_ENUM_ENTRY = 7;
     */
    FieldOptions_OptionTargetType[FieldOptions_OptionTargetType["TARGET_TYPE_ENUM_ENTRY"] = 7] = "TARGET_TYPE_ENUM_ENTRY";
    /**
     * @generated from enum value: TARGET_TYPE_SERVICE = 8;
     */
    FieldOptions_OptionTargetType[FieldOptions_OptionTargetType["TARGET_TYPE_SERVICE"] = 8] = "TARGET_TYPE_SERVICE";
    /**
     * @generated from enum value: TARGET_TYPE_METHOD = 9;
     */
    FieldOptions_OptionTargetType[FieldOptions_OptionTargetType["TARGET_TYPE_METHOD"] = 9] = "TARGET_TYPE_METHOD";
})(FieldOptions_OptionTargetType || (FieldOptions_OptionTargetType = {}));
/**
 * Is this method side-effect-free (or safe in HTTP parlance), or idempotent,
 * or neither? HTTP based RPC implementation may choose GET verb for safe
 * methods, and PUT verb for idempotent methods instead of the default POST.
 *
 * @generated from enum google.protobuf.MethodOptions.IdempotencyLevel
 */
var MethodOptions_IdempotencyLevel;
(function (MethodOptions_IdempotencyLevel) {
    /**
     * @generated from enum value: IDEMPOTENCY_UNKNOWN = 0;
     */
    MethodOptions_IdempotencyLevel[MethodOptions_IdempotencyLevel["IDEMPOTENCY_UNKNOWN"] = 0] = "IDEMPOTENCY_UNKNOWN";
    /**
     * implies idempotent
     *
     * @generated from enum value: NO_SIDE_EFFECTS = 1;
     */
    MethodOptions_IdempotencyLevel[MethodOptions_IdempotencyLevel["NO_SIDE_EFFECTS"] = 1] = "NO_SIDE_EFFECTS";
    /**
     * idempotent, but may have side effects
     *
     * @generated from enum value: IDEMPOTENT = 2;
     */
    MethodOptions_IdempotencyLevel[MethodOptions_IdempotencyLevel["IDEMPOTENT"] = 2] = "IDEMPOTENT";
})(MethodOptions_IdempotencyLevel || (MethodOptions_IdempotencyLevel = {}));
/**
 * @generated from enum google.protobuf.FeatureSet.VisibilityFeature.DefaultSymbolVisibility
 */
var FeatureSet_VisibilityFeature_DefaultSymbolVisibility;
(function (FeatureSet_VisibilityFeature_DefaultSymbolVisibility) {
    /**
     * @generated from enum value: DEFAULT_SYMBOL_VISIBILITY_UNKNOWN = 0;
     */
    FeatureSet_VisibilityFeature_DefaultSymbolVisibility[FeatureSet_VisibilityFeature_DefaultSymbolVisibility["DEFAULT_SYMBOL_VISIBILITY_UNKNOWN"] = 0] = "DEFAULT_SYMBOL_VISIBILITY_UNKNOWN";
    /**
     * Default pre-EDITION_2024, all UNSET visibility are export.
     *
     * @generated from enum value: EXPORT_ALL = 1;
     */
    FeatureSet_VisibilityFeature_DefaultSymbolVisibility[FeatureSet_VisibilityFeature_DefaultSymbolVisibility["EXPORT_ALL"] = 1] = "EXPORT_ALL";
    /**
     * All top-level symbols default to export, nested default to local.
     *
     * @generated from enum value: EXPORT_TOP_LEVEL = 2;
     */
    FeatureSet_VisibilityFeature_DefaultSymbolVisibility[FeatureSet_VisibilityFeature_DefaultSymbolVisibility["EXPORT_TOP_LEVEL"] = 2] = "EXPORT_TOP_LEVEL";
    /**
     * All symbols default to local.
     *
     * @generated from enum value: LOCAL_ALL = 3;
     */
    FeatureSet_VisibilityFeature_DefaultSymbolVisibility[FeatureSet_VisibilityFeature_DefaultSymbolVisibility["LOCAL_ALL"] = 3] = "LOCAL_ALL";
    /**
     * All symbols local by default. Nested types cannot be exported.
     * With special case caveat for message { enum {} reserved 1 to max; }
     * This is the recommended setting for new protos.
     *
     * @generated from enum value: STRICT = 4;
     */
    FeatureSet_VisibilityFeature_DefaultSymbolVisibility[FeatureSet_VisibilityFeature_DefaultSymbolVisibility["STRICT"] = 4] = "STRICT";
})(FeatureSet_VisibilityFeature_DefaultSymbolVisibility || (FeatureSet_VisibilityFeature_DefaultSymbolVisibility = {}));
/**
 * @generated from enum google.protobuf.FeatureSet.FieldPresence
 */
var FeatureSet_FieldPresence;
(function (FeatureSet_FieldPresence) {
    /**
     * @generated from enum value: FIELD_PRESENCE_UNKNOWN = 0;
     */
    FeatureSet_FieldPresence[FeatureSet_FieldPresence["FIELD_PRESENCE_UNKNOWN"] = 0] = "FIELD_PRESENCE_UNKNOWN";
    /**
     * @generated from enum value: EXPLICIT = 1;
     */
    FeatureSet_FieldPresence[FeatureSet_FieldPresence["EXPLICIT"] = 1] = "EXPLICIT";
    /**
     * @generated from enum value: IMPLICIT = 2;
     */
    FeatureSet_FieldPresence[FeatureSet_FieldPresence["IMPLICIT"] = 2] = "IMPLICIT";
    /**
     * @generated from enum value: LEGACY_REQUIRED = 3;
     */
    FeatureSet_FieldPresence[FeatureSet_FieldPresence["LEGACY_REQUIRED"] = 3] = "LEGACY_REQUIRED";
})(FeatureSet_FieldPresence || (FeatureSet_FieldPresence = {}));
/**
 * @generated from enum google.protobuf.FeatureSet.EnumType
 */
var FeatureSet_EnumType;
(function (FeatureSet_EnumType) {
    /**
     * @generated from enum value: ENUM_TYPE_UNKNOWN = 0;
     */
    FeatureSet_EnumType[FeatureSet_EnumType["ENUM_TYPE_UNKNOWN"] = 0] = "ENUM_TYPE_UNKNOWN";
    /**
     * @generated from enum value: OPEN = 1;
     */
    FeatureSet_EnumType[FeatureSet_EnumType["OPEN"] = 1] = "OPEN";
    /**
     * @generated from enum value: CLOSED = 2;
     */
    FeatureSet_EnumType[FeatureSet_EnumType["CLOSED"] = 2] = "CLOSED";
})(FeatureSet_EnumType || (FeatureSet_EnumType = {}));
/**
 * @generated from enum google.protobuf.FeatureSet.RepeatedFieldEncoding
 */
var FeatureSet_RepeatedFieldEncoding;
(function (FeatureSet_RepeatedFieldEncoding) {
    /**
     * @generated from enum value: REPEATED_FIELD_ENCODING_UNKNOWN = 0;
     */
    FeatureSet_RepeatedFieldEncoding[FeatureSet_RepeatedFieldEncoding["REPEATED_FIELD_ENCODING_UNKNOWN"] = 0] = "REPEATED_FIELD_ENCODING_UNKNOWN";
    /**
     * @generated from enum value: PACKED = 1;
     */
    FeatureSet_RepeatedFieldEncoding[FeatureSet_RepeatedFieldEncoding["PACKED"] = 1] = "PACKED";
    /**
     * @generated from enum value: EXPANDED = 2;
     */
    FeatureSet_RepeatedFieldEncoding[FeatureSet_RepeatedFieldEncoding["EXPANDED"] = 2] = "EXPANDED";
})(FeatureSet_RepeatedFieldEncoding || (FeatureSet_RepeatedFieldEncoding = {}));
/**
 * @generated from enum google.protobuf.FeatureSet.Utf8Validation
 */
var FeatureSet_Utf8Validation;
(function (FeatureSet_Utf8Validation) {
    /**
     * @generated from enum value: UTF8_VALIDATION_UNKNOWN = 0;
     */
    FeatureSet_Utf8Validation[FeatureSet_Utf8Validation["UTF8_VALIDATION_UNKNOWN"] = 0] = "UTF8_VALIDATION_UNKNOWN";
    /**
     * @generated from enum value: VERIFY = 2;
     */
    FeatureSet_Utf8Validation[FeatureSet_Utf8Validation["VERIFY"] = 2] = "VERIFY";
    /**
     * @generated from enum value: NONE = 3;
     */
    FeatureSet_Utf8Validation[FeatureSet_Utf8Validation["NONE"] = 3] = "NONE";
})(FeatureSet_Utf8Validation || (FeatureSet_Utf8Validation = {}));
/**
 * @generated from enum google.protobuf.FeatureSet.MessageEncoding
 */
var FeatureSet_MessageEncoding;
(function (FeatureSet_MessageEncoding) {
    /**
     * @generated from enum value: MESSAGE_ENCODING_UNKNOWN = 0;
     */
    FeatureSet_MessageEncoding[FeatureSet_MessageEncoding["MESSAGE_ENCODING_UNKNOWN"] = 0] = "MESSAGE_ENCODING_UNKNOWN";
    /**
     * @generated from enum value: LENGTH_PREFIXED = 1;
     */
    FeatureSet_MessageEncoding[FeatureSet_MessageEncoding["LENGTH_PREFIXED"] = 1] = "LENGTH_PREFIXED";
    /**
     * @generated from enum value: DELIMITED = 2;
     */
    FeatureSet_MessageEncoding[FeatureSet_MessageEncoding["DELIMITED"] = 2] = "DELIMITED";
})(FeatureSet_MessageEncoding || (FeatureSet_MessageEncoding = {}));
/**
 * @generated from enum google.protobuf.FeatureSet.JsonFormat
 */
var FeatureSet_JsonFormat;
(function (FeatureSet_JsonFormat) {
    /**
     * @generated from enum value: JSON_FORMAT_UNKNOWN = 0;
     */
    FeatureSet_JsonFormat[FeatureSet_JsonFormat["JSON_FORMAT_UNKNOWN"] = 0] = "JSON_FORMAT_UNKNOWN";
    /**
     * @generated from enum value: ALLOW = 1;
     */
    FeatureSet_JsonFormat[FeatureSet_JsonFormat["ALLOW"] = 1] = "ALLOW";
    /**
     * @generated from enum value: LEGACY_BEST_EFFORT = 2;
     */
    FeatureSet_JsonFormat[FeatureSet_JsonFormat["LEGACY_BEST_EFFORT"] = 2] = "LEGACY_BEST_EFFORT";
})(FeatureSet_JsonFormat || (FeatureSet_JsonFormat = {}));
/**
 * @generated from enum google.protobuf.FeatureSet.EnforceNamingStyle
 */
var FeatureSet_EnforceNamingStyle;
(function (FeatureSet_EnforceNamingStyle) {
    /**
     * @generated from enum value: ENFORCE_NAMING_STYLE_UNKNOWN = 0;
     */
    FeatureSet_EnforceNamingStyle[FeatureSet_EnforceNamingStyle["ENFORCE_NAMING_STYLE_UNKNOWN"] = 0] = "ENFORCE_NAMING_STYLE_UNKNOWN";
    /**
     * @generated from enum value: STYLE2024 = 1;
     */
    FeatureSet_EnforceNamingStyle[FeatureSet_EnforceNamingStyle["STYLE2024"] = 1] = "STYLE2024";
    /**
     * @generated from enum value: STYLE_LEGACY = 2;
     */
    FeatureSet_EnforceNamingStyle[FeatureSet_EnforceNamingStyle["STYLE_LEGACY"] = 2] = "STYLE_LEGACY";
})(FeatureSet_EnforceNamingStyle || (FeatureSet_EnforceNamingStyle = {}));
/**
 * Represents the identified object's effect on the element in the original
 * .proto file.
 *
 * @generated from enum google.protobuf.GeneratedCodeInfo.Annotation.Semantic
 */
var GeneratedCodeInfo_Annotation_Semantic;
(function (GeneratedCodeInfo_Annotation_Semantic) {
    /**
     * There is no effect or the effect is indescribable.
     *
     * @generated from enum value: NONE = 0;
     */
    GeneratedCodeInfo_Annotation_Semantic[GeneratedCodeInfo_Annotation_Semantic["NONE"] = 0] = "NONE";
    /**
     * The element is set or otherwise mutated.
     *
     * @generated from enum value: SET = 1;
     */
    GeneratedCodeInfo_Annotation_Semantic[GeneratedCodeInfo_Annotation_Semantic["SET"] = 1] = "SET";
    /**
     * An alias to the element is returned.
     *
     * @generated from enum value: ALIAS = 2;
     */
    GeneratedCodeInfo_Annotation_Semantic[GeneratedCodeInfo_Annotation_Semantic["ALIAS"] = 2] = "ALIAS";
})(GeneratedCodeInfo_Annotation_Semantic || (GeneratedCodeInfo_Annotation_Semantic = {}));
/**
 * The full set of known editions.
 *
 * @generated from enum google.protobuf.Edition
 */
var Edition;
(function (Edition) {
    /**
     * A placeholder for an unknown edition value.
     *
     * @generated from enum value: EDITION_UNKNOWN = 0;
     */
    Edition[Edition["EDITION_UNKNOWN"] = 0] = "EDITION_UNKNOWN";
    /**
     * A placeholder edition for specifying default behaviors *before* a feature
     * was first introduced.  This is effectively an "infinite past".
     *
     * @generated from enum value: EDITION_LEGACY = 900;
     */
    Edition[Edition["EDITION_LEGACY"] = 900] = "EDITION_LEGACY";
    /**
     * Legacy syntax "editions".  These pre-date editions, but behave much like
     * distinct editions.  These can't be used to specify the edition of proto
     * files, but feature definitions must supply proto2/proto3 defaults for
     * backwards compatibility.
     *
     * @generated from enum value: EDITION_PROTO2 = 998;
     */
    Edition[Edition["EDITION_PROTO2"] = 998] = "EDITION_PROTO2";
    /**
     * @generated from enum value: EDITION_PROTO3 = 999;
     */
    Edition[Edition["EDITION_PROTO3"] = 999] = "EDITION_PROTO3";
    /**
     * Editions that have been released.  The specific values are arbitrary and
     * should not be depended on, but they will always be time-ordered for easy
     * comparison.
     *
     * @generated from enum value: EDITION_2023 = 1000;
     */
    Edition[Edition["EDITION_2023"] = 1000] = "EDITION_2023";
    /**
     * @generated from enum value: EDITION_2024 = 1001;
     */
    Edition[Edition["EDITION_2024"] = 1001] = "EDITION_2024";
    /**
     * Placeholder editions for testing feature resolution.  These should not be
     * used or relied on outside of tests.
     *
     * @generated from enum value: EDITION_1_TEST_ONLY = 1;
     */
    Edition[Edition["EDITION_1_TEST_ONLY"] = 1] = "EDITION_1_TEST_ONLY";
    /**
     * @generated from enum value: EDITION_2_TEST_ONLY = 2;
     */
    Edition[Edition["EDITION_2_TEST_ONLY"] = 2] = "EDITION_2_TEST_ONLY";
    /**
     * @generated from enum value: EDITION_99997_TEST_ONLY = 99997;
     */
    Edition[Edition["EDITION_99997_TEST_ONLY"] = 99997] = "EDITION_99997_TEST_ONLY";
    /**
     * @generated from enum value: EDITION_99998_TEST_ONLY = 99998;
     */
    Edition[Edition["EDITION_99998_TEST_ONLY"] = 99998] = "EDITION_99998_TEST_ONLY";
    /**
     * @generated from enum value: EDITION_99999_TEST_ONLY = 99999;
     */
    Edition[Edition["EDITION_99999_TEST_ONLY"] = 99999] = "EDITION_99999_TEST_ONLY";
    /**
     * Placeholder for specifying unbounded edition support.  This should only
     * ever be used by plugins that can expect to never require any changes to
     * support a new edition.
     *
     * @generated from enum value: EDITION_MAX = 2147483647;
     */
    Edition[Edition["EDITION_MAX"] = 2147483647] = "EDITION_MAX";
})(Edition || (Edition = {}));
/**
 * Describes the 'visibility' of a symbol with respect to the proto import
 * system. Symbols can only be imported when the visibility rules do not prevent
 * it (ex: local symbols cannot be imported).  Visibility modifiers can only set
 * on `message` and `enum` as they are the only types available to be referenced
 * from other files.
 *
 * @generated from enum google.protobuf.SymbolVisibility
 */
var SymbolVisibility;
(function (SymbolVisibility) {
    /**
     * @generated from enum value: VISIBILITY_UNSET = 0;
     */
    SymbolVisibility[SymbolVisibility["VISIBILITY_UNSET"] = 0] = "VISIBILITY_UNSET";
    /**
     * @generated from enum value: VISIBILITY_LOCAL = 1;
     */
    SymbolVisibility[SymbolVisibility["VISIBILITY_LOCAL"] = 1] = "VISIBILITY_LOCAL";
    /**
     * @generated from enum value: VISIBILITY_EXPORT = 2;
     */
    SymbolVisibility[SymbolVisibility["VISIBILITY_EXPORT"] = 2] = "VISIBILITY_EXPORT";
})(SymbolVisibility || (SymbolVisibility = {}));

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// Default options for parsing binary data.
const readDefaults = {
    readUnknownFields: true,
};
function makeReadOptions$1(options) {
    return options ? Object.assign(Object.assign({}, readDefaults), options) : readDefaults;
}
/**
 * Parse serialized binary data.
 */
function fromBinary(schema, bytes, options) {
    const msg = reflect(schema, undefined, false);
    readMessage$1(msg, new BinaryReader(bytes), makeReadOptions$1(options), false, bytes.byteLength);
    return msg.message;
}
/**
 * If `delimited` is false, read the length given in `lengthOrDelimitedFieldNo`.
 *
 * If `delimited` is true, read until an EndGroup tag. `lengthOrDelimitedFieldNo`
 * is the expected field number.
 *
 * @private
 */
function readMessage$1(message, reader, options, delimited, lengthOrDelimitedFieldNo) {
    var _a;
    const end = delimited ? reader.len : reader.pos + lengthOrDelimitedFieldNo;
    let fieldNo;
    let wireType;
    const unknownFields = (_a = message.getUnknown()) !== null && _a !== void 0 ? _a : [];
    while (reader.pos < end) {
        [fieldNo, wireType] = reader.tag();
        if (delimited && wireType == WireType.EndGroup) {
            break;
        }
        const field = message.findNumber(fieldNo);
        if (!field) {
            const data = reader.skip(wireType, fieldNo);
            if (options.readUnknownFields) {
                unknownFields.push({ no: fieldNo, wireType, data });
            }
            continue;
        }
        readField$1(message, reader, field, wireType, options);
    }
    if (delimited) {
        if (wireType != WireType.EndGroup || fieldNo !== lengthOrDelimitedFieldNo) {
            throw new Error("invalid end group tag");
        }
    }
    if (unknownFields.length > 0) {
        message.setUnknown(unknownFields);
    }
}
/**
 * @private
 */
function readField$1(message, reader, field, wireType, options) {
    var _a;
    switch (field.fieldKind) {
        case "scalar":
            message.set(field, readScalar(reader, field.scalar));
            break;
        case "enum":
            const val = readScalar(reader, ScalarType.INT32);
            if (field.enum.open) {
                message.set(field, val);
            }
            else {
                const ok = field.enum.values.some((v) => v.number === val);
                if (ok) {
                    message.set(field, val);
                }
                else if (options.readUnknownFields) {
                    const bytes = [];
                    varint32write(val, bytes);
                    const unknownFields = (_a = message.getUnknown()) !== null && _a !== void 0 ? _a : [];
                    unknownFields.push({
                        no: field.number,
                        wireType,
                        data: new Uint8Array(bytes),
                    });
                    message.setUnknown(unknownFields);
                }
            }
            break;
        case "message":
            message.set(field, readMessageField$1(reader, options, field, message.get(field)));
            break;
        case "list":
            readListField$1(reader, wireType, message.get(field), options);
            break;
        case "map":
            readMapEntry(reader, message.get(field), options);
            break;
    }
}
// Read a map field, expecting key field = 1, value field = 2
function readMapEntry(reader, map, options) {
    const field = map.field();
    let key;
    let val;
    // Read the length of the map entry, which is a varint.
    const len = reader.uint32();
    // WARNING: Calculate end AFTER advancing reader.pos (above), so that
    //          reader.pos is at the start of the map entry.
    const end = reader.pos + len;
    while (reader.pos < end) {
        const [fieldNo] = reader.tag();
        switch (fieldNo) {
            case 1:
                key = readScalar(reader, field.mapKey);
                break;
            case 2:
                switch (field.mapKind) {
                    case "scalar":
                        val = readScalar(reader, field.scalar);
                        break;
                    case "enum":
                        val = reader.int32();
                        break;
                    case "message":
                        val = readMessageField$1(reader, options, field);
                        break;
                }
                break;
        }
    }
    if (key === undefined) {
        key = scalarZeroValue(field.mapKey, false);
    }
    if (val === undefined) {
        switch (field.mapKind) {
            case "scalar":
                val = scalarZeroValue(field.scalar, false);
                break;
            case "enum":
                val = field.enum.values[0].number;
                break;
            case "message":
                val = reflect(field.message, undefined, false);
                break;
        }
    }
    map.set(key, val);
}
function readListField$1(reader, wireType, list, options) {
    var _a;
    const field = list.field();
    if (field.listKind === "message") {
        list.add(readMessageField$1(reader, options, field));
        return;
    }
    const scalarType = (_a = field.scalar) !== null && _a !== void 0 ? _a : ScalarType.INT32;
    const packed = wireType == WireType.LengthDelimited &&
        scalarType != ScalarType.STRING &&
        scalarType != ScalarType.BYTES;
    if (!packed) {
        list.add(readScalar(reader, scalarType));
        return;
    }
    const e = reader.uint32() + reader.pos;
    while (reader.pos < e) {
        list.add(readScalar(reader, scalarType));
    }
}
function readMessageField$1(reader, options, field, mergeMessage) {
    const delimited = field.delimitedEncoding;
    const message = mergeMessage !== null && mergeMessage !== void 0 ? mergeMessage : reflect(field.message, undefined, false);
    readMessage$1(message, reader, options, delimited, delimited ? field.number : reader.uint32());
    return message;
}
function readScalar(reader, type) {
    switch (type) {
        case ScalarType.STRING:
            return reader.string();
        case ScalarType.BOOL:
            return reader.bool();
        case ScalarType.DOUBLE:
            return reader.double();
        case ScalarType.FLOAT:
            return reader.float();
        case ScalarType.INT32:
            return reader.int32();
        case ScalarType.INT64:
            return reader.int64();
        case ScalarType.UINT64:
            return reader.uint64();
        case ScalarType.FIXED64:
            return reader.fixed64();
        case ScalarType.BYTES:
            return reader.bytes();
        case ScalarType.FIXED32:
            return reader.fixed32();
        case ScalarType.SFIXED32:
            return reader.sfixed32();
        case ScalarType.SFIXED64:
            return reader.sfixed64();
        case ScalarType.SINT64:
            return reader.sint64();
        case ScalarType.UINT32:
            return reader.uint32();
        case ScalarType.SINT32:
            return reader.sint32();
    }
}

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Hydrate a file descriptor.
 *
 * @private
 */
function fileDesc(b64, imports) {
    var _a;
    const root = fromBinary(FileDescriptorProtoSchema, base64Decode(b64));
    root.messageType.forEach(restoreJsonNames);
    root.dependency = (_a = imports === null || imports === void 0 ? void 0 : imports.map((f) => f.proto.name)) !== null && _a !== void 0 ? _a : [];
    const reg = createFileRegistry(root, (protoFileName) => imports === null || imports === void 0 ? void 0 : imports.find((f) => f.proto.name === protoFileName));
    // biome-ignore lint/style/noNonNullAssertion: non-null assertion because we just created the registry from the file we look up
    return reg.getFile(root.name);
}

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Describes the file google/protobuf/timestamp.proto.
 */
const file_google_protobuf_timestamp = /*@__PURE__*/ fileDesc("Ch9nb29nbGUvcHJvdG9idWYvdGltZXN0YW1wLnByb3RvEg9nb29nbGUucHJvdG9idWYiKwoJVGltZXN0YW1wEg8KB3NlY29uZHMYASABKAMSDQoFbmFub3MYAiABKAVChQEKE2NvbS5nb29nbGUucHJvdG9idWZCDlRpbWVzdGFtcFByb3RvUAFaMmdvb2dsZS5nb2xhbmcub3JnL3Byb3RvYnVmL3R5cGVzL2tub3duL3RpbWVzdGFtcHBi+AEBogIDR1BCqgIeR29vZ2xlLlByb3RvYnVmLldlbGxLbm93blR5cGVzYgZwcm90bzM");

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Describes the file google/protobuf/any.proto.
 */
const file_google_protobuf_any = /*@__PURE__*/ fileDesc("Chlnb29nbGUvcHJvdG9idWYvYW55LnByb3RvEg9nb29nbGUucHJvdG9idWYiJgoDQW55EhAKCHR5cGVfdXJsGAEgASgJEg0KBXZhbHVlGAIgASgMQnYKE2NvbS5nb29nbGUucHJvdG9idWZCCEFueVByb3RvUAFaLGdvb2dsZS5nb2xhbmcub3JnL3Byb3RvYnVmL3R5cGVzL2tub3duL2FueXBiogIDR1BCqgIeR29vZ2xlLlByb3RvYnVmLldlbGxLbm93blR5cGVzYgZwcm90bzM");
/**
 * Describes the message google.protobuf.Any.
 * Use `create(AnySchema)` to create a new message.
 */
const AnySchema = /*@__PURE__*/ messageDesc(file_google_protobuf_any, 0);

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// bootstrap-inject google.protobuf.FeatureSet.FieldPresence.LEGACY_REQUIRED: const $name: FeatureSet_FieldPresence.$localName = $number;
const LEGACY_REQUIRED$1 = 3;
// Default options for serializing binary data.
const writeDefaults = {
    writeUnknownFields: true,
};
function makeWriteOptions$1(options) {
    return options ? Object.assign(Object.assign({}, writeDefaults), options) : writeDefaults;
}
function toBinary(schema, message, options) {
    return writeFields(new BinaryWriter(), makeWriteOptions$1(options), reflect(schema, message)).finish();
}
function writeFields(writer, opts, msg) {
    var _a;
    for (const f of msg.sortedFields) {
        if (!msg.isSet(f)) {
            if (f.presence == LEGACY_REQUIRED$1) {
                throw new Error(`cannot encode ${f} to binary: required field not set`);
            }
            continue;
        }
        writeField(writer, opts, msg, f);
    }
    if (opts.writeUnknownFields) {
        for (const { no, wireType, data } of (_a = msg.getUnknown()) !== null && _a !== void 0 ? _a : []) {
            writer.tag(no, wireType).raw(data);
        }
    }
    return writer;
}
/**
 * @private
 */
function writeField(writer, opts, msg, field) {
    var _a;
    switch (field.fieldKind) {
        case "scalar":
        case "enum":
            writeScalar(writer, msg.desc.typeName, field.name, (_a = field.scalar) !== null && _a !== void 0 ? _a : ScalarType.INT32, field.number, msg.get(field));
            break;
        case "list":
            writeListField(writer, opts, field, msg.get(field));
            break;
        case "message":
            writeMessageField(writer, opts, field, msg.get(field));
            break;
        case "map":
            for (const [key, val] of msg.get(field)) {
                writeMapEntry(writer, opts, field, key, val);
            }
            break;
    }
}
function writeScalar(writer, msgName, fieldName, scalarType, fieldNo, value) {
    writeScalarValue(writer.tag(fieldNo, writeTypeOfScalar(scalarType)), msgName, fieldName, scalarType, value);
}
function writeMessageField(writer, opts, field, message) {
    if (field.delimitedEncoding) {
        writeFields(writer.tag(field.number, WireType.StartGroup), opts, message).tag(field.number, WireType.EndGroup);
    }
    else {
        writeFields(writer.tag(field.number, WireType.LengthDelimited).fork(), opts, message).join();
    }
}
function writeListField(writer, opts, field, list) {
    var _a;
    if (field.listKind == "message") {
        for (const item of list) {
            writeMessageField(writer, opts, field, item);
        }
        return;
    }
    const scalarType = (_a = field.scalar) !== null && _a !== void 0 ? _a : ScalarType.INT32;
    if (field.packed) {
        if (!list.size) {
            return;
        }
        writer.tag(field.number, WireType.LengthDelimited).fork();
        for (const item of list) {
            writeScalarValue(writer, field.parent.typeName, field.name, scalarType, item);
        }
        writer.join();
        return;
    }
    for (const item of list) {
        writeScalar(writer, field.parent.typeName, field.name, scalarType, field.number, item);
    }
}
function writeMapEntry(writer, opts, field, key, value) {
    var _a;
    writer.tag(field.number, WireType.LengthDelimited).fork();
    // write key, expecting key field number = 1
    writeScalar(writer, field.parent.typeName, field.name, field.mapKey, 1, key);
    // write value, expecting value field number = 2
    switch (field.mapKind) {
        case "scalar":
        case "enum":
            writeScalar(writer, field.parent.typeName, field.name, (_a = field.scalar) !== null && _a !== void 0 ? _a : ScalarType.INT32, 2, value);
            break;
        case "message":
            writeFields(writer.tag(2, WireType.LengthDelimited).fork(), opts, value).join();
            break;
    }
    writer.join();
}
function writeScalarValue(writer, msgName, fieldName, type, value) {
    try {
        switch (type) {
            case ScalarType.STRING:
                writer.string(value);
                break;
            case ScalarType.BOOL:
                writer.bool(value);
                break;
            case ScalarType.DOUBLE:
                writer.double(value);
                break;
            case ScalarType.FLOAT:
                writer.float(value);
                break;
            case ScalarType.INT32:
                writer.int32(value);
                break;
            case ScalarType.INT64:
                writer.int64(value);
                break;
            case ScalarType.UINT64:
                writer.uint64(value);
                break;
            case ScalarType.FIXED64:
                writer.fixed64(value);
                break;
            case ScalarType.BYTES:
                writer.bytes(value);
                break;
            case ScalarType.FIXED32:
                writer.fixed32(value);
                break;
            case ScalarType.SFIXED32:
                writer.sfixed32(value);
                break;
            case ScalarType.SFIXED64:
                writer.sfixed64(value);
                break;
            case ScalarType.SINT64:
                writer.sint64(value);
                break;
            case ScalarType.UINT32:
                writer.uint32(value);
                break;
            case ScalarType.SINT32:
                writer.sint32(value);
                break;
        }
    }
    catch (e) {
        if (e instanceof Error) {
            throw new Error(`cannot encode field ${msgName}.${fieldName} to binary: ${e.message}`);
        }
        throw e;
    }
}
function writeTypeOfScalar(type) {
    switch (type) {
        case ScalarType.BYTES:
        case ScalarType.STRING:
            return WireType.LengthDelimited;
        case ScalarType.DOUBLE:
        case ScalarType.FIXED64:
        case ScalarType.SFIXED64:
            return WireType.Bit64;
        case ScalarType.FIXED32:
        case ScalarType.SFIXED32:
        case ScalarType.FLOAT:
            return WireType.Bit32;
        default:
            return WireType.Varint;
    }
}

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
function anyPack(schema, message, into) {
    let ret = false;
    if (!into) {
        into = create(AnySchema);
        ret = true;
    }
    into.value = toBinary(schema, message);
    into.typeUrl = typeNameToUrl(message.$typeName);
    return ret ? into : undefined;
}
function anyIs(any, descOrTypeName) {
    if (any.typeUrl === "") {
        return false;
    }
    const want = typeof descOrTypeName == "string"
        ? descOrTypeName
        : descOrTypeName.typeName;
    const got = typeUrlToName(any.typeUrl);
    return want === got;
}
function anyUnpack(any, registryOrMessageDesc) {
    if (any.typeUrl === "") {
        return undefined;
    }
    const desc = registryOrMessageDesc.kind == "message"
        ? registryOrMessageDesc
        : registryOrMessageDesc.getMessage(typeUrlToName(any.typeUrl));
    if (!desc || !anyIs(any, desc)) {
        return undefined;
    }
    return fromBinary(desc, any.value);
}
function typeNameToUrl(name) {
    return `type.googleapis.com/${name}`;
}
function typeUrlToName(url) {
    const slash = url.lastIndexOf("/");
    const name = slash >= 0 ? url.substring(slash + 1) : url;
    if (!name.length) {
        throw new Error(`invalid type url: ${url}`);
    }
    return name;
}

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Describes the file google/protobuf/empty.proto.
 */
const file_google_protobuf_empty = /*@__PURE__*/ fileDesc("Chtnb29nbGUvcHJvdG9idWYvZW1wdHkucHJvdG8SD2dvb2dsZS5wcm90b2J1ZiIHCgVFbXB0eUJ9ChNjb20uZ29vZ2xlLnByb3RvYnVmQgpFbXB0eVByb3RvUAFaLmdvb2dsZS5nb2xhbmcub3JnL3Byb3RvYnVmL3R5cGVzL2tub3duL2VtcHR5cGL4AQGiAgNHUEKqAh5Hb29nbGUuUHJvdG9idWYuV2VsbEtub3duVHlwZXNiBnByb3RvMw");

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Describes the file google/protobuf/struct.proto.
 */
const file_google_protobuf_struct = /*@__PURE__*/ fileDesc("Chxnb29nbGUvcHJvdG9idWYvc3RydWN0LnByb3RvEg9nb29nbGUucHJvdG9idWYihAEKBlN0cnVjdBIzCgZmaWVsZHMYASADKAsyIy5nb29nbGUucHJvdG9idWYuU3RydWN0LkZpZWxkc0VudHJ5GkUKC0ZpZWxkc0VudHJ5EgsKA2tleRgBIAEoCRIlCgV2YWx1ZRgCIAEoCzIWLmdvb2dsZS5wcm90b2J1Zi5WYWx1ZToCOAEi6gEKBVZhbHVlEjAKCm51bGxfdmFsdWUYASABKA4yGi5nb29nbGUucHJvdG9idWYuTnVsbFZhbHVlSAASFgoMbnVtYmVyX3ZhbHVlGAIgASgBSAASFgoMc3RyaW5nX3ZhbHVlGAMgASgJSAASFAoKYm9vbF92YWx1ZRgEIAEoCEgAEi8KDHN0cnVjdF92YWx1ZRgFIAEoCzIXLmdvb2dsZS5wcm90b2J1Zi5TdHJ1Y3RIABIwCgpsaXN0X3ZhbHVlGAYgASgLMhouZ29vZ2xlLnByb3RvYnVmLkxpc3RWYWx1ZUgAQgYKBGtpbmQiMwoJTGlzdFZhbHVlEiYKBnZhbHVlcxgBIAMoCzIWLmdvb2dsZS5wcm90b2J1Zi5WYWx1ZSobCglOdWxsVmFsdWUSDgoKTlVMTF9WQUxVRRAAQn8KE2NvbS5nb29nbGUucHJvdG9idWZCC1N0cnVjdFByb3RvUAFaL2dvb2dsZS5nb2xhbmcub3JnL3Byb3RvYnVmL3R5cGVzL2tub3duL3N0cnVjdHBi+AEBogIDR1BCqgIeR29vZ2xlLlByb3RvYnVmLldlbGxLbm93blR5cGVzYgZwcm90bzM");
/**
 * Describes the message google.protobuf.Struct.
 * Use `create(StructSchema)` to create a new message.
 */
const StructSchema = /*@__PURE__*/ messageDesc(file_google_protobuf_struct, 0);
/**
 * Describes the message google.protobuf.Value.
 * Use `create(ValueSchema)` to create a new message.
 */
const ValueSchema = /*@__PURE__*/ messageDesc(file_google_protobuf_struct, 1);
/**
 * Describes the message google.protobuf.ListValue.
 * Use `create(ListValueSchema)` to create a new message.
 */
const ListValueSchema = /*@__PURE__*/ messageDesc(file_google_protobuf_struct, 2);
/**
 * `NullValue` is a singleton enumeration to represent the null value for the
 * `Value` type union.
 *
 * The JSON representation for `NullValue` is JSON `null`.
 *
 * @generated from enum google.protobuf.NullValue
 */
var NullValue;
(function (NullValue) {
    /**
     * Null value.
     *
     * @generated from enum value: NULL_VALUE = 0;
     */
    NullValue[NullValue["NULL_VALUE"] = 0] = "NULL_VALUE";
})(NullValue || (NullValue = {}));

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Retrieve an extension value from a message.
 *
 * The function never returns undefined. Use hasExtension() to check whether an
 * extension is set. If the extension is not set, this function returns the
 * default value (if one was specified in the protobuf source), or the zero value
 * (for example `0` for numeric types, `[]` for repeated extension fields, and
 * an empty message instance for message fields).
 *
 * Extensions are stored as unknown fields on a message. To mutate an extension
 * value, make sure to store the new value with setExtension() after mutating.
 *
 * If the extension does not extend the given message, an error is raised.
 */
function getExtension(message, extension) {
    assertExtendee(extension, message);
    const ufs = filterUnknownFields(message.$unknown, extension);
    const [container, field, get] = createExtensionContainer(extension);
    for (const uf of ufs) {
        readField$1(container, new BinaryReader(uf.data), field, uf.wireType, {
            readUnknownFields: true,
        });
    }
    return get();
}
/**
 * Set an extension value on a message. If the message already has a value for
 * this extension, the value is replaced.
 *
 * If the extension does not extend the given message, an error is raised.
 */
function setExtension(message, extension, value) {
    var _a;
    assertExtendee(extension, message);
    const ufs = ((_a = message.$unknown) !== null && _a !== void 0 ? _a : []).filter((uf) => uf.no !== extension.number);
    const [container, field] = createExtensionContainer(extension, value);
    const writer = new BinaryWriter();
    writeField(writer, { writeUnknownFields: true }, container, field);
    const reader = new BinaryReader(writer.finish());
    while (reader.pos < reader.len) {
        const [no, wireType] = reader.tag();
        const data = reader.skip(wireType, no);
        ufs.push({ no, wireType, data });
    }
    message.$unknown = ufs;
}
function filterUnknownFields(unknownFields, extension) {
    if (unknownFields === undefined)
        return [];
    if (extension.fieldKind === "enum" || extension.fieldKind === "scalar") {
        // singular scalar fields do not merge, we pick the last
        for (let i = unknownFields.length - 1; i >= 0; --i) {
            if (unknownFields[i].no == extension.number) {
                return [unknownFields[i]];
            }
        }
        return [];
    }
    return unknownFields.filter((uf) => uf.no === extension.number);
}
/**
 * @private
 */
function createExtensionContainer(extension, value) {
    const localName = extension.typeName;
    const field = Object.assign(Object.assign({}, extension), { kind: "field", parent: extension.extendee, localName });
    const desc = Object.assign(Object.assign({}, extension.extendee), { fields: [field], members: [field], oneofs: [] });
    const container = create(desc, value !== undefined ? { [localName]: value } : undefined);
    return [
        reflect(desc, container),
        field,
        () => {
            const value = container[localName];
            if (value === undefined) {
                // biome-ignore lint/style/noNonNullAssertion: Only message fields are undefined, rest will have a zero value.
                const desc = extension.message;
                if (isWrapperDesc(desc)) {
                    return scalarZeroValue(desc.fields[0].scalar, desc.fields[0].longAsString);
                }
                return create(desc);
            }
            return value;
        },
    ];
}
function assertExtendee(extension, message) {
    if (extension.extendee.typeName != message.$typeName) {
        throw new Error(`extension ${extension.typeName} can only be applied to message ${extension.extendee.typeName}`);
    }
}

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// bootstrap-inject google.protobuf.FeatureSet.FieldPresence.LEGACY_REQUIRED: const $name: FeatureSet_FieldPresence.$localName = $number;
const LEGACY_REQUIRED = 3;
// bootstrap-inject google.protobuf.FeatureSet.FieldPresence.IMPLICIT: const $name: FeatureSet_FieldPresence.$localName = $number;
const IMPLICIT = 2;
// Default options for serializing to JSON.
const jsonWriteDefaults = {
    alwaysEmitImplicit: false,
    enumAsInteger: false,
    useProtoFieldName: false,
};
function makeWriteOptions(options) {
    return options ? Object.assign(Object.assign({}, jsonWriteDefaults), options) : jsonWriteDefaults;
}
/**
 * Serialize the message to a JSON value, a JavaScript value that can be
 * passed to JSON.stringify().
 */
function toJson(schema, message, options) {
    return reflectToJson(reflect(schema, message), makeWriteOptions(options));
}
/**
 * Serialize the message to a JSON string.
 */
function toJsonString(schema, message, options) {
    var _a;
    const jsonValue = toJson(schema, message, options);
    return JSON.stringify(jsonValue, null, (_a = options === null || options === void 0 ? void 0 : options.prettySpaces) !== null && _a !== void 0 ? _a : 0);
}
function reflectToJson(msg, opts) {
    var _a;
    const wktJson = tryWktToJson(msg, opts);
    if (wktJson !== undefined)
        return wktJson;
    const json = {};
    for (const f of msg.sortedFields) {
        if (!msg.isSet(f)) {
            if (f.presence == LEGACY_REQUIRED) {
                throw new Error(`cannot encode ${f} to JSON: required field not set`);
            }
            if (!opts.alwaysEmitImplicit || f.presence !== IMPLICIT) {
                // Fields with implicit presence omit zero values (e.g. empty string) by default
                continue;
            }
        }
        const jsonValue = fieldToJson(f, msg.get(f), opts);
        if (jsonValue !== undefined) {
            json[jsonName(f, opts)] = jsonValue;
        }
    }
    if (opts.registry) {
        const tagSeen = new Set();
        for (const { no } of (_a = msg.getUnknown()) !== null && _a !== void 0 ? _a : []) {
            // Same tag can appear multiple times, so we
            // keep track and skip identical ones.
            if (!tagSeen.has(no)) {
                tagSeen.add(no);
                const extension = opts.registry.getExtensionFor(msg.desc, no);
                if (!extension) {
                    continue;
                }
                const value = getExtension(msg.message, extension);
                const [container, field] = createExtensionContainer(extension, value);
                const jsonValue = fieldToJson(field, container.get(field), opts);
                if (jsonValue !== undefined) {
                    json[extension.jsonName] = jsonValue;
                }
            }
        }
    }
    return json;
}
function fieldToJson(f, val, opts) {
    switch (f.fieldKind) {
        case "scalar":
            return scalarToJson(f, val);
        case "message":
            return reflectToJson(val, opts);
        case "enum":
            return enumToJsonInternal(f.enum, val, opts.enumAsInteger);
        case "list":
            return listToJson(val, opts);
        case "map":
            return mapToJson(val, opts);
    }
}
function mapToJson(map, opts) {
    const f = map.field();
    const jsonObj = {};
    switch (f.mapKind) {
        case "scalar":
            for (const [entryKey, entryValue] of map) {
                jsonObj[entryKey] = scalarToJson(f, entryValue);
            }
            break;
        case "message":
            for (const [entryKey, entryValue] of map) {
                jsonObj[entryKey] = reflectToJson(entryValue, opts);
            }
            break;
        case "enum":
            for (const [entryKey, entryValue] of map) {
                jsonObj[entryKey] = enumToJsonInternal(f.enum, entryValue, opts.enumAsInteger);
            }
            break;
    }
    return opts.alwaysEmitImplicit || map.size > 0 ? jsonObj : undefined;
}
function listToJson(list, opts) {
    const f = list.field();
    const jsonArr = [];
    switch (f.listKind) {
        case "scalar":
            for (const item of list) {
                jsonArr.push(scalarToJson(f, item));
            }
            break;
        case "enum":
            for (const item of list) {
                jsonArr.push(enumToJsonInternal(f.enum, item, opts.enumAsInteger));
            }
            break;
        case "message":
            for (const item of list) {
                jsonArr.push(reflectToJson(item, opts));
            }
            break;
    }
    return opts.alwaysEmitImplicit || jsonArr.length > 0 ? jsonArr : undefined;
}
function enumToJsonInternal(desc, value, enumAsInteger) {
    var _a;
    if (typeof value != "number") {
        throw new Error(`cannot encode ${desc} to JSON: expected number, got ${formatVal(value)}`);
    }
    if (desc.typeName == "google.protobuf.NullValue") {
        return null;
    }
    if (enumAsInteger) {
        return value;
    }
    const val = desc.value[value];
    return (_a = val === null || val === void 0 ? void 0 : val.name) !== null && _a !== void 0 ? _a : value; // if we don't know the enum value, just return the number
}
function scalarToJson(field, value) {
    var _a, _b, _c, _d, _e, _f;
    switch (field.scalar) {
        // int32, fixed32, uint32: JSON value will be a decimal number. Either numbers or strings are accepted.
        case ScalarType.INT32:
        case ScalarType.SFIXED32:
        case ScalarType.SINT32:
        case ScalarType.FIXED32:
        case ScalarType.UINT32:
            if (typeof value != "number") {
                throw new Error(`cannot encode ${field} to JSON: ${(_a = checkField(field, value)) === null || _a === void 0 ? void 0 : _a.message}`);
            }
            return value;
        // float, double: JSON value will be a number or one of the special string values "NaN", "Infinity", and "-Infinity".
        // Either numbers or strings are accepted. Exponent notation is also accepted.
        case ScalarType.FLOAT:
        case ScalarType.DOUBLE: // eslint-disable-line no-fallthrough
            if (typeof value != "number") {
                throw new Error(`cannot encode ${field} to JSON: ${(_b = checkField(field, value)) === null || _b === void 0 ? void 0 : _b.message}`);
            }
            if (Number.isNaN(value))
                return "NaN";
            if (value === Number.POSITIVE_INFINITY)
                return "Infinity";
            if (value === Number.NEGATIVE_INFINITY)
                return "-Infinity";
            return value;
        // string:
        case ScalarType.STRING:
            if (typeof value != "string") {
                throw new Error(`cannot encode ${field} to JSON: ${(_c = checkField(field, value)) === null || _c === void 0 ? void 0 : _c.message}`);
            }
            return value;
        // bool:
        case ScalarType.BOOL:
            if (typeof value != "boolean") {
                throw new Error(`cannot encode ${field} to JSON: ${(_d = checkField(field, value)) === null || _d === void 0 ? void 0 : _d.message}`);
            }
            return value;
        // JSON value will be a decimal string. Either numbers or strings are accepted.
        case ScalarType.UINT64:
        case ScalarType.FIXED64:
        case ScalarType.INT64:
        case ScalarType.SFIXED64:
        case ScalarType.SINT64:
            if (typeof value != "bigint" && typeof value != "string") {
                throw new Error(`cannot encode ${field} to JSON: ${(_e = checkField(field, value)) === null || _e === void 0 ? void 0 : _e.message}`);
            }
            return value.toString();
        // bytes: JSON value will be the data encoded as a string using standard base64 encoding with paddings.
        // Either standard or URL-safe base64 encoding with/without paddings are accepted.
        case ScalarType.BYTES:
            if (value instanceof Uint8Array) {
                return base64Encode(value);
            }
            throw new Error(`cannot encode ${field} to JSON: ${(_f = checkField(field, value)) === null || _f === void 0 ? void 0 : _f.message}`);
    }
}
function jsonName(f, opts) {
    return opts.useProtoFieldName ? f.name : f.jsonName;
}
// returns a json value if wkt, otherwise returns undefined.
function tryWktToJson(msg, opts) {
    if (!msg.desc.typeName.startsWith("google.protobuf.")) {
        return undefined;
    }
    switch (msg.desc.typeName) {
        case "google.protobuf.Any":
            return anyToJson(msg.message, opts);
        case "google.protobuf.Timestamp":
            return timestampToJson(msg.message);
        case "google.protobuf.Duration":
            return durationToJson(msg.message);
        case "google.protobuf.FieldMask":
            return fieldMaskToJson(msg.message);
        case "google.protobuf.Struct":
            return structToJson(msg.message);
        case "google.protobuf.Value":
            return valueToJson(msg.message);
        case "google.protobuf.ListValue":
            return listValueToJson(msg.message);
        default:
            if (isWrapperDesc(msg.desc)) {
                const valueField = msg.desc.fields[0];
                return scalarToJson(valueField, msg.get(valueField));
            }
            return undefined;
    }
}
function anyToJson(val, opts) {
    if (val.typeUrl === "") {
        return {};
    }
    const { registry } = opts;
    let message;
    let desc;
    if (registry) {
        message = anyUnpack(val, registry);
        if (message) {
            desc = registry.getMessage(message.$typeName);
        }
    }
    if (!desc || !message) {
        throw new Error(`cannot encode message ${val.$typeName} to JSON: "${val.typeUrl}" is not in the type registry`);
    }
    let json = reflectToJson(reflect(desc, message), opts);
    if (desc.typeName.startsWith("google.protobuf.") ||
        json === null ||
        Array.isArray(json) ||
        typeof json !== "object") {
        json = { value: json };
    }
    json["@type"] = val.typeUrl;
    return json;
}
function durationToJson(val) {
    if (Number(val.seconds) > 315576000000 ||
        Number(val.seconds) < -315576e6) {
        throw new Error(`cannot encode message ${val.$typeName} to JSON: value out of range`);
    }
    let text = val.seconds.toString();
    if (val.nanos !== 0) {
        let nanosStr = Math.abs(val.nanos).toString();
        nanosStr = "0".repeat(9 - nanosStr.length) + nanosStr;
        if (nanosStr.substring(3) === "000000") {
            nanosStr = nanosStr.substring(0, 3);
        }
        else if (nanosStr.substring(6) === "000") {
            nanosStr = nanosStr.substring(0, 6);
        }
        text += "." + nanosStr;
        if (val.nanos < 0 && Number(val.seconds) == 0) {
            text = "-" + text;
        }
    }
    return text + "s";
}
function fieldMaskToJson(val) {
    return val.paths
        .map((p) => {
        if (p.match(/_[0-9]?_/g) || p.match(/[A-Z]/g)) {
            throw new Error(`cannot encode message ${val.$typeName} to JSON: lowerCamelCase of path name "` +
                p +
                '" is irreversible');
        }
        return protoCamelCase(p);
    })
        .join(",");
}
function structToJson(val) {
    const json = {};
    for (const [k, v] of Object.entries(val.fields)) {
        json[k] = valueToJson(v);
    }
    return json;
}
function valueToJson(val) {
    switch (val.kind.case) {
        case "nullValue":
            return null;
        case "numberValue":
            if (!Number.isFinite(val.kind.value)) {
                throw new Error(`${val.$typeName} cannot be NaN or Infinity`);
            }
            return val.kind.value;
        case "boolValue":
            return val.kind.value;
        case "stringValue":
            return val.kind.value;
        case "structValue":
            return structToJson(val.kind.value);
        case "listValue":
            return listValueToJson(val.kind.value);
        default:
            throw new Error(`${val.$typeName} must have a value`);
    }
}
function listValueToJson(val) {
    return val.values.map(valueToJson);
}
function timestampToJson(val) {
    const ms = Number(val.seconds) * 1000;
    if (ms < Date.parse("0001-01-01T00:00:00Z") ||
        ms > Date.parse("9999-12-31T23:59:59Z")) {
        throw new Error(`cannot encode message ${val.$typeName} to JSON: must be from 0001-01-01T00:00:00Z to 9999-12-31T23:59:59Z inclusive`);
    }
    if (val.nanos < 0) {
        throw new Error(`cannot encode message ${val.$typeName} to JSON: nanos must not be negative`);
    }
    let z = "Z";
    if (val.nanos > 0) {
        const nanosStr = (val.nanos + 1000000000).toString().substring(1);
        if (nanosStr.substring(3) === "000000") {
            z = "." + nanosStr.substring(0, 3) + "Z";
        }
        else if (nanosStr.substring(6) === "000") {
            z = "." + nanosStr.substring(0, 6) + "Z";
        }
        else {
            z = "." + nanosStr + "Z";
        }
    }
    return new Date(ms).toISOString().replace(".000Z", z);
}

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// Default options for parsing JSON.
const jsonReadDefaults = {
    ignoreUnknownFields: false,
};
function makeReadOptions(options) {
    return options ? Object.assign(Object.assign({}, jsonReadDefaults), options) : jsonReadDefaults;
}
/**
 * Parse a message from a JSON string.
 */
function fromJsonString(schema, json, options) {
    return fromJson(schema, parseJsonString(json, schema.typeName), options);
}
/**
 * Parse a message from a JSON value.
 */
function fromJson(schema, json, options) {
    const msg = reflect(schema);
    try {
        readMessage(msg, json, makeReadOptions(options));
    }
    catch (e) {
        if (isFieldError(e)) {
            // @ts-expect-error we use the ES2022 error CTOR option "cause" for better stack traces
            throw new Error(`cannot decode ${e.field()} from JSON: ${e.message}`, {
                cause: e,
            });
        }
        throw e;
    }
    return msg.message;
}
function readMessage(msg, json, opts) {
    var _a;
    if (tryWktFromJson(msg, json, opts)) {
        return;
    }
    if (json == null || Array.isArray(json) || typeof json != "object") {
        throw new Error(`cannot decode ${msg.desc} from JSON: ${formatVal(json)}`);
    }
    const oneofSeen = new Map();
    const jsonNames = new Map();
    for (const field of msg.desc.fields) {
        jsonNames.set(field.name, field).set(field.jsonName, field);
    }
    for (const [jsonKey, jsonValue] of Object.entries(json)) {
        const field = jsonNames.get(jsonKey);
        if (field) {
            if (field.oneof) {
                if (jsonValue === null && field.fieldKind == "scalar") {
                    // see conformance test Required.Proto3.JsonInput.OneofFieldNull{First,Second}
                    continue;
                }
                const seen = oneofSeen.get(field.oneof);
                if (seen !== undefined) {
                    throw new FieldError(field.oneof, `oneof set multiple times by ${seen.name} and ${field.name}`);
                }
                oneofSeen.set(field.oneof, field);
            }
            readField(msg, field, jsonValue, opts);
        }
        else {
            let extension = undefined;
            if (jsonKey.startsWith("[") &&
                jsonKey.endsWith("]") &&
                // biome-ignore lint/suspicious/noAssignInExpressions: no
                (extension = (_a = opts.registry) === null || _a === void 0 ? void 0 : _a.getExtension(jsonKey.substring(1, jsonKey.length - 1))) &&
                extension.extendee.typeName === msg.desc.typeName) {
                const [container, field, get] = createExtensionContainer(extension);
                readField(container, field, jsonValue, opts);
                setExtension(msg.message, extension, get());
            }
            if (!extension && !opts.ignoreUnknownFields) {
                throw new Error(`cannot decode ${msg.desc} from JSON: key "${jsonKey}" is unknown`);
            }
        }
    }
}
function readField(msg, field, json, opts) {
    switch (field.fieldKind) {
        case "scalar":
            readScalarField(msg, field, json);
            break;
        case "enum":
            readEnumField(msg, field, json, opts);
            break;
        case "message":
            readMessageField(msg, field, json, opts);
            break;
        case "list":
            readListField(msg.get(field), json, opts);
            break;
        case "map":
            readMapField(msg.get(field), json, opts);
            break;
    }
}
function readMapField(map, json, opts) {
    if (json === null) {
        return;
    }
    const field = map.field();
    if (typeof json != "object" || Array.isArray(json)) {
        throw new FieldError(field, "expected object, got " + formatVal(json));
    }
    for (const [jsonMapKey, jsonMapValue] of Object.entries(json)) {
        if (jsonMapValue === null) {
            throw new FieldError(field, "map value must not be null");
        }
        let value;
        switch (field.mapKind) {
            case "message":
                const msgValue = reflect(field.message);
                readMessage(msgValue, jsonMapValue, opts);
                value = msgValue;
                break;
            case "enum":
                value = readEnum(field.enum, jsonMapValue, opts.ignoreUnknownFields, true);
                if (value === tokenIgnoredUnknownEnum) {
                    return;
                }
                break;
            case "scalar":
                value = scalarFromJson(field, jsonMapValue, true);
                break;
        }
        const key = mapKeyFromJson(field.mapKey, jsonMapKey);
        map.set(key, value);
    }
}
function readListField(list, json, opts) {
    if (json === null) {
        return;
    }
    const field = list.field();
    if (!Array.isArray(json)) {
        throw new FieldError(field, "expected Array, got " + formatVal(json));
    }
    for (const jsonItem of json) {
        if (jsonItem === null) {
            throw new FieldError(field, "list item must not be null");
        }
        switch (field.listKind) {
            case "message":
                const msgValue = reflect(field.message);
                readMessage(msgValue, jsonItem, opts);
                list.add(msgValue);
                break;
            case "enum":
                const enumValue = readEnum(field.enum, jsonItem, opts.ignoreUnknownFields, true);
                if (enumValue !== tokenIgnoredUnknownEnum) {
                    list.add(enumValue);
                }
                break;
            case "scalar":
                list.add(scalarFromJson(field, jsonItem, true));
                break;
        }
    }
}
function readMessageField(msg, field, json, opts) {
    if (json === null && field.message.typeName != "google.protobuf.Value") {
        msg.clear(field);
        return;
    }
    const msgValue = msg.isSet(field) ? msg.get(field) : reflect(field.message);
    readMessage(msgValue, json, opts);
    msg.set(field, msgValue);
}
function readEnumField(msg, field, json, opts) {
    const enumValue = readEnum(field.enum, json, opts.ignoreUnknownFields, false);
    if (enumValue === tokenNull) {
        msg.clear(field);
    }
    else if (enumValue !== tokenIgnoredUnknownEnum) {
        msg.set(field, enumValue);
    }
}
function readScalarField(msg, field, json) {
    const scalarValue = scalarFromJson(field, json, false);
    if (scalarValue === tokenNull) {
        msg.clear(field);
    }
    else {
        msg.set(field, scalarValue);
    }
}
const tokenIgnoredUnknownEnum = Symbol();
function readEnum(desc, json, ignoreUnknownFields, nullAsZeroValue) {
    if (json === null) {
        if (desc.typeName == "google.protobuf.NullValue") {
            return 0; // google.protobuf.NullValue.NULL_VALUE = 0
        }
        return nullAsZeroValue ? desc.values[0].number : tokenNull;
    }
    switch (typeof json) {
        case "number":
            if (Number.isInteger(json)) {
                return json;
            }
            break;
        case "string":
            const value = desc.values.find((ev) => ev.name === json);
            if (value !== undefined) {
                return value.number;
            }
            if (ignoreUnknownFields) {
                return tokenIgnoredUnknownEnum;
            }
            break;
    }
    throw new Error(`cannot decode ${desc} from JSON: ${formatVal(json)}`);
}
const tokenNull = Symbol();
function scalarFromJson(field, json, nullAsZeroValue) {
    if (json === null) {
        if (nullAsZeroValue) {
            return scalarZeroValue(field.scalar, false);
        }
        return tokenNull;
    }
    // int64, sfixed64, sint64, fixed64, uint64: Reflect supports string and number.
    // string, bool: Supported by reflect.
    switch (field.scalar) {
        // float, double: JSON value will be a number or one of the special string values "NaN", "Infinity", and "-Infinity".
        // Either numbers or strings are accepted. Exponent notation is also accepted.
        case ScalarType.DOUBLE:
        case ScalarType.FLOAT:
            if (json === "NaN")
                return NaN;
            if (json === "Infinity")
                return Number.POSITIVE_INFINITY;
            if (json === "-Infinity")
                return Number.NEGATIVE_INFINITY;
            if (typeof json == "number") {
                if (Number.isNaN(json)) {
                    // NaN must be encoded with string constants
                    throw new FieldError(field, "unexpected NaN number");
                }
                if (!Number.isFinite(json)) {
                    // Infinity must be encoded with string constants
                    throw new FieldError(field, "unexpected infinite number");
                }
                break;
            }
            if (typeof json == "string") {
                if (json === "") {
                    // empty string is not a number
                    break;
                }
                if (json.trim().length !== json.length) {
                    // extra whitespace
                    break;
                }
                const float = Number(json);
                if (!Number.isFinite(float)) {
                    // Infinity and NaN must be encoded with string constants
                    break;
                }
                return float;
            }
            break;
        // int32, fixed32, uint32: JSON value will be a decimal number. Either numbers or strings are accepted.
        case ScalarType.INT32:
        case ScalarType.FIXED32:
        case ScalarType.SFIXED32:
        case ScalarType.SINT32:
        case ScalarType.UINT32:
            return int32FromJson(json);
        // bytes: JSON value will be the data encoded as a string using standard base64 encoding with paddings.
        // Either standard or URL-safe base64 encoding with/without paddings are accepted.
        case ScalarType.BYTES:
            if (typeof json == "string") {
                if (json === "") {
                    return new Uint8Array(0);
                }
                try {
                    return base64Decode(json);
                }
                catch (e) {
                    const message = e instanceof Error ? e.message : String(e);
                    throw new FieldError(field, message);
                }
            }
            break;
    }
    return json;
}
/**
 * Try to parse a JSON value to a map key for the reflect API.
 *
 * Returns the input if the JSON value cannot be converted.
 */
function mapKeyFromJson(type, json) {
    switch (type) {
        case ScalarType.BOOL:
            switch (json) {
                case "true":
                    return true;
                case "false":
                    return false;
            }
            return json;
        case ScalarType.INT32:
        case ScalarType.FIXED32:
        case ScalarType.UINT32:
        case ScalarType.SFIXED32:
        case ScalarType.SINT32:
            return int32FromJson(json);
        default:
            return json;
    }
}
/**
 * Try to parse a JSON value to a 32-bit integer for the reflect API.
 *
 * Returns the input if the JSON value cannot be converted.
 */
function int32FromJson(json) {
    if (typeof json == "string") {
        if (json === "") {
            // empty string is not a number
            return json;
        }
        if (json.trim().length !== json.length) {
            // extra whitespace
            return json;
        }
        const num = Number(json);
        if (Number.isNaN(num)) {
            // not a number
            return json;
        }
        return num;
    }
    return json;
}
function parseJsonString(jsonString, typeName) {
    try {
        return JSON.parse(jsonString);
    }
    catch (e) {
        const message = e instanceof Error ? e.message : String(e);
        throw new Error(`cannot decode message ${typeName} from JSON: ${message}`, 
        // @ts-expect-error we use the ES2022 error CTOR option "cause" for better stack traces
        { cause: e });
    }
}
function tryWktFromJson(msg, jsonValue, opts) {
    if (!msg.desc.typeName.startsWith("google.protobuf.")) {
        return false;
    }
    switch (msg.desc.typeName) {
        case "google.protobuf.Any":
            anyFromJson(msg.message, jsonValue, opts);
            return true;
        case "google.protobuf.Timestamp":
            timestampFromJson(msg.message, jsonValue);
            return true;
        case "google.protobuf.Duration":
            durationFromJson(msg.message, jsonValue);
            return true;
        case "google.protobuf.FieldMask":
            fieldMaskFromJson(msg.message, jsonValue);
            return true;
        case "google.protobuf.Struct":
            structFromJson(msg.message, jsonValue);
            return true;
        case "google.protobuf.Value":
            valueFromJson(msg.message, jsonValue);
            return true;
        case "google.protobuf.ListValue":
            listValueFromJson(msg.message, jsonValue);
            return true;
        default:
            if (isWrapperDesc(msg.desc)) {
                const valueField = msg.desc.fields[0];
                if (jsonValue === null) {
                    msg.clear(valueField);
                }
                else {
                    msg.set(valueField, scalarFromJson(valueField, jsonValue, true));
                }
                return true;
            }
            return false;
    }
}
function anyFromJson(any, json, opts) {
    var _a;
    if (json === null || Array.isArray(json) || typeof json != "object") {
        throw new Error(`cannot decode message ${any.$typeName} from JSON: expected object but got ${formatVal(json)}`);
    }
    if (Object.keys(json).length == 0) {
        return;
    }
    const typeUrl = json["@type"];
    if (typeof typeUrl != "string" || typeUrl == "") {
        throw new Error(`cannot decode message ${any.$typeName} from JSON: "@type" is empty`);
    }
    const typeName = typeUrl.includes("/")
        ? typeUrl.substring(typeUrl.lastIndexOf("/") + 1)
        : typeUrl;
    if (!typeName.length) {
        throw new Error(`cannot decode message ${any.$typeName} from JSON: "@type" is invalid`);
    }
    const desc = (_a = opts.registry) === null || _a === void 0 ? void 0 : _a.getMessage(typeName);
    if (!desc) {
        throw new Error(`cannot decode message ${any.$typeName} from JSON: ${typeUrl} is not in the type registry`);
    }
    const msg = reflect(desc);
    if (typeName.startsWith("google.protobuf.") &&
        Object.prototype.hasOwnProperty.call(json, "value")) {
        const value = json.value;
        readMessage(msg, value, opts);
    }
    else {
        const copy = Object.assign({}, json);
        // biome-ignore lint/performance/noDelete: <explanation>
        delete copy["@type"];
        readMessage(msg, copy, opts);
    }
    anyPack(msg.desc, msg.message, any);
}
function timestampFromJson(timestamp, json) {
    if (typeof json !== "string") {
        throw new Error(`cannot decode message ${timestamp.$typeName} from JSON: ${formatVal(json)}`);
    }
    const matches = json.match(/^([0-9]{4})-([0-9]{2})-([0-9]{2})T([0-9]{2}):([0-9]{2}):([0-9]{2})(?:\.([0-9]{1,9}))?(?:Z|([+-][0-9][0-9]:[0-9][0-9]))$/);
    if (!matches) {
        throw new Error(`cannot decode message ${timestamp.$typeName} from JSON: invalid RFC 3339 string`);
    }
    const ms = Date.parse(
    // biome-ignore format: want this to read well
    matches[1] + "-" + matches[2] + "-" + matches[3] + "T" + matches[4] + ":" + matches[5] + ":" + matches[6] + (matches[8] ? matches[8] : "Z"));
    if (Number.isNaN(ms)) {
        throw new Error(`cannot decode message ${timestamp.$typeName} from JSON: invalid RFC 3339 string`);
    }
    if (ms < Date.parse("0001-01-01T00:00:00Z") ||
        ms > Date.parse("9999-12-31T23:59:59Z")) {
        throw new Error(`cannot decode message ${timestamp.$typeName} from JSON: must be from 0001-01-01T00:00:00Z to 9999-12-31T23:59:59Z inclusive`);
    }
    timestamp.seconds = protoInt64.parse(ms / 1000);
    timestamp.nanos = 0;
    if (matches[7]) {
        timestamp.nanos =
            parseInt("1" + matches[7] + "0".repeat(9 - matches[7].length)) -
                1000000000;
    }
}
function durationFromJson(duration, json) {
    if (typeof json !== "string") {
        throw new Error(`cannot decode message ${duration.$typeName} from JSON: ${formatVal(json)}`);
    }
    const match = json.match(/^(-?[0-9]+)(?:\.([0-9]+))?s/);
    if (match === null) {
        throw new Error(`cannot decode message ${duration.$typeName} from JSON: ${formatVal(json)}`);
    }
    const longSeconds = Number(match[1]);
    if (longSeconds > 315576000000 || longSeconds < -315576e6) {
        throw new Error(`cannot decode message ${duration.$typeName} from JSON: ${formatVal(json)}`);
    }
    duration.seconds = protoInt64.parse(longSeconds);
    if (typeof match[2] !== "string") {
        return;
    }
    const nanosStr = match[2] + "0".repeat(9 - match[2].length);
    duration.nanos = parseInt(nanosStr);
    if (longSeconds < 0 || Object.is(longSeconds, -0)) {
        duration.nanos = -duration.nanos;
    }
}
function fieldMaskFromJson(fieldMask, json) {
    if (typeof json !== "string") {
        throw new Error(`cannot decode message ${fieldMask.$typeName} from JSON: ${formatVal(json)}`);
    }
    if (json === "") {
        return;
    }
    function camelToSnake(str) {
        if (str.includes("_")) {
            throw new Error(`cannot decode message ${fieldMask.$typeName} from JSON: path names must be lowerCamelCase`);
        }
        const sc = str.replace(/[A-Z]/g, (letter) => "_" + letter.toLowerCase());
        return sc[0] === "_" ? sc.substring(1) : sc;
    }
    fieldMask.paths = json.split(",").map(camelToSnake);
}
function structFromJson(struct, json) {
    if (typeof json != "object" || json == null || Array.isArray(json)) {
        throw new Error(`cannot decode message ${struct.$typeName} from JSON ${formatVal(json)}`);
    }
    for (const [k, v] of Object.entries(json)) {
        const parsedV = create(ValueSchema);
        valueFromJson(parsedV, v);
        struct.fields[k] = parsedV;
    }
}
function valueFromJson(value, json) {
    switch (typeof json) {
        case "number":
            value.kind = { case: "numberValue", value: json };
            break;
        case "string":
            value.kind = { case: "stringValue", value: json };
            break;
        case "boolean":
            value.kind = { case: "boolValue", value: json };
            break;
        case "object":
            if (json === null) {
                value.kind = { case: "nullValue", value: NullValue.NULL_VALUE };
            }
            else if (Array.isArray(json)) {
                const listValue = create(ListValueSchema);
                listValueFromJson(listValue, json);
                value.kind = { case: "listValue", value: listValue };
            }
            else {
                const struct = create(StructSchema);
                structFromJson(struct, json);
                value.kind = { case: "structValue", value: struct };
            }
            break;
        default:
            throw new Error(`cannot decode message ${value.$typeName} from JSON ${formatVal(json)}`);
    }
    return value;
}
function listValueFromJson(listValue, json) {
    if (!Array.isArray(json)) {
        throw new Error(`cannot decode message ${listValue.$typeName} from JSON ${formatVal(json)}`);
    }
    for (const e of json) {
        const value = create(ValueSchema);
        valueFromJson(value, e);
        listValue.values.push(value);
    }
}

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * codeToString returns the string representation of a Code.
 *
 * @private Internal code, does not follow semantic versioning.
 */
function codeToString(value) {
    const name = Code[value];
    if (typeof name != "string") {
        return value.toString();
    }
    return (name[0].toLowerCase() +
        name.substring(1).replace(/[A-Z]/g, (c) => "_" + c.toLowerCase()));
}

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * ConnectError captures four pieces of information: a Code, an error
 * message, an optional cause of the error, and an optional collection of
 * arbitrary Protobuf messages called  "details".
 *
 * Because developer tools typically show just the error message, we prefix
 * it with the status code, so that the most important information is always
 * visible immediately.
 *
 * Error details are wrapped with google.protobuf.Any on the wire, so that
 * a server or middleware can attach arbitrary data to an error. Use the
 * method findDetails() to retrieve the details.
 */
class ConnectError extends Error {
    /**
     * Create a new ConnectError.
     * If no code is provided, code "unknown" is used.
     * Outgoing details are only relevant for the server side - a service may
     * raise an error with details, and it is up to the protocol implementation
     * to encode and send the details along with error.
     */
    constructor(message, code = Code.Unknown, metadata, outgoingDetails, cause) {
        super(createMessage(message, code));
        this.name = "ConnectError";
        // see https://www.typescriptlang.org/docs/handbook/release-notes/typescript-2-2.html#example
        Object.setPrototypeOf(this, new.target.prototype);
        this.rawMessage = message;
        this.code = code;
        this.metadata = new Headers(metadata !== null && metadata !== void 0 ? metadata : {});
        this.details = outgoingDetails !== null && outgoingDetails !== void 0 ? outgoingDetails : [];
        this.cause = cause;
    }
    /**
     * Convert any value - typically a caught error into a ConnectError,
     * following these rules:
     * - If the value is already a ConnectError, return it as is.
     * - If the value is an AbortError or TimeoutError from the fetch API, return
     *   the message of the error with code Canceled.
     * - For other Errors, return the error message with code Unknown by default.
     * - For other values, return the values String representation as a message,
     *   with the code Unknown by default.
     * The original value will be used for the "cause" property for the new
     * ConnectError.
     */
    static from(reason, code = Code.Unknown) {
        if (reason instanceof ConnectError) {
            return reason;
        }
        if (reason instanceof Error) {
            if (reason.name == "AbortError" || reason.name == "TimeoutError") {
                // Fetch requests can only be canceled with an AbortController,
                // or with AbortSignal.timeout().
                return new ConnectError(reason.message, Code.Canceled);
            }
            return new ConnectError(reason.message, code, undefined, undefined, reason);
        }
        return new ConnectError(String(reason), code, undefined, undefined, reason);
    }
    static [Symbol.hasInstance](v) {
        if (!(v instanceof Error)) {
            return false;
        }
        if (Object.getPrototypeOf(v) === ConnectError.prototype) {
            return true;
        }
        return (v.name === "ConnectError" &&
            "code" in v &&
            typeof v.code === "number" &&
            "metadata" in v &&
            "details" in v &&
            Array.isArray(v.details) &&
            "rawMessage" in v &&
            typeof v.rawMessage == "string" &&
            "cause" in v);
    }
    findDetails(typeOrRegistry) {
        const registry = typeOrRegistry.kind === "message"
            ? {
                getMessage: (typeName) => typeName === typeOrRegistry.typeName ? typeOrRegistry : undefined,
            }
            : typeOrRegistry;
        const details = [];
        for (const data of this.details) {
            if ("desc" in data) {
                if (registry.getMessage(data.desc.typeName)) {
                    details.push(create(data.desc, data.value));
                }
                continue;
            }
            const desc = registry.getMessage(data.type);
            if (desc) {
                try {
                    details.push(fromBinary(desc, data.value));
                }
                catch (_) {
                    // We silently give up if we are unable to parse the detail, because
                    // that appears to be the least worst behavior.
                    // It is very unlikely that a user surrounds a catch body handling the
                    // error with another try-catch statement, and we do not want to
                    // recommend doing so.
                }
            }
        }
        return details;
    }
}
/**
 * Create an error message, prefixing the given code.
 */
function createMessage(message, code) {
    return message.length
        ? `[${codeToString(code)}] ${message}`
        : `[${codeToString(code)}]`;
}

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
function decodeBinaryHeader(value, desc, options) {
    try {
        const bytes = base64Decode(value);
        if (desc) {
            return fromBinary(desc, bytes, options);
        }
        return bytes;
    }
    catch (e) {
        throw ConnectError.from(e, Code.DataLoss);
    }
}

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Create any client for the given service.
 *
 * The given createMethod function is called for each method definition
 * of the service. The function it returns is added to the client object
 * as a method.
 */
function makeAnyClient(service, createMethod) {
    const client = {};
    for (const desc of service.methods) {
        const method = createMethod(desc);
        if (method != null) {
            client[desc.localName] = method;
        }
    }
    return client;
}

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * compressedFlag indicates that the data in a EnvelopedMessage is
 * compressed. It has the same meaning in the gRPC-Web, gRPC-HTTP2,
 * and Connect protocols.
 *
 * @private Internal code, does not follow semantic versioning.
 */
const compressedFlag = 0b00000001;

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * At most, allow ~4GiB to be received or sent per message.
 * zlib used by Node.js caps maxOutputLength at this value. It also happens to
 * be the maximum theoretical message size supported by protobuf-es.
 */
const maxReadMaxBytes = 0xffffffff;
const maxWriteMaxBytes = maxReadMaxBytes;
/**
 * The default value for the compressMinBytes option. The CPU cost of compressing
 * very small messages usually isn't worth the small reduction in network I/O, so
 * the default value is 1 kibibyte.
 */
const defaultCompressMinBytes = 1024;
/**
 * Asserts that the options writeMaxBytes, readMaxBytes, and compressMinBytes
 * are within sane limits, and returns default values where no value is
 * provided.
 *
 * @private Internal code, does not follow semantic versioning.
 */
function validateReadWriteMaxBytes(readMaxBytes, writeMaxBytes, compressMinBytes) {
    writeMaxBytes !== null && writeMaxBytes !== void 0 ? writeMaxBytes : (writeMaxBytes = maxWriteMaxBytes);
    readMaxBytes !== null && readMaxBytes !== void 0 ? readMaxBytes : (readMaxBytes = maxReadMaxBytes);
    compressMinBytes !== null && compressMinBytes !== void 0 ? compressMinBytes : (compressMinBytes = defaultCompressMinBytes);
    if (writeMaxBytes < 1 || writeMaxBytes > maxWriteMaxBytes) {
        throw new ConnectError(`writeMaxBytes ${writeMaxBytes} must be >= 1 and <= ${maxWriteMaxBytes}`, Code.Internal);
    }
    if (readMaxBytes < 1 || readMaxBytes > maxReadMaxBytes) {
        throw new ConnectError(`readMaxBytes ${readMaxBytes} must be >= 1 and <= ${maxReadMaxBytes}`, Code.Internal);
    }
    return {
        readMaxBytes,
        writeMaxBytes,
        compressMinBytes,
    };
}
/**
 * Raise an error ResourceExhausted if more than writeMaxByte are written.
 *
 * @private Internal code, does not follow semantic versioning.
 */
function assertWriteMaxBytes(writeMaxBytes, bytesWritten) {
    if (bytesWritten > writeMaxBytes) {
        throw new ConnectError(`message size ${bytesWritten} is larger than configured writeMaxBytes ${writeMaxBytes}`, Code.ResourceExhausted);
    }
}
/**
 * Raise an error ResourceExhausted if more than readMaxBytes are read.
 *
 * @private Internal code, does not follow semantic versioning.
 */
function assertReadMaxBytes(readMaxBytes, bytesRead, totalSizeKnown = false) {
    if (bytesRead > readMaxBytes) {
        let message = `message size is larger than configured readMaxBytes ${readMaxBytes}`;
        if (totalSizeKnown) {
            message = `message size ${bytesRead} is larger than configured readMaxBytes ${readMaxBytes}`;
        }
        throw new ConnectError(message, Code.ResourceExhausted);
    }
}

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Create an EnvelopeDecoder. The `readMaxBytes` argument limits the maximum
 * size for individual messages.
 *
 * @private Internal code, does not follow semantic versioning.
 */
function createEnvelopeDecoder(readMaxBytes) {
    return new EnvelopeDecoderImpl(readMaxBytes);
}
class EnvelopeDecoderImpl {
    constructor(readMaxBytes) {
        this.readMaxBytes = readMaxBytes;
        // Envelope headers are 5 bytes: 1 byte for flags, 4 bytes message length
        this.header = new Uint8Array(5);
        this.headerView = new DataView(this.header.buffer);
        this.buf = [];
    }
    get byteLength() {
        return this.buf.reduce((a, b) => a + b.byteLength, 0);
    }
    decode(chunk) {
        this.buf.push(chunk);
        const envs = [];
        for (;;) {
            let env = this.pop();
            if (!env) {
                break;
            }
            envs.push(env);
        }
        return envs;
    }
    // consume an enveloped message
    pop() {
        if (!this.env) {
            this.env = this.head();
            if (!this.env) {
                return undefined;
            }
        }
        if (this.cons(this.env.data)) {
            const env = this.env;
            this.env = undefined;
            return env;
        }
        return undefined;
    }
    // consume header
    head() {
        if (!this.cons(this.header)) {
            return undefined;
        }
        const flags = this.headerView.getUint8(0); // first byte is flags
        const length = this.headerView.getUint32(1); // 4 bytes message length
        assertReadMaxBytes(this.readMaxBytes, length, true);
        return {
            flags,
            data: new Uint8Array(length),
        };
    }
    // consume from buffer, fill target
    cons(target) {
        const wantLength = target.byteLength;
        if (this.byteLength < wantLength) {
            return false;
        }
        let offset = 0;
        while (offset < wantLength) {
            const chunk = this.buf.shift(); // we check length above
            if (chunk.byteLength > wantLength - offset) {
                target.set(chunk.subarray(0, wantLength - offset), offset);
                this.buf.unshift(chunk.subarray(wantLength - offset));
                offset += wantLength - offset;
            }
            else {
                target.set(chunk, offset);
                offset += chunk.byteLength;
            }
        }
        return true;
    }
}
/**
 * Compress an EnvelopedMessage.
 *
 * Raises Internal if an enveloped message is already compressed.
 *
 * @private Internal code, does not follow semantic versioning.
 */
async function envelopeCompress(envelope, compression, compressMinBytes) {
    let { flags, data } = envelope;
    if ((flags & compressedFlag) === compressedFlag) {
        throw new ConnectError("invalid envelope, already compressed", Code.Internal);
    }
    if (compression && data.byteLength >= compressMinBytes) {
        data = await compression.compress(data);
        flags = flags | compressedFlag;
    }
    return { data, flags };
}
/**
 * Decompress an EnvelopedMessage.
 *
 * Raises InvalidArgument if an envelope is compressed, but compression is null.
 *
 * Relies on the provided Compression to raise ResourceExhausted if the
 * *decompressed* message size is larger than readMaxBytes. If the envelope is
 * not compressed, readMaxBytes is not honored.
 *
 * @private Internal code, does not follow semantic versioning.
 */
async function envelopeDecompress(envelope, compression, readMaxBytes) {
    let { flags, data } = envelope;
    if ((flags & compressedFlag) === compressedFlag) {
        if (!compression) {
            throw new ConnectError("received compressed envelope, but do not know how to decompress", Code.Internal);
        }
        data = await compression.decompress(data, readMaxBytes);
        flags = flags ^ compressedFlag;
    }
    return { data, flags };
}
/**
 * Encode a single enveloped message.
 *
 * @private Internal code, does not follow semantic versioning.
 */
function encodeEnvelope(flags, data) {
    const bytes = new Uint8Array(data.length + 5);
    bytes.set(data, 5);
    const v = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength);
    v.setUint8(0, flags); // first byte is flags
    v.setUint32(1, data.length); // 4 bytes message length
    return bytes;
}

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
var __asyncValues$2 = (undefined && undefined.__asyncValues) || function (o) {
    if (!Symbol.asyncIterator) throw new TypeError("Symbol.asyncIterator is not defined.");
    var m = o[Symbol.asyncIterator], i;
    return m ? m.call(o) : (o = typeof __values === "function" ? __values(o) : o[Symbol.iterator](), i = {}, verb("next"), verb("throw"), verb("return"), i[Symbol.asyncIterator] = function () { return this; }, i);
    function verb(n) { i[n] = o[n] && function (v) { return new Promise(function (resolve, reject) { v = o[n](v), settle(resolve, reject, v.done, v.value); }); }; }
    function settle(resolve, reject, d, v) { Promise.resolve(v).then(function(v) { resolve({ value: v, done: d }); }, reject); }
};
var __await$2 = (undefined && undefined.__await) || function (v) { return this instanceof __await$2 ? (this.v = v, this) : new __await$2(v); };
var __asyncGenerator$2 = (undefined && undefined.__asyncGenerator) || function (thisArg, _arguments, generator) {
    if (!Symbol.asyncIterator) throw new TypeError("Symbol.asyncIterator is not defined.");
    var g = generator.apply(thisArg, _arguments || []), i, q = [];
    return i = Object.create((typeof AsyncIterator === "function" ? AsyncIterator : Object).prototype), verb("next"), verb("throw"), verb("return", awaitReturn), i[Symbol.asyncIterator] = function () { return this; }, i;
    function awaitReturn(f) { return function (v) { return Promise.resolve(v).then(f, reject); }; }
    function verb(n, f) { if (g[n]) { i[n] = function (v) { return new Promise(function (a, b) { q.push([n, v, a, b]) > 1 || resume(n, v); }); }; if (f) i[n] = f(i[n]); } }
    function resume(n, v) { try { step(g[n](v)); } catch (e) { settle(q[0][3], e); } }
    function step(r) { r.value instanceof __await$2 ? Promise.resolve(r.value.v).then(fulfill, reject) : settle(q[0][2], r); }
    function fulfill(value) { resume("next", value); }
    function reject(value) { resume("throw", value); }
    function settle(f, v) { if (f(v), q.shift(), q.length) resume(q[0][0], q[0][1]); }
};
var __asyncDelegator$2 = (undefined && undefined.__asyncDelegator) || function (o) {
    var i, p;
    return i = {}, verb("next"), verb("throw", function (e) { throw e; }), verb("return"), i[Symbol.iterator] = function () { return this; }, i;
    function verb(n, f) { i[n] = o[n] ? function (v) { return (p = !p) ? { value: __await$2(o[n](v)), done: false } : f ? f(v) : v; } : f; }
};
function pipeTo(source, ...rest) {
    const [transforms, sink, opt] = pickTransformsAndSink(rest);
    let iterable = source;
    let abortable;
    if ((opt === null || opt === void 0 ? void 0 : opt.propagateDownStreamError) === true) {
        iterable = abortable = makeIterableAbortable(iterable);
    }
    // @ts-ignore
    iterable = pipe(iterable, ...transforms, { propagateDownStreamError: false });
    return sink(iterable).catch((reason) => {
        if (abortable) {
            return abortable.abort(reason).then(() => Promise.reject(reason));
        }
        return Promise.reject(reason);
    });
}
// pick transforms, the sink, and options from the pipeTo() rest parameter
function pickTransformsAndSink(rest) {
    let opt;
    if (typeof rest[rest.length - 1] != "function") {
        opt = rest.pop();
    }
    const sink = rest.pop();
    return [rest, sink, opt];
}
function pipe(source, ...rest) {
    return __asyncGenerator$2(this, arguments, function* pipe_1() {
        var _a;
        const [transforms, opt] = pickTransforms(rest);
        let abortable;
        const sourceIt = source[Symbol.asyncIterator]();
        const cachedSource = {
            [Symbol.asyncIterator]() {
                return sourceIt;
            },
        };
        let iterable = cachedSource;
        if ((opt === null || opt === void 0 ? void 0 : opt.propagateDownStreamError) === true) {
            iterable = abortable = makeIterableAbortable(iterable);
        }
        for (const t of transforms) {
            iterable = t(iterable);
        }
        const it = iterable[Symbol.asyncIterator]();
        try {
            for (;;) {
                const r = yield __await$2(it.next());
                if (r.done === true) {
                    break;
                }
                if (!abortable) {
                    yield yield __await$2(r.value);
                    continue;
                }
                try {
                    yield yield __await$2(r.value);
                }
                catch (e) {
                    yield __await$2(abortable.abort(e)); // propagate downstream error to the source
                    throw e;
                }
            }
        }
        finally {
            if ((opt === null || opt === void 0 ? void 0 : opt.propagateDownStreamError) === true) {
                // Call return on the source iterable to indicate
                // that we will no longer consume it and it should
                // cleanup any allocated resources.
                (_a = sourceIt.return) === null || _a === void 0 ? void 0 : _a.call(sourceIt).catch(() => {
                    // return returns a promise, which we don't care about.
                    //
                    // Uncaught promises are thrown at sometime/somewhere by the event loop,
                    // this is to ensure error is caught and ignored.
                });
            }
        }
    });
}
function pickTransforms(rest) {
    let opt;
    if (typeof rest[rest.length - 1] != "function") {
        opt = rest.pop();
    }
    return [rest, opt];
}
function transformSerializeEnvelope(serialization, endStreamFlag, endSerialization) {
    {
        return function (iterable) {
            return __asyncGenerator$2(this, arguments, function* () {
                var _a, e_4, _b, _c;
                try {
                    for (var _d = true, iterable_4 = __asyncValues$2(iterable), iterable_4_1; iterable_4_1 = yield __await$2(iterable_4.next()), _a = iterable_4_1.done, !_a; _d = true) {
                        _c = iterable_4_1.value;
                        _d = false;
                        const chunk = _c;
                        const data = serialization.serialize(chunk);
                        yield yield __await$2({ flags: 0, data });
                    }
                }
                catch (e_4_1) { e_4 = { error: e_4_1 }; }
                finally {
                    try {
                        if (!_d && !_a && (_b = iterable_4.return)) yield __await$2(_b.call(iterable_4));
                    }
                    finally { if (e_4) throw e_4.error; }
                }
            });
        };
    }
}
function transformParseEnvelope(serialization, endStreamFlag, endSerialization) {
    // code path always yields T
    return function (iterable) {
        return __asyncGenerator$2(this, arguments, function* () {
            var _a, e_7, _b, _c;
            try {
                for (var _d = true, iterable_7 = __asyncValues$2(iterable), iterable_7_1; iterable_7_1 = yield __await$2(iterable_7.next()), _a = iterable_7_1.done, !_a; _d = true) {
                    _c = iterable_7_1.value;
                    _d = false;
                    const { flags, data } = _c;
                    if (endStreamFlag !== undefined &&
                        (flags & endStreamFlag) === endStreamFlag) ;
                    yield yield __await$2(serialization.parse(data));
                }
            }
            catch (e_7_1) { e_7 = { error: e_7_1 }; }
            finally {
                try {
                    if (!_d && !_a && (_b = iterable_7.return)) yield __await$2(_b.call(iterable_7));
                }
                finally { if (e_7) throw e_7.error; }
            }
        });
    };
}
/**
 * Creates an AsyncIterableTransform that takes enveloped messages as a source,
 * and compresses them if they are larger than compressMinBytes.
 *
 * @private Internal code, does not follow semantic versioning.
 */
function transformCompressEnvelope(compression, compressMinBytes) {
    return function (iterable) {
        return __asyncGenerator$2(this, arguments, function* () {
            var _a, e_8, _b, _c;
            try {
                for (var _d = true, iterable_8 = __asyncValues$2(iterable), iterable_8_1; iterable_8_1 = yield __await$2(iterable_8.next()), _a = iterable_8_1.done, !_a; _d = true) {
                    _c = iterable_8_1.value;
                    _d = false;
                    const env = _c;
                    yield yield __await$2(yield __await$2(envelopeCompress(env, compression, compressMinBytes)));
                }
            }
            catch (e_8_1) { e_8 = { error: e_8_1 }; }
            finally {
                try {
                    if (!_d && !_a && (_b = iterable_8.return)) yield __await$2(_b.call(iterable_8));
                }
                finally { if (e_8) throw e_8.error; }
            }
        });
    };
}
/**
 * Creates an AsyncIterableTransform that takes enveloped messages as a source,
 * and decompresses them using the given compression.
 *
 * The iterable raises an error if the decompressed payload of an enveloped
 * message is larger than readMaxBytes, or if no compression is provided.
 *
 * @private Internal code, does not follow semantic versioning.
 */
function transformDecompressEnvelope(compression, readMaxBytes) {
    return function (iterable) {
        return __asyncGenerator$2(this, arguments, function* () {
            var _a, e_9, _b, _c;
            try {
                for (var _d = true, iterable_9 = __asyncValues$2(iterable), iterable_9_1; iterable_9_1 = yield __await$2(iterable_9.next()), _a = iterable_9_1.done, !_a; _d = true) {
                    _c = iterable_9_1.value;
                    _d = false;
                    const env = _c;
                    yield yield __await$2(yield __await$2(envelopeDecompress(env, compression, readMaxBytes)));
                }
            }
            catch (e_9_1) { e_9 = { error: e_9_1 }; }
            finally {
                try {
                    if (!_d && !_a && (_b = iterable_9.return)) yield __await$2(_b.call(iterable_9));
                }
                finally { if (e_9) throw e_9.error; }
            }
        });
    };
}
/**
 * Create an AsyncIterableTransform that takes enveloped messages as a source,
 * and joins them into a stream of raw bytes.
 *
 * @private Internal code, does not follow semantic versioning.
 */
function transformJoinEnvelopes() {
    return function (iterable) {
        return __asyncGenerator$2(this, arguments, function* () {
            var _a, e_10, _b, _c;
            try {
                for (var _d = true, iterable_10 = __asyncValues$2(iterable), iterable_10_1; iterable_10_1 = yield __await$2(iterable_10.next()), _a = iterable_10_1.done, !_a; _d = true) {
                    _c = iterable_10_1.value;
                    _d = false;
                    const { flags, data } = _c;
                    yield yield __await$2(encodeEnvelope(flags, data));
                }
            }
            catch (e_10_1) { e_10 = { error: e_10_1 }; }
            finally {
                try {
                    if (!_d && !_a && (_b = iterable_10.return)) yield __await$2(_b.call(iterable_10));
                }
                finally { if (e_10) throw e_10.error; }
            }
        });
    };
}
/**
 * Create an AsyncIterableTransform that takes raw bytes as a source, and splits
 * them into enveloped messages.
 *
 * The iterable raises an error
 * - if the payload of an enveloped message is larger than readMaxBytes,
 * - if the stream ended before an enveloped message fully arrived,
 * - or if the stream ended with extraneous data.
 *
 * @private Internal code, does not follow semantic versioning.
 */
function transformSplitEnvelope(readMaxBytes) {
    return function (iterable) {
        return __asyncGenerator$2(this, arguments, function* () {
            var _a, e_11, _b, _c;
            const buffer = createEnvelopeDecoder(readMaxBytes);
            try {
                for (var _d = true, iterable_11 = __asyncValues$2(iterable), iterable_11_1; iterable_11_1 = yield __await$2(iterable_11.next()), _a = iterable_11_1.done, !_a; _d = true) {
                    _c = iterable_11_1.value;
                    _d = false;
                    const chunk = _c;
                    for (const env of buffer.decode(chunk)) {
                        yield yield __await$2(env);
                    }
                }
            }
            catch (e_11_1) { e_11 = { error: e_11_1 }; }
            finally {
                try {
                    if (!_d && !_a && (_b = iterable_11.return)) yield __await$2(_b.call(iterable_11));
                }
                finally { if (e_11) throw e_11.error; }
            }
            if (buffer.byteLength > 0) {
                throw new ConnectError("protocol error: incomplete envelope", Code.InvalidArgument);
            }
        });
    };
}
/**
 * Wrap the given iterable and return an iterable with an abort() method.
 *
 * This function exists purely for convenience. Where one would typically have
 * to access the iterator directly, advance through all elements, and call
 * AsyncIterator.throw() to notify the upstream iterable, this function allows
 * to use convenient for-await loops and still notify the upstream iterable:
 *
 * ```ts
 * const abortable = makeIterableAbortable(iterable);
 * for await (const ele of abortable) {
 *   await abortable.abort("ERR");
 * }
 * ```
 * There are a couple of limitations of this function:
 * - the given async iterable must implement throw
 * - the async iterable cannot be re-use
 * - if source catches errors and yields values for them, they are ignored, and
 *   the source may still dangle
 *
 * There are four possible ways an async function* can handle yield errors:
 * 1. don't catch errors at all - Abortable.abort() will resolve "rethrown"
 * 2. catch errors and rethrow - Abortable.abort() will resolve "rethrown"
 * 3. catch errors and return - Abortable.abort() will resolve "completed"
 * 4. catch errors and yield a value - Abortable.abort() will resolve "caught"
 *
 * Note that catching errors and yielding a value is problematic, and it should
 * be documented that this may leave the source in a dangling state.
 *
 * @private Internal code, does not follow semantic versioning.
 */
function makeIterableAbortable(iterable) {
    const innerCandidate = iterable[Symbol.asyncIterator]();
    if (innerCandidate.throw === undefined) {
        throw new Error("AsyncIterable does not implement throw");
    }
    const inner = innerCandidate;
    let aborted;
    let resultPromise;
    let it = {
        next() {
            resultPromise = inner.next().finally(() => {
                resultPromise = undefined;
            });
            return resultPromise;
        },
        throw(e) {
            return inner.throw(e);
        },
    };
    if (innerCandidate.return !== undefined) {
        it = Object.assign(Object.assign({}, it), { return(value) {
                return inner.return(value);
            } });
    }
    let used = false;
    return {
        abort(reason) {
            if (aborted) {
                return aborted.state;
            }
            const f = () => {
                return inner.throw(reason).then((r) => (r.done === true ? "completed" : "caught"), () => "rethrown");
            };
            if (resultPromise) {
                aborted = { reason, state: resultPromise.then(f, f) };
                return aborted.state;
            }
            aborted = { reason, state: f() };
            return aborted.state;
        },
        [Symbol.asyncIterator]() {
            if (used) {
                throw new Error("AsyncIterable cannot be re-used");
            }
            used = true;
            return it;
        },
    };
}
/**
 * Create an asynchronous iterable from an array.
 *
 * @private Internal code, does not follow semantic versioning.
 */
function createAsyncIterable(items) {
    return __asyncGenerator$2(this, arguments, function* createAsyncIterable_1() {
        yield __await$2(yield* __asyncDelegator$2(__asyncValues$2(items)));
    });
}

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
var __asyncValues$1 = (undefined && undefined.__asyncValues) || function (o) {
    if (!Symbol.asyncIterator) throw new TypeError("Symbol.asyncIterator is not defined.");
    var m = o[Symbol.asyncIterator], i;
    return m ? m.call(o) : (o = typeof __values === "function" ? __values(o) : o[Symbol.iterator](), i = {}, verb("next"), verb("throw"), verb("return"), i[Symbol.asyncIterator] = function () { return this; }, i);
    function verb(n) { i[n] = o[n] && function (v) { return new Promise(function (resolve, reject) { v = o[n](v), settle(resolve, reject, v.done, v.value); }); }; }
    function settle(resolve, reject, d, v) { Promise.resolve(v).then(function(v) { resolve({ value: v, done: d }); }, reject); }
};
var __await$1 = (undefined && undefined.__await) || function (v) { return this instanceof __await$1 ? (this.v = v, this) : new __await$1(v); };
var __asyncDelegator$1 = (undefined && undefined.__asyncDelegator) || function (o) {
    var i, p;
    return i = {}, verb("next"), verb("throw", function (e) { throw e; }), verb("return"), i[Symbol.iterator] = function () { return this; }, i;
    function verb(n, f) { i[n] = o[n] ? function (v) { return (p = !p) ? { value: __await$1(o[n](v)), done: false } : f ? f(v) : v; } : f; }
};
var __asyncGenerator$1 = (undefined && undefined.__asyncGenerator) || function (thisArg, _arguments, generator) {
    if (!Symbol.asyncIterator) throw new TypeError("Symbol.asyncIterator is not defined.");
    var g = generator.apply(thisArg, _arguments || []), i, q = [];
    return i = Object.create((typeof AsyncIterator === "function" ? AsyncIterator : Object).prototype), verb("next"), verb("throw"), verb("return", awaitReturn), i[Symbol.asyncIterator] = function () { return this; }, i;
    function awaitReturn(f) { return function (v) { return Promise.resolve(v).then(f, reject); }; }
    function verb(n, f) { if (g[n]) { i[n] = function (v) { return new Promise(function (a, b) { q.push([n, v, a, b]) > 1 || resume(n, v); }); }; if (f) i[n] = f(i[n]); } }
    function resume(n, v) { try { step(g[n](v)); } catch (e) { settle(q[0][3], e); } }
    function step(r) { r.value instanceof __await$1 ? Promise.resolve(r.value.v).then(fulfill, reject) : settle(q[0][2], r); }
    function fulfill(value) { resume("next", value); }
    function reject(value) { resume("throw", value); }
    function settle(f, v) { if (f(v), q.shift(), q.length) resume(q[0][0], q[0][1]); }
};
/**
 * Create a Client for the given service, invoking RPCs through the
 * given transport.
 */
function createClient(service, transport) {
    return makeAnyClient(service, (method) => {
        switch (method.methodKind) {
            case "unary":
                return createUnaryFn(transport, method);
            case "server_streaming":
                return createServerStreamingFn(transport, method);
            case "client_streaming":
                return createClientStreamingFn(transport, method);
            case "bidi_streaming":
                return createBiDiStreamingFn(transport, method);
            default:
                return null;
        }
    });
}
function createUnaryFn(transport, method) {
    return async (input, options) => {
        var _a, _b;
        const response = await transport.unary(method, options === null || options === void 0 ? void 0 : options.signal, options === null || options === void 0 ? void 0 : options.timeoutMs, options === null || options === void 0 ? void 0 : options.headers, input, options === null || options === void 0 ? void 0 : options.contextValues);
        (_a = options === null || options === void 0 ? void 0 : options.onHeader) === null || _a === void 0 ? void 0 : _a.call(options, response.header);
        (_b = options === null || options === void 0 ? void 0 : options.onTrailer) === null || _b === void 0 ? void 0 : _b.call(options, response.trailer);
        return response.message;
    };
}
function createServerStreamingFn(transport, method) {
    return (input, options) => handleStreamResponse(transport.stream(method, options === null || options === void 0 ? void 0 : options.signal, options === null || options === void 0 ? void 0 : options.timeoutMs, options === null || options === void 0 ? void 0 : options.headers, createAsyncIterable([input]), options === null || options === void 0 ? void 0 : options.contextValues), options);
}
function createClientStreamingFn(transport, method) {
    return async (request, options) => {
        var _a, e_1, _b, _c;
        var _d, _e;
        const response = await transport.stream(method, options === null || options === void 0 ? void 0 : options.signal, options === null || options === void 0 ? void 0 : options.timeoutMs, options === null || options === void 0 ? void 0 : options.headers, request, options === null || options === void 0 ? void 0 : options.contextValues);
        (_d = options === null || options === void 0 ? void 0 : options.onHeader) === null || _d === void 0 ? void 0 : _d.call(options, response.header);
        let singleMessage;
        let count = 0;
        try {
            for (var _f = true, _g = __asyncValues$1(response.message), _h; _h = await _g.next(), _a = _h.done, !_a; _f = true) {
                _c = _h.value;
                _f = false;
                const message = _c;
                singleMessage = message;
                count++;
            }
        }
        catch (e_1_1) { e_1 = { error: e_1_1 }; }
        finally {
            try {
                if (!_f && !_a && (_b = _g.return)) await _b.call(_g);
            }
            finally { if (e_1) throw e_1.error; }
        }
        if (!singleMessage) {
            throw new ConnectError("protocol error: missing response message", Code.Unimplemented);
        }
        if (count > 1) {
            throw new ConnectError("protocol error: received extra messages for client streaming method", Code.Unimplemented);
        }
        (_e = options === null || options === void 0 ? void 0 : options.onTrailer) === null || _e === void 0 ? void 0 : _e.call(options, response.trailer);
        return singleMessage;
    };
}
function createBiDiStreamingFn(transport, method) {
    return (request, options) => handleStreamResponse(transport.stream(method, options === null || options === void 0 ? void 0 : options.signal, options === null || options === void 0 ? void 0 : options.timeoutMs, options === null || options === void 0 ? void 0 : options.headers, request, options === null || options === void 0 ? void 0 : options.contextValues), options);
}
function handleStreamResponse(stream, options) {
    const it = (function () {
        return __asyncGenerator$1(this, arguments, function* () {
            var _a, _b;
            const response = yield __await$1(stream);
            (_a = options === null || options === void 0 ? void 0 : options.onHeader) === null || _a === void 0 ? void 0 : _a.call(options, response.header);
            yield __await$1(yield* __asyncDelegator$1(__asyncValues$1(response.message)));
            (_b = options === null || options === void 0 ? void 0 : options.onTrailer) === null || _b === void 0 ? void 0 : _b.call(options, response.trailer);
        });
    })()[Symbol.asyncIterator]();
    // Create a new iterable to omit throw/return.
    return {
        [Symbol.asyncIterator]: () => ({
            next: () => it.next(),
        }),
    };
}

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Create an AbortController that is automatically aborted if one of the given
 * signals is aborted.
 *
 * For convenience, the linked AbortSignals can be undefined.
 *
 * If the controller or any of the signals is aborted, all event listeners are
 * removed.
 *
 * @private Internal code, does not follow semantic versioning.
 */
function createLinkedAbortController(...signals) {
    const controller = new AbortController();
    const sa = signals.filter((s) => s !== undefined).concat(controller.signal);
    for (const signal of sa) {
        if (signal.aborted) {
            onAbort.apply(signal);
            break;
        }
        signal.addEventListener("abort", onAbort);
    }
    function onAbort() {
        if (!controller.signal.aborted) {
            controller.abort(getAbortSignalReason(this));
        }
        for (const signal of sa) {
            signal.removeEventListener("abort", onAbort);
        }
    }
    return controller;
}
/**
 * Create a deadline signal. The returned object contains an AbortSignal, but
 * also a cleanup function to stop the timer, which must be called once the
 * calling code is no longer interested in the signal.
 *
 * Ideally, we would simply use AbortSignal.timeout(), but it is not widely
 * available yet.
 *
 * @private Internal code, does not follow semantic versioning.
 */
function createDeadlineSignal(timeoutMs) {
    const controller = new AbortController();
    const listener = () => {
        controller.abort(new ConnectError("the operation timed out", Code.DeadlineExceeded));
    };
    let timeoutId;
    if (timeoutMs !== undefined) {
        if (timeoutMs <= 0)
            listener();
        else
            timeoutId = setTimeout(listener, timeoutMs);
    }
    return {
        signal: controller.signal,
        cleanup: () => clearTimeout(timeoutId),
    };
}
/**
 * Returns the reason why an AbortSignal was aborted. Returns undefined if the
 * signal has not been aborted.
 *
 * The property AbortSignal.reason is not widely available. This function
 * returns an AbortError if the signal is aborted, but reason is undefined.
 *
 * @private Internal code, does not follow semantic versioning.
 */
function getAbortSignalReason(signal) {
    if (!signal.aborted) {
        return undefined;
    }
    if (signal.reason !== undefined) {
        return signal.reason;
    }
    // AbortSignal.reason is available in Node.js v16, v18, and later,
    // and in all browsers since early 2022.
    const e = new Error("This operation was aborted");
    e.name = "AbortError";
    return e;
}

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * createContextValues creates a new ContextValues.
 */
function createContextValues() {
    return {
        get(key) {
            return key.id in this ? this[key.id] : key.defaultValue;
        },
        set(key, value) {
            this[key.id] = value;
            return this;
        },
        delete(key) {
            delete this[key.id];
            return this;
        },
    };
}

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * @private Internal code, does not follow semantic versioning.
 */
const headerContentType = "Content-Type";
const headerEncoding = "Grpc-Encoding";
const headerAcceptEncoding = "Grpc-Accept-Encoding";
const headerTimeout = "Grpc-Timeout";
const headerGrpcStatus = "Grpc-Status";
const headerGrpcMessage = "Grpc-Message";
const headerStatusDetailsBin = "Grpc-Status-Details-Bin";
const headerUserAgent = "User-Agent";

// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Hydrate a service descriptor.
 *
 * @private
 */
function serviceDesc(file, path, ...paths) {
    if (paths.length > 0) {
        throw new Error();
    }
    return file.services[path];
}

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Describes the file status.proto.
 */
const file_status = /*@__PURE__*/ fileDesc("CgxzdGF0dXMucHJvdG8SCmdvb2dsZS5ycGMiTgoGU3RhdHVzEgwKBGNvZGUYASABKAUSDwoHbWVzc2FnZRgCIAEoCRIlCgdkZXRhaWxzGAMgAygLMhQuZ29vZ2xlLnByb3RvYnVmLkFueUJeCg5jb20uZ29vZ2xlLnJwY0ILU3RhdHVzUHJvdG9QAVo3Z29vZ2xlLmdvbGFuZy5vcmcvZ2VucHJvdG8vZ29vZ2xlYXBpcy9ycGMvc3RhdHVzO3N0YXR1c6ICA1JQQ2IGcHJvdG8z", [file_google_protobuf_any]);
/**
 * Describes the message google.rpc.Status.
 * Use `create(StatusSchema)` to create a new message.
 */
const StatusSchema = /*@__PURE__*/ messageDesc(file_status, 0);

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * The value of the Grpc-Status header or trailer in case of success.
 * Used by the gRPC and gRPC-web protocols.
 *
 * @private Internal code, does not follow semantic versioning.
 */
const grpcStatusOk = "0";
/**
 * Find an error status in the given Headers object, which can be either
 * a trailer, or a header (as allowed for so-called trailers-only responses).
 * The field "grpc-status-details-bin" is inspected, and if not present,
 * the fields "grpc-status" and "grpc-message" are used.
 * Returns an error only if the gRPC status code is > 0.
 *
 * @private Internal code, does not follow semantic versioning.
 */
function findTrailerError(headerOrTrailer) {
    // TODO
    // let code: Code;
    // let message: string = "";
    var _a;
    // Prefer the protobuf-encoded data to the grpc-status header.
    const statusBytes = headerOrTrailer.get(headerStatusDetailsBin);
    if (statusBytes != null) {
        const status = decodeBinaryHeader(statusBytes, StatusSchema);
        if (status.code == 0) {
            return undefined;
        }
        const error = new ConnectError(status.message, status.code, headerOrTrailer);
        error.details = status.details.map((any) => ({
            type: any.typeUrl.substring(any.typeUrl.lastIndexOf("/") + 1),
            value: any.value,
        }));
        return error;
    }
    const grpcStatus = headerOrTrailer.get(headerGrpcStatus);
    if (grpcStatus != null) {
        if (grpcStatus === grpcStatusOk) {
            return undefined;
        }
        const code = parseInt(grpcStatus, 10);
        if (code in Code) {
            return new ConnectError(decodeURIComponent((_a = headerOrTrailer.get(headerGrpcMessage)) !== null && _a !== void 0 ? _a : ""), code, headerOrTrailer);
        }
        return new ConnectError(`invalid grpc-status: ${grpcStatus}`, Code.Internal, headerOrTrailer);
    }
    return undefined;
}

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Create a URL for the given RPC. This simply adds the qualified
 * service name, a slash, and the method name to the path of the given
 * baseUrl.
 *
 * For example, the baseUri https://example.com and method "Say" from
 * the service example.ElizaService results in:
 * https://example.com/example.ElizaService/Say
 *
 * This format is used by the protocols Connect, gRPC and Twirp.
 *
 * Note that this function also accepts a protocol-relative baseUrl.
 * If given an empty string or "/" as a baseUrl, it returns just the
 * path.
 */
function createMethodUrl(baseUrl, method) {
    return baseUrl
        .toString()
        .replace(/\/?$/, `/${method.parent.typeName}/${method.name}`);
}

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 *  Takes a partial protobuf messages of the
 *  specified message type as input, and returns full instances.
 */
function normalize(desc, message) {
    return create(desc, message);
}
/**
 * Takes an AsyncIterable of partial protobuf messages of the
 * specified message type as input, and yields full instances.
 */
function normalizeIterable(desc, input) {
    function transform(result) {
        if (result.done === true) {
            return result;
        }
        return {
            done: result.done,
            value: normalize(desc, result.value),
        };
    }
    return {
        [Symbol.asyncIterator]() {
            const it = input[Symbol.asyncIterator]();
            const res = {
                next: () => it.next().then(transform),
            };
            if (it.throw !== undefined) {
                res.throw = (e) => it.throw(e).then(transform);
            }
            if (it.return !== undefined) {
                res.return = (v) => it.return(v).then(transform);
            }
            return res;
        },
    };
}

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * applyInterceptors takes the given UnaryFn or ServerStreamingFn, and wraps
 * it with each of the given interceptors, returning a new UnaryFn or
 * ServerStreamingFn.
 */
function applyInterceptors(next, interceptors) {
    if (!interceptors) {
        return next;
    }
    for (const i of interceptors.concat().reverse()) {
        next = i(next);
    }
    return next;
}

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Sets default JSON serialization options for connect-es.
 *
 * With standard protobuf JSON serialization, unknown JSON fields are
 * rejected by default. In connect-es, unknown JSON fields are ignored
 * by default.
 */
function getJsonOptions(options) {
    var _a;
    const o = Object.assign({}, options);
    (_a = o.ignoreUnknownFields) !== null && _a !== void 0 ? _a : (o.ignoreUnknownFields = true);
    return o;
}
/**
 * Create an object that provides convenient access to request and response
 * message serialization for a given method.
 *
 * @private Internal code, does not follow semantic versioning.
 */
function createMethodSerializationLookup(method, binaryOptions, jsonOptions, limitOptions) {
    const inputBinary = limitSerialization(createBinarySerialization(method.input, binaryOptions), limitOptions);
    const inputJson = limitSerialization(createJsonSerialization(method.input, jsonOptions), limitOptions);
    const outputBinary = limitSerialization(createBinarySerialization(method.output, binaryOptions), limitOptions);
    const outputJson = limitSerialization(createJsonSerialization(method.output, jsonOptions), limitOptions);
    return {
        getI(useBinaryFormat) {
            return useBinaryFormat ? inputBinary : inputJson;
        },
        getO(useBinaryFormat) {
            return useBinaryFormat ? outputBinary : outputJson;
        },
    };
}
/**
 * Apply I/O limits to a Serialization object, returning a new object.
 *
 * @private Internal code, does not follow semantic versioning.
 */
function limitSerialization(serialization, limitOptions) {
    return {
        serialize(data) {
            const bytes = serialization.serialize(data);
            assertWriteMaxBytes(limitOptions.writeMaxBytes, bytes.byteLength);
            return bytes;
        },
        parse(data) {
            assertReadMaxBytes(limitOptions.readMaxBytes, data.byteLength, true);
            return serialization.parse(data);
        },
    };
}
/**
 * Creates a Serialization object for serializing the given protobuf message
 * with the protobuf binary format.
 */
function createBinarySerialization(desc, options) {
    return {
        parse(data) {
            try {
                return fromBinary(desc, data, options);
            }
            catch (e) {
                const m = e instanceof Error ? e.message : String(e);
                throw new ConnectError(`parse binary: ${m}`, Code.Internal);
            }
        },
        serialize(data) {
            try {
                return toBinary(desc, data, options);
            }
            catch (e) {
                const m = e instanceof Error ? e.message : String(e);
                throw new ConnectError(`serialize binary: ${m}`, Code.Internal);
            }
        },
    };
}
/**
 * Creates a Serialization object for serializing the given protobuf message
 * with the protobuf canonical JSON encoding.
 *
 * By default, unknown fields are ignored.
 */
function createJsonSerialization(desc, options) {
    var _a, _b;
    const textEncoder = (_a = options === null || options === void 0 ? void 0 : options.textEncoder) !== null && _a !== void 0 ? _a : new TextEncoder();
    const textDecoder = (_b = options === null || options === void 0 ? void 0 : options.textDecoder) !== null && _b !== void 0 ? _b : new TextDecoder();
    const o = getJsonOptions(options);
    return {
        parse(data) {
            try {
                const json = textDecoder.decode(data);
                return fromJsonString(desc, json, o);
            }
            catch (e) {
                throw ConnectError.from(e, Code.InvalidArgument);
            }
        },
        serialize(data) {
            try {
                const json = toJsonString(desc, data, o);
                return textEncoder.encode(json);
            }
            catch (e) {
                throw ConnectError.from(e, Code.Internal);
            }
        },
    };
}

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Regular Expression that matches any valid gRPC Content-Type header value.
 *
 * @private Internal code, does not follow semantic versioning.
 */
const contentTypeRegExp = /^application\/grpc(?:\+(?:(json)(?:; ?charset=utf-?8)?|proto))?$/i;
const contentTypeProto = "application/grpc+proto";
const contentTypeJson = "application/grpc+json";
/**
 * Parse a gRPC Content-Type header.
 *
 * @private Internal code, does not follow semantic versioning.
 */
function parseContentType(contentType) {
    const match = contentType === null || contentType === void 0 ? void 0 : contentType.match(contentTypeRegExp);
    if (!match) {
        return undefined;
    }
    const binary = !match[1];
    return { binary };
}

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Runs a unary method with the given interceptors. Note that this function
 * is only used when implementing a Transport.
 */
function runUnaryCall(opt) {
    const next = applyInterceptors(opt.next, opt.interceptors);
    const [signal, abort, done] = setupSignal(opt);
    const req = Object.assign(Object.assign({}, opt.req), { message: normalize(opt.req.method.input, opt.req.message), signal });
    return next(req).then((res) => {
        done();
        return res;
    }, abort);
}
/**
 * Runs a server-streaming method with the given interceptors. Note that this
 * function is only used when implementing a Transport.
 */
function runStreamingCall(opt) {
    const next = applyInterceptors(opt.next, opt.interceptors);
    const [signal, abort, done] = setupSignal(opt);
    const req = Object.assign(Object.assign({}, opt.req), { message: normalizeIterable(opt.req.method.input, opt.req.message), signal });
    let doneCalled = false;
    // Call return on the request iterable to indicate
    // that we will no longer consume it and it should
    // cleanup any allocated resources.
    signal.addEventListener("abort", function () {
        var _a, _b;
        const it = opt.req.message[Symbol.asyncIterator]();
        // If the signal is aborted due to an error, we want to throw
        // the error to the request iterator.
        if (!doneCalled) {
            (_a = it.throw) === null || _a === void 0 ? void 0 : _a.call(it, this.reason).catch(() => {
                // throw returns a promise, which we don't care about.
                //
                // Uncaught promises are thrown at sometime/somewhere by the event loop,
                // this is to ensure error is caught and ignored.
            });
        }
        (_b = it.return) === null || _b === void 0 ? void 0 : _b.call(it).catch(() => {
            // return returns a promise, which we don't care about.
            //
            // Uncaught promises are thrown at sometime/somewhere by the event loop,
            // this is to ensure error is caught and ignored.
        });
    });
    return next(req).then((res) => {
        return Object.assign(Object.assign({}, res), { message: {
                [Symbol.asyncIterator]() {
                    const it = res.message[Symbol.asyncIterator]();
                    return {
                        next() {
                            return it.next().then((r) => {
                                if (r.done == true) {
                                    doneCalled = true;
                                    done();
                                }
                                return r;
                            }, abort);
                        },
                        // We deliberately omit throw/return.
                    };
                },
            } });
    }, abort);
}
/**
 * Create an AbortSignal for Transport implementations. The signal is available
 * in UnaryRequest and StreamingRequest, and is triggered when the call is
 * aborted (via a timeout or explicit cancellation), errored (e.g. when reading
 * an error from the server from the wire), or finished successfully.
 *
 * Transport implementations can pass the signal to HTTP clients to ensure that
 * there are no unused connections leak.
 *
 * Returns a tuple:
 * [0]: The signal, which is also aborted if the optional deadline is reached.
 * [1]: Function to call if the Transport encountered an error.
 * [2]: Function to call if the Transport finished without an error.
 */
function setupSignal(opt) {
    const { signal, cleanup } = createDeadlineSignal(opt.timeoutMs);
    const controller = createLinkedAbortController(opt.signal, signal);
    return [
        controller.signal,
        function abort(reason) {
            // We peek at the deadline signal because fetch() will throw an error on
            // abort that discards the signal reason.
            const e = ConnectError.from(signal.aborted ? getAbortSignalReason(signal) : reason);
            controller.abort(e);
            cleanup();
            return Promise.reject(e);
        },
        function done() {
            cleanup();
            controller.abort();
        },
    ];
}

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Validates a trailer for the gRPC and the gRPC-web protocol.
 *
 * If the trailer contains an error status, a ConnectError is
 * thrown. It will include trailer and header in the error's
 * "metadata" property.
 *
 * Throws a ConnectError with code "internal" if neither the trailer
 * nor the header contain the Grpc-Status field.
 *
 * @private Internal code, does not follow semantic versioning.
 */
function validateTrailer(trailer, header) {
    const err = findTrailerError(trailer);
    if (err) {
        header.forEach((value, key) => {
            err.metadata.append(key, value);
        });
        throw err;
    }
    if (!header.has(headerGrpcStatus) && !trailer.has(headerGrpcStatus)) {
        throw new ConnectError("protocol error: missing status", Code.Internal);
    }
}

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Determine the gRPC-web error code for the given HTTP status code.
 * See https://github.com/grpc/grpc/blob/master/doc/http-grpc-status-mapping.md.
 *
 * @private Internal code, does not follow semantic versioning.
 */
function codeFromHttpStatus(httpStatus) {
    switch (httpStatus) {
        case 400: // Bad Request
            return Code.Internal;
        case 401: // Unauthorized
            return Code.Unauthenticated;
        case 403: // Forbidden
            return Code.PermissionDenied;
        case 404: // Not Found
            return Code.Unimplemented;
        case 429: // Too Many Requests
            return Code.Unavailable;
        case 502: // Bad Gateway
            return Code.Unavailable;
        case 503: // Service Unavailable
            return Code.Unavailable;
        case 504: // Gateway Timeout
            return Code.Unavailable;
        default:
            // 200 is UNKNOWN because there should be a grpc-status in case of truly OK response.
            return Code.Unknown;
    }
}

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Creates headers for a gRPC request.
 *
 * @private Internal code, does not follow semantic versioning.
 */
function requestHeader(useBinaryFormat, timeoutMs, userProvidedHeaders) {
    const result = new Headers(userProvidedHeaders !== null && userProvidedHeaders !== void 0 ? userProvidedHeaders : {});
    result.set(headerContentType, useBinaryFormat ? contentTypeProto : contentTypeJson);
    if (!result.has(headerUserAgent)) {
        // Note that we do not strictly comply with gRPC user agents.
        // We use "connect-es/1.2.3" where gRPC would use "grpc-es/1.2.3".
        // See https://github.com/grpc/grpc/blob/c462bb8d485fc1434ecfae438823ca8d14cf3154/doc/PROTOCOL-HTTP2.md#user-agents
        result.set(headerUserAgent, "connect-es/2.1.0");
    }
    if (timeoutMs !== undefined) {
        result.set(headerTimeout, `${timeoutMs}m`);
    }
    // The gRPC-HTTP2 specification requires this - it flushes out proxies that
    // don't support HTTP trailers.
    result.set("Te", "trailers");
    return result;
}
/**
 * Creates headers for a gRPC request with compression.
 *
 * @private Internal code, does not follow semantic versioning.
 */
function requestHeaderWithCompression(useBinaryFormat, timeoutMs, userProvidedHeaders, acceptCompression, sendCompression) {
    const result = requestHeader(useBinaryFormat, timeoutMs, userProvidedHeaders);
    if (sendCompression != null) {
        result.set(headerEncoding, sendCompression.name);
    }
    if (acceptCompression.length > 0) {
        result.set(headerAcceptEncoding, acceptCompression.map((c) => c.name).join(","));
    }
    return result;
}

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Validates response status and header for the gRPC protocol.
 * Throws a ConnectError if the header contains an error status,
 * or if the HTTP status indicates an error.
 *
 * Returns an object that indicates whether a gRPC status was found
 * in the response header. In this case, clients can not expect a
 * trailer.
 *
 * @private Internal code, does not follow semantic versioning.
 */
function validateResponse(status, headers) {
    if (status != 200) {
        throw new ConnectError(`HTTP ${status}`, codeFromHttpStatus(status), headers);
    }
    const mimeType = headers.get(headerContentType);
    const parsedType = parseContentType(mimeType);
    if (parsedType == undefined) {
        throw new ConnectError(`unsupported content type ${mimeType}`, Code.Unknown);
    }
    return {
        foundStatus: headers.has(headerGrpcStatus),
        headerError: findTrailerError(headers),
    };
}
/**
 * Validates response status and header for the gRPC protocol.
 * This function is identical to validateResponse(), but also verifies
 * that a given encoding header is acceptable.
 *
 * Returns an object with the response compression, and a boolean
 * indicating whether a gRPC status was found in the response header
 * (in this case, clients can not expect a trailer).
 *
 * @private Internal code, does not follow semantic versioning.
 */
function validateResponseWithCompression(acceptCompression, status, headers) {
    const { foundStatus, headerError } = validateResponse(status, headers);
    let compression;
    const encoding = headers.get(headerEncoding);
    if (encoding !== null && encoding.toLowerCase() !== "identity") {
        compression = acceptCompression.find((c) => c.name === encoding);
        if (!compression) {
            throw new ConnectError(`unsupported response encoding "${encoding}"`, Code.Internal, headers);
        }
    }
    return {
        foundStatus,
        compression,
        headerError,
    };
}

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
var __asyncValues = (undefined && undefined.__asyncValues) || function (o) {
    if (!Symbol.asyncIterator) throw new TypeError("Symbol.asyncIterator is not defined.");
    var m = o[Symbol.asyncIterator], i;
    return m ? m.call(o) : (o = typeof __values === "function" ? __values(o) : o[Symbol.iterator](), i = {}, verb("next"), verb("throw"), verb("return"), i[Symbol.asyncIterator] = function () { return this; }, i);
    function verb(n) { i[n] = o[n] && function (v) { return new Promise(function (resolve, reject) { v = o[n](v), settle(resolve, reject, v.done, v.value); }); }; }
    function settle(resolve, reject, d, v) { Promise.resolve(v).then(function(v) { resolve({ value: v, done: d }); }, reject); }
};
var __await = (undefined && undefined.__await) || function (v) { return this instanceof __await ? (this.v = v, this) : new __await(v); };
var __asyncDelegator = (undefined && undefined.__asyncDelegator) || function (o) {
    var i, p;
    return i = {}, verb("next"), verb("throw", function (e) { throw e; }), verb("return"), i[Symbol.iterator] = function () { return this; }, i;
    function verb(n, f) { i[n] = o[n] ? function (v) { return (p = !p) ? { value: __await(o[n](v)), done: false } : f ? f(v) : v; } : f; }
};
var __asyncGenerator = (undefined && undefined.__asyncGenerator) || function (thisArg, _arguments, generator) {
    if (!Symbol.asyncIterator) throw new TypeError("Symbol.asyncIterator is not defined.");
    var g = generator.apply(thisArg, _arguments || []), i, q = [];
    return i = Object.create((typeof AsyncIterator === "function" ? AsyncIterator : Object).prototype), verb("next"), verb("throw"), verb("return", awaitReturn), i[Symbol.asyncIterator] = function () { return this; }, i;
    function awaitReturn(f) { return function (v) { return Promise.resolve(v).then(f, reject); }; }
    function verb(n, f) { if (g[n]) { i[n] = function (v) { return new Promise(function (a, b) { q.push([n, v, a, b]) > 1 || resume(n, v); }); }; if (f) i[n] = f(i[n]); } }
    function resume(n, v) { try { step(g[n](v)); } catch (e) { settle(q[0][3], e); } }
    function step(r) { r.value instanceof __await ? Promise.resolve(r.value.v).then(fulfill, reject) : settle(q[0][2], r); }
    function fulfill(value) { resume("next", value); }
    function reject(value) { resume("throw", value); }
    function settle(f, v) { if (f(v), q.shift(), q.length) resume(q[0][0], q[0][1]); }
};
/**
 * Create a Transport for the gRPC protocol.
 */
function createTransport(opt) {
    return {
        async unary(method, signal, timeoutMs, header, message, contextValues) {
            const serialization = createMethodSerializationLookup(method, opt.binaryOptions, opt.jsonOptions, opt);
            timeoutMs =
                timeoutMs === undefined
                    ? opt.defaultTimeoutMs
                    : timeoutMs <= 0
                        ? undefined
                        : timeoutMs;
            return await runUnaryCall({
                interceptors: opt.interceptors,
                signal,
                timeoutMs,
                req: {
                    stream: false,
                    service: method.parent,
                    method,
                    requestMethod: "POST",
                    url: createMethodUrl(opt.baseUrl, method),
                    header: requestHeaderWithCompression(opt.useBinaryFormat, timeoutMs, header, opt.acceptCompression, opt.sendCompression),
                    contextValues: contextValues !== null && contextValues !== void 0 ? contextValues : createContextValues(),
                    message,
                },
                next: async (req) => {
                    const uRes = await opt.httpClient({
                        url: req.url,
                        method: "POST",
                        header: req.header,
                        signal: req.signal,
                        body: pipe(createAsyncIterable([req.message]), transformSerializeEnvelope(serialization.getI(opt.useBinaryFormat)), transformCompressEnvelope(opt.sendCompression, opt.compressMinBytes), transformJoinEnvelopes(), {
                            propagateDownStreamError: true,
                        }),
                    });
                    const { compression, headerError } = validateResponseWithCompression(opt.acceptCompression, uRes.status, uRes.header);
                    const message = await pipeTo(uRes.body, transformSplitEnvelope(opt.readMaxBytes), transformDecompressEnvelope(compression !== null && compression !== void 0 ? compression : null, opt.readMaxBytes), transformParseEnvelope(serialization.getO(opt.useBinaryFormat)), async (iterable) => {
                        var _a, e_1, _b, _c;
                        let message;
                        try {
                            for (var _d = true, iterable_1 = __asyncValues(iterable), iterable_1_1; iterable_1_1 = await iterable_1.next(), _a = iterable_1_1.done, !_a; _d = true) {
                                _c = iterable_1_1.value;
                                _d = false;
                                const chunk = _c;
                                if (message !== undefined) {
                                    throw new ConnectError("protocol error: received extra output message for unary method", Code.Unimplemented);
                                }
                                message = chunk;
                            }
                        }
                        catch (e_1_1) { e_1 = { error: e_1_1 }; }
                        finally {
                            try {
                                if (!_d && !_a && (_b = iterable_1.return)) await _b.call(iterable_1);
                            }
                            finally { if (e_1) throw e_1.error; }
                        }
                        return message;
                    }, { propagateDownStreamError: false });
                    validateTrailer(uRes.trailer, uRes.header);
                    if (message === undefined) {
                        // Trailers only response
                        if (headerError) {
                            throw headerError;
                        }
                        throw new ConnectError("protocol error: missing output message for unary method", uRes.trailer.has(headerGrpcStatus)
                            ? Code.Unimplemented
                            : Code.Unknown);
                    }
                    if (headerError) {
                        throw new ConnectError("protocol error: received output message for unary method with error status", Code.Unknown);
                    }
                    return {
                        stream: false,
                        service: method.parent,
                        method,
                        header: uRes.header,
                        message,
                        trailer: uRes.trailer,
                    };
                },
            });
        },
        async stream(method, signal, timeoutMs, header, input, contextValues) {
            const serialization = createMethodSerializationLookup(method, opt.binaryOptions, opt.jsonOptions, opt);
            timeoutMs =
                timeoutMs === undefined
                    ? opt.defaultTimeoutMs
                    : timeoutMs <= 0
                        ? undefined
                        : timeoutMs;
            return runStreamingCall({
                interceptors: opt.interceptors,
                signal,
                timeoutMs,
                req: {
                    stream: true,
                    service: method.parent,
                    method,
                    requestMethod: "POST",
                    url: createMethodUrl(opt.baseUrl, method),
                    header: requestHeaderWithCompression(opt.useBinaryFormat, timeoutMs, header, opt.acceptCompression, opt.sendCompression),
                    contextValues: contextValues !== null && contextValues !== void 0 ? contextValues : createContextValues(),
                    message: input,
                },
                next: async (req) => {
                    const uRes = await opt.httpClient({
                        url: req.url,
                        method: "POST",
                        header: req.header,
                        signal: req.signal,
                        body: pipe(req.message, transformSerializeEnvelope(serialization.getI(opt.useBinaryFormat)), transformCompressEnvelope(opt.sendCompression, opt.compressMinBytes), transformJoinEnvelopes(), { propagateDownStreamError: true }),
                    });
                    const { compression, foundStatus, headerError } = validateResponseWithCompression(opt.acceptCompression, uRes.status, uRes.header);
                    if (headerError) {
                        throw headerError;
                    }
                    const res = Object.assign(Object.assign({}, req), { header: uRes.header, trailer: uRes.trailer, message: pipe(uRes.body, transformSplitEnvelope(opt.readMaxBytes), transformDecompressEnvelope(compression !== null && compression !== void 0 ? compression : null, opt.readMaxBytes), transformParseEnvelope(serialization.getO(opt.useBinaryFormat)), function (iterable) {
                            return __asyncGenerator(this, arguments, function* () {
                                yield __await(yield* __asyncDelegator(__asyncValues(iterable)));
                                if (!foundStatus) {
                                    validateTrailer(uRes.trailer, uRes.header);
                                }
                            });
                        }, { propagateDownStreamError: true }) });
                    return res;
                },
            });
        },
    };
}

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Similar to ConnectError.from(), this function turns any value into
 * a ConnectError, but special cases some Node.js specific error codes and
 * sets an appropriate Connect error code.
 */
function connectErrorFromNodeReason(reason) {
    let code = Code.Internal;
    const chain = unwrapNodeErrorChain(reason).map(getNodeErrorProps);
    if (chain.some((p) => p.code == "ERR_STREAM_WRITE_AFTER_END")) {
        // We do not want intentional errors from the server to be shadowed
        // by client-side errors.
        // This can occur if the server has written a response with an error
        // and has ended the connection. This response may already sit in a
        // buffer on the client, while it is still writing to the request
        // body.
        // To avoid this problem, we wrap ERR_STREAM_WRITE_AFTER_END as a
        // ConnectError with Code.Aborted. The special meaning of this code
        // in this situation is documented in StreamingConn.send() and in
        // createServerStreamingFn().
        code = Code.Aborted;
    }
    else if (chain.some((p) => p.code == "ERR_STREAM_DESTROYED" ||
        p.code == "ERR_HTTP2_INVALID_STREAM" ||
        p.code == "ECONNRESET")) {
        // A handler whose stream is suddenly destroyed usually means the client
        // hung up. This behavior is triggered by the conformance test "cancel_after_begin".
        code = Code.Aborted;
    }
    else if (chain.some((p) => p.code == "ETIMEDOUT" ||
        p.code == "ENOTFOUND" ||
        p.code == "EAI_AGAIN" ||
        p.code == "ECONNREFUSED")) {
        // Calling an unresolvable host should raise a ConnectError with
        // Code.Aborted.
        // This behavior is covered by the conformance test "unresolvable_host".
        code = Code.Unavailable;
    }
    const ce = ConnectError.from(reason, code);
    if (ce !== reason) {
        ce.cause = reason;
    }
    return ce;
}
/**
 * Unwraps a chain of errors, by walking through all "cause" properties
 * recursively.
 * This function is useful to find the root cause of an error.
 */
function unwrapNodeErrorChain(reason) {
    const chain = [];
    for (;;) {
        if (!(reason instanceof Error)) {
            break;
        }
        if (chain.includes(reason)) {
            // safeguard against infinite loop when "cause" points to an ancestor
            break;
        }
        chain.push(reason);
        if (!("cause" in reason)) {
            break;
        }
        reason = reason.cause;
    }
    return chain;
}
/**
 * Returns standard Node.js error properties from the given reason, if present.
 *
 * For context: Every error raised by Node.js APIs should expose a `code`
 * property - a string that permanently identifies the error. Some errors may
 * have an additional `syscall` property - a string that identifies native
 * components, for example "getaddrinfo" of libuv.
 * For more information, see https://github.com/nodejs/node/blob/f6052c68c1f9a4400a723e9c0b79da14197ab754/lib/internal/errors.js
 */
function getNodeErrorProps(reason) {
    const props = {};
    if (reason instanceof Error) {
        if ("code" in reason && typeof reason.code == "string") {
            props.code = reason.code;
        }
        if ("syscall" in reason && typeof reason.syscall == "string") {
            props.syscall = reason.syscall;
        }
    }
    return props;
}
/**
 * Returns a ConnectError for an HTTP/2 error code.
 */
function connectErrorFromH2ResetCode(rstCode) {
    switch (rstCode) {
        case H2Code.PROTOCOL_ERROR:
        case H2Code.INTERNAL_ERROR:
        case H2Code.FLOW_CONTROL_ERROR:
        case H2Code.SETTINGS_TIMEOUT:
        case H2Code.FRAME_SIZE_ERROR:
        case H2Code.COMPRESSION_ERROR:
        case H2Code.CONNECT_ERROR:
            return new ConnectError(`http/2 stream closed with error code ${H2Code[rstCode]} (0x${rstCode.toString(16)})`, Code.Internal);
        case H2Code.REFUSED_STREAM:
            return new ConnectError(`http/2 stream closed with error code ${H2Code[rstCode]} (0x${rstCode.toString(16)})`, Code.Unavailable);
        case H2Code.CANCEL:
            return new ConnectError(`http/2 stream closed with error code ${H2Code[rstCode]} (0x${rstCode.toString(16)})`, Code.Canceled);
        case H2Code.ENHANCE_YOUR_CALM:
            return new ConnectError(`http/2 stream closed with error code ${H2Code[rstCode]} (0x${rstCode.toString(16)})`, Code.ResourceExhausted);
        case H2Code.INADEQUATE_SECURITY:
            return new ConnectError(`http/2 stream closed with error code ${H2Code[rstCode]} (0x${rstCode.toString(16)})`, Code.PermissionDenied);
        case H2Code.HTTP_1_1_REQUIRED:
            return new ConnectError(`http/2 stream closed with error code ${H2Code[rstCode]} (0x${rstCode.toString(16)})`, Code.PermissionDenied);
    }
    return undefined;
}
var H2Code;
(function (H2Code) {
    H2Code[H2Code["PROTOCOL_ERROR"] = 1] = "PROTOCOL_ERROR";
    H2Code[H2Code["INTERNAL_ERROR"] = 2] = "INTERNAL_ERROR";
    H2Code[H2Code["FLOW_CONTROL_ERROR"] = 3] = "FLOW_CONTROL_ERROR";
    H2Code[H2Code["SETTINGS_TIMEOUT"] = 4] = "SETTINGS_TIMEOUT";
    H2Code[H2Code["STREAM_CLOSED"] = 5] = "STREAM_CLOSED";
    H2Code[H2Code["FRAME_SIZE_ERROR"] = 6] = "FRAME_SIZE_ERROR";
    H2Code[H2Code["REFUSED_STREAM"] = 7] = "REFUSED_STREAM";
    H2Code[H2Code["CANCEL"] = 8] = "CANCEL";
    H2Code[H2Code["COMPRESSION_ERROR"] = 9] = "COMPRESSION_ERROR";
    H2Code[H2Code["CONNECT_ERROR"] = 10] = "CONNECT_ERROR";
    H2Code[H2Code["ENHANCE_YOUR_CALM"] = 11] = "ENHANCE_YOUR_CALM";
    H2Code[H2Code["INADEQUATE_SECURITY"] = 12] = "INADEQUATE_SECURITY";
    H2Code[H2Code["HTTP_1_1_REQUIRED"] = 13] = "HTTP_1_1_REQUIRED";
})(H2Code || (H2Code = {}));

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
const gzip = promisify(zlib.gzip);
const gunzip = promisify(zlib.gunzip);
const brotliCompress = promisify(zlib.brotliCompress);
const brotliDecompress = promisify(zlib.brotliDecompress);
/**
 * The gzip compression algorithm, implemented with the Node.js built-in module
 * zlib. This value can be used for the `sendCompression` and `acceptCompression`
 * option of client transports, or for the `acceptCompression` option of server
 * plugins like `fastifyConnectPlugin` from @connectrpc/connect-fastify.
 */
const compressionGzip = {
    name: "gzip",
    compress(bytes) {
        return asUint8ArrayArrayBuffer(gzip(bytes, {}));
    },
    decompress(bytes, readMaxBytes) {
        if (bytes.length == 0) {
            return Promise.resolve(new Uint8Array(0));
        }
        return asUint8ArrayArrayBuffer(wrapZLibErrors(gunzip(bytes, {
            maxOutputLength: readMaxBytes,
        }), readMaxBytes));
    },
};
/**
 * The brotli compression algorithm, implemented with the Node.js built-in module
 * zlib. This value can be used for the `sendCompression` and `acceptCompression`
 * option of client transports, or for the `acceptCompression` option of server
 * plugins like `fastifyConnectPlugin` from @connectrpc/connect-fastify.
 */
const compressionBrotli = {
    name: "br",
    compress(bytes) {
        return asUint8ArrayArrayBuffer(brotliCompress(bytes, {}));
    },
    decompress(bytes, readMaxBytes) {
        if (bytes.length == 0) {
            return Promise.resolve(new Uint8Array(0));
        }
        return asUint8ArrayArrayBuffer(wrapZLibErrors(brotliDecompress(bytes, {
            maxOutputLength: readMaxBytes,
        }), readMaxBytes));
    },
};
function asUint8ArrayArrayBuffer(bytes) {
    return bytes.then((b) => {
        if (b.buffer instanceof ArrayBuffer) {
            return b;
        }
        return new Uint8Array(b);
    });
}
function wrapZLibErrors(promise, readMaxBytes) {
    return promise.catch((e) => {
        var _a;
        const props = getNodeErrorProps(e);
        let code = Code.Internal;
        let message = "decompression failed";
        switch (props.code) {
            case "ERR_BUFFER_TOO_LARGE":
                code = Code.ResourceExhausted;
                message = `message is larger than configured readMaxBytes ${readMaxBytes} after decompression`;
                break;
            case "Z_DATA_ERROR":
            case "ERR_PADDING_2":
                code = Code.InvalidArgument;
                break;
            default:
                if ((_a = props.code) === null || _a === void 0 ? void 0 : _a.startsWith("ERR__ERROR_FORMAT_")) {
                    code = Code.InvalidArgument;
                }
                break;
        }
        return Promise.reject(new ConnectError(message, code, undefined, undefined, e));
    });
}

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Convert a Node.js header object to a fetch API Headers object.
 *
 * This function works for Node.js incoming and outgoing headers, and for the
 * http and the http2 package.
 *
 * HTTP/2 pseudo-headers (:method, :path, etc.) are stripped.
 */
function nodeHeaderToWebHeader(nodeHeaders) {
    const header = new Headers();
    for (const [k, v] of Object.entries(nodeHeaders)) {
        if (k.startsWith(":")) {
            continue;
        }
        if (v === undefined) {
            continue;
        }
        if (typeof v == "string") {
            header.append(k, v);
        }
        else if (typeof v == "number") {
            header.append(k, String(v));
        }
        else {
            for (const e of v) {
                header.append(k, e);
            }
        }
    }
    return header;
}
function webHeaderToNodeHeaders(headersInit, defaultNodeHeaders) {
    if (headersInit === undefined && defaultNodeHeaders === undefined) {
        return undefined;
    }
    const o = Object.create(null);
    if (defaultNodeHeaders !== undefined) {
        if (Array.isArray(defaultNodeHeaders)) {
            // headers may be an Array where the keys and values are in the same list.
            // It is _not_ a list of tuples. So, the even-numbered offsets are key values,
            // and the odd-numbered offsets are the associated values.
            for (let i = 0; i + 1 < defaultNodeHeaders.length; i += 2) {
                const key = defaultNodeHeaders[i];
                const value = defaultNodeHeaders[i + 1];
                if (Array.isArray(o[key])) {
                    o[key].push(value);
                }
                else if (typeof o[key] == "string") {
                    o[key] = [o[key], value];
                }
                else {
                    o[key] = value;
                }
            }
        }
        else {
            for (const [key, value] of Object.entries(defaultNodeHeaders)) {
                if (Array.isArray(value)) {
                    o[key] = value.concat();
                }
                else if (value !== undefined) {
                    o[key] = value;
                }
            }
        }
    }
    if (headersInit !== undefined) {
        if (Array.isArray(headersInit)) {
            for (const [key, value] of headersInit) {
                appendWebHeader(o, key, value);
            }
        }
        else if ("forEach" in headersInit) {
            if (typeof headersInit.forEach == "function") {
                headersInit.forEach((value, key) => {
                    appendWebHeader(o, key, value);
                });
            }
        }
        else {
            for (const [key, value] of Object.entries(headersInit)) {
                appendWebHeader(o, key, value);
            }
        }
    }
    return o;
}
function appendWebHeader(o, key, value) {
    key = key.toLowerCase();
    const existing = o[key];
    if (Array.isArray(existing)) {
        existing.push(value);
    }
    else if (existing === undefined) {
        o[key] = value;
    }
    else {
        o[key] = [existing.toString(), value];
    }
}

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Manage an HTTP/2 connection and keep it alive with PING frames.
 *
 * The logic is based on "Basic Keepalive" described in
 * https://github.com/grpc/proposal/blob/0ba0c1905050525f9b0aee46f3f23c8e1e515489/A8-client-side-keepalive.md#basic-keepalive
 * as well as the client channel arguments described in
 * https://github.com/grpc/grpc/blob/8e137e524a1b1da7bbf4603662876d5719563b57/doc/keepalive.md
 *
 * Usually, the managers tracks exactly one connection, but if a connection
 * receives a GOAWAY frame with NO_ERROR, the connection is maintained until
 * all streams have finished, and new requests will open a new connection.
 */
class Http2SessionManager {
    /**
     * The current state of the connection:
     *
     * - "closed"
     *   The connection is closed, or no connection has been opened yet.
     * - connecting
     *   Currently establishing a connection.
     *
     * - "open"
     *   A connection is open and has open streams. PING frames are sent every
     *   pingIntervalMs, unless a stream received data.
     *   If a PING frame is not responded to within pingTimeoutMs, the connection
     *   and all open streams close.
     *
     * - "idle"
     *   A connection is open, but it does not have any open streams.
     *   If pingIdleConnection is enabled, PING frames are used to keep the
     *   connection alive, similar to an "open" connection.
     *   If a connection is idle for longer than idleConnectionTimeoutMs, it closes.
     *   If a request is made on an idle connection that has not been used for
     *   longer than pingIntervalMs, the connection is verified.
     *
     * - "verifying"
     *   Verifying a connection after a long period of inactivity before issuing a
     *   request. A PING frame is sent, and if it times out within pingTimeoutMs, a
     *   new connection is opened.
     *
     * - "error"
     *   The connection is closed because of a transient error. A connection
     *   may have failed to reach the host, or the connection may have died,
     *   or it may have been aborted.
     */
    state() {
        if (this.s.t == "ready") {
            if (this.verifying !== undefined) {
                return "verifying";
            }
            return this.s.streamCount() > 0 ? "open" : "idle";
        }
        return this.s.t;
    }
    /**
     * Returns the error object if the connection is in the "error" state,
     * `undefined` otherwise.
     */
    error() {
        if (this.s.t == "error") {
            return this.s.reason;
        }
        return undefined;
    }
    constructor(url, pingOptions, http2SessionOptions) {
        var _a, _b, _c, _d;
        this.s = closed();
        this.shuttingDown = [];
        this.authority = new URL(url).origin;
        this.http2SessionOptions = http2SessionOptions;
        this.options = {
            pingIntervalMs: (_a = pingOptions === null || pingOptions === void 0 ? void 0 : pingOptions.pingIntervalMs) !== null && _a !== void 0 ? _a : Number.POSITIVE_INFINITY,
            pingTimeoutMs: (_b = pingOptions === null || pingOptions === void 0 ? void 0 : pingOptions.pingTimeoutMs) !== null && _b !== void 0 ? _b : 1000 * 15,
            pingIdleConnection: (_c = pingOptions === null || pingOptions === void 0 ? void 0 : pingOptions.pingIdleConnection) !== null && _c !== void 0 ? _c : false,
            idleConnectionTimeoutMs: (_d = pingOptions === null || pingOptions === void 0 ? void 0 : pingOptions.idleConnectionTimeoutMs) !== null && _d !== void 0 ? _d : 1000 * 60 * 15,
        };
    }
    /**
     * Open a connection if none exists, verify an existing connection if
     * necessary.
     */
    async connect() {
        try {
            const ready = await this.gotoReady();
            return ready.streamCount() > 0 ? "open" : "idle";
        }
        catch (e) {
            return "error";
        }
    }
    /**
     * Issue a request.
     *
     * This method automatically opens a connection if none exists, and verifies
     * an existing connection if necessary. It calls http2.ClientHttp2Session.request(),
     * and keeps track of all open http2.ClientHttp2Stream.
     *
     * Clients must call notifyResponseByteRead() whenever they successfully read
     * data from the http2.ClientHttp2Stream.
     */
    async request(method, path, headers, options) {
        // Request sometimes fails with goaway/destroyed
        // errors, we use a loop to retry.
        //
        // This is not expected to happen often, but it is possible that a
        // connection is closed while we are trying to open a stream.
        //
        // Ref: https://github.com/nodejs/help/issues/2105
        for (;;) {
            const ready = await this.gotoReady();
            try {
                const stream = ready.conn.request(Object.assign(Object.assign({}, headers), { ":method": method, ":path": path }), options);
                ready.registerRequest(stream);
                return stream;
            }
            catch (e) {
                // Check to see if the connection is closed or destroyed
                // and if so, we try again.
                if (ready.conn.closed || ready.conn.destroyed) {
                    continue;
                }
                throw e;
            }
        }
    }
    /**
     * Notify the manager of a successful read from a http2.ClientHttp2Stream.
     *
     * Clients must call this function whenever they successfully read data from
     * a http2.ClientHttp2Stream obtained from request(). This informs the
     * keep-alive logic that the connection is alive, and prevents it from sending
     * unnecessary PING frames.
     */
    notifyResponseByteRead(stream) {
        if (this.s.t == "ready") {
            this.s.responseByteRead(stream);
        }
        for (const s of this.shuttingDown) {
            s.responseByteRead(stream);
        }
    }
    /**
     * If there is an open connection, close it. This also closes any open streams.
     */
    abort(reason) {
        var _a, _b, _c;
        const err = reason !== null && reason !== void 0 ? reason : new ConnectError("connection aborted", Code.Canceled);
        (_b = (_a = this.s).abort) === null || _b === void 0 ? void 0 : _b.call(_a, err);
        for (const s of this.shuttingDown) {
            (_c = s.abort) === null || _c === void 0 ? void 0 : _c.call(s, err);
        }
        this.setState(closedOrError(err));
    }
    async gotoReady() {
        if (this.s.t == "ready") {
            if (this.s.isShuttingDown() ||
                this.s.conn.closed ||
                this.s.conn.destroyed) {
                this.setState(connect(this.authority, this.http2SessionOptions));
            }
            else if (this.s.requiresVerify()) {
                await this.verify(this.s);
            }
        }
        else if (this.s.t == "closed" || this.s.t == "error") {
            this.setState(connect(this.authority, this.http2SessionOptions));
        }
        while (this.s.t !== "ready") {
            if (this.s.t === "error") {
                throw this.s.reason;
            }
            if (this.s.t === "connecting") {
                await this.s.conn;
            }
        }
        return this.s;
    }
    setState(state) {
        var _a, _b;
        (_b = (_a = this.s).onExitState) === null || _b === void 0 ? void 0 : _b.call(_a);
        if (this.s.t == "ready" && this.s.isShuttingDown()) {
            // Maintain connections that have been asked to shut down.
            const sd = this.s;
            this.shuttingDown.push(sd);
            sd.onClose = sd.onError = () => {
                const i = this.shuttingDown.indexOf(sd);
                if (i !== -1) {
                    this.shuttingDown.splice(i, 1);
                }
            };
        }
        switch (state.t) {
            case "connecting":
                state.conn.then((value) => {
                    this.setState(ready(value, this.options));
                }, (reason) => {
                    this.setState(closedOrError(reason));
                });
                break;
            case "ready":
                state.onClose = () => this.setState(closed());
                state.onError = (err) => this.setState(closedOrError(err));
                break;
        }
        this.s = state;
    }
    verify(stateReady) {
        if (this.verifying !== undefined) {
            return this.verifying;
        }
        this.verifying = stateReady
            .verify()
            .then((success) => {
            if (success) {
                return;
            }
            // verify() has destroyed the old connection
            this.setState(connect(this.authority, this.http2SessionOptions));
        }, (reason) => {
            this.setState(closedOrError(reason));
        })
            .finally(() => {
            this.verifying = undefined;
        });
        return this.verifying;
    }
}
function closed() {
    return {
        t: "closed",
    };
}
function error(reason) {
    return {
        t: "error",
        reason,
    };
}
function closedOrError(reason) {
    const isCancel = reason instanceof ConnectError &&
        ConnectError.from(reason).code == Code.Canceled;
    return isCancel ? closed() : error(reason);
}
function connect(authority, http2SessionOptions) {
    let resolve;
    let reject;
    const conn = new Promise((res, rej) => {
        resolve = res;
        reject = rej;
    });
    const newConn = http2.connect(authority, http2SessionOptions);
    newConn.on("connect", onConnect);
    newConn.on("error", onError);
    function onConnect() {
        resolve === null || resolve === void 0 ? void 0 : resolve(newConn);
        cleanup();
    }
    function onError(err) {
        reject === null || reject === void 0 ? void 0 : reject(connectErrorFromNodeReason(err));
        cleanup();
    }
    function cleanup() {
        newConn.off("connect", onConnect);
        newConn.off("error", onError);
    }
    return {
        t: "connecting",
        conn,
        abort(reason) {
            if (!newConn.destroyed) {
                newConn.destroy(undefined, http2.constants.NGHTTP2_CANCEL);
            }
            // According to the documentation, destroy() should immediately terminate
            // the session and the socket, but we still receive a "connect" event.
            // We must not resolve a broken connection, so we reject it manually here.
            reject === null || reject === void 0 ? void 0 : reject(reason);
        },
        onExitState() {
            cleanup();
        },
    };
}
function ready(conn, options) {
    // Users have reported an error "The session has been destroyed" raised
    // from H2SessionManager.request(), see https://github.com/connectrpc/connect-es/issues/683
    // This assertion will show whether the session already died in the
    // "connecting" state.
    assertSessionOpen(conn);
    // Do not block Node.js from exiting on an idle connection.
    // Note that we ref() again for the first stream to open, and unref() again
    // for the last stream to close.
    conn.unref();
    // the last time we were sure that the connection is alive, via a PING
    // response, or via received response bytes
    let lastAliveAt = Date.now();
    // how many streams are currently open on this session
    let streamCount = 0;
    // timer for the keep-alive interval
    let pingIntervalId;
    // timer for waiting for a PING response
    let pingTimeoutId;
    // keep track of GOAWAY - gracefully shut down open streams / wait for connection to error
    let receivedGoAway = false;
    // keep track of GOAWAY with ENHANCE_YOUR_CALM and with debug data too_many_pings
    let receivedGoAwayEnhanceYourCalmTooManyPings = false;
    // timer for closing connections without open streams, must be initialized
    let idleTimeoutId;
    resetIdleTimeout();
    const state = {
        t: "ready",
        conn,
        streamCount() {
            return streamCount;
        },
        requiresVerify() {
            const elapsedMs = Date.now() - lastAliveAt;
            return elapsedMs > options.pingIntervalMs;
        },
        isShuttingDown() {
            return receivedGoAway;
        },
        onClose: undefined,
        onError: undefined,
        registerRequest(stream) {
            streamCount++;
            if (streamCount == 1) {
                conn.ref();
                resetPingInterval(); // reset to ping with the appropriate interval for "open"
                stopIdleTimeout();
            }
            stream.once("response", () => {
                lastAliveAt = Date.now();
                resetPingInterval();
            });
            stream.once("close", () => {
                streamCount--;
                if (streamCount == 0) {
                    conn.unref();
                    resetPingInterval(); // reset to ping with the appropriate interval for "idle"
                    resetIdleTimeout();
                }
            });
        },
        responseByteRead(stream) {
            if (stream.session !== conn) {
                return;
            }
            if (conn.closed || conn.destroyed) {
                return;
            }
            if (streamCount <= 0) {
                return;
            }
            lastAliveAt = Date.now();
            resetPingInterval();
        },
        verify() {
            conn.ref();
            return new Promise((resolve) => {
                commonPing(() => {
                    if (streamCount == 0)
                        conn.unref();
                    resolve(true);
                });
                conn.once("error", () => resolve(false));
            });
        },
        abort(reason) {
            if (!conn.destroyed) {
                conn.once("error", () => {
                    // conn.destroy() may raise an error after onExitState() was called
                    // and our error listeners are removed.
                    // We attach this one to swallow uncaught exceptions.
                });
                conn.destroy(reason, http2.constants.NGHTTP2_CANCEL);
            }
        },
        onExitState() {
            if (state.isShuttingDown()) {
                // Per the interface, this method is called when the manager is leaving
                // the state. We maintain this connection in the session manager until
                // all streams have finished, so we do not detach event listeners here.
                return;
            }
            cleanup();
            this.onError = undefined;
            this.onClose = undefined;
        },
    };
    // start or restart the ping interval
    function resetPingInterval() {
        stopPingInterval();
        if (streamCount > 0 || options.pingIdleConnection) {
            pingIntervalId = safeSetTimeout(onPingInterval, options.pingIntervalMs);
        }
    }
    function stopPingInterval() {
        clearTimeout(pingIntervalId);
        clearTimeout(pingTimeoutId);
    }
    function onPingInterval() {
        commonPing(resetPingInterval);
    }
    function commonPing(onSuccess) {
        clearTimeout(pingTimeoutId);
        pingTimeoutId = safeSetTimeout(() => {
            conn.destroy(new ConnectError("PING timed out", Code.Unavailable), http2.constants.NGHTTP2_CANCEL);
        }, options.pingTimeoutMs);
        conn.ping((err, duration) => {
            clearTimeout(pingTimeoutId);
            if (err !== null) {
                // We will receive an ERR_HTTP2_PING_CANCEL here if we destroy the
                // connection with a pending ping.
                // We might also see other errors, but they should be picked up by the
                // "error" event listener.
                return;
            }
            if (duration > options.pingTimeoutMs) {
                // setTimeout is not precise, and HTTP/2 pings take less than 1ms in
                // tests.
                conn.destroy(new ConnectError("PING timed out", Code.Unavailable), http2.constants.NGHTTP2_CANCEL);
                return;
            }
            lastAliveAt = Date.now();
            onSuccess();
        });
    }
    function stopIdleTimeout() {
        clearTimeout(idleTimeoutId);
    }
    function resetIdleTimeout() {
        idleTimeoutId = safeSetTimeout(onIdleTimeout, options.idleConnectionTimeoutMs);
    }
    function onIdleTimeout() {
        conn.close();
        onClose(); // trigger a state change right away, so we are not open to races
    }
    function onGoaway(errorCode, lastStreamID, opaqueData) {
        receivedGoAway = true;
        const tooManyPingsAscii = Buffer.from("too_many_pings", "ascii");
        if (errorCode === http2.constants.NGHTTP2_ENHANCE_YOUR_CALM &&
            opaqueData != null &&
            opaqueData.equals(tooManyPingsAscii)) {
            // double pingIntervalMs, following the last paragraph of https://github.com/grpc/proposal/blob/0ba0c1905050525f9b0aee46f3f23c8e1e515489/A8-client-side-keepalive.md#basic-keepalive
            options.pingIntervalMs = options.pingIntervalMs * 2;
            receivedGoAwayEnhanceYourCalmTooManyPings = true;
        }
        if (errorCode === http2.constants.NGHTTP2_NO_ERROR && streamCount == 0) {
            // Node.js v16 closes the connection on its own when it receives a GOAWAY
            // frame and there are no open streams (emitting a "close" event and
            // destroying the session), but later versions do not.
            // Calling close() ourselves is ineffective here - it appears that the
            // method is already being called, see https://github.com/nodejs/node/blob/198affc63973805ce5102d246f6b7822be57f5fc/lib/internal/http2/core.js#L681
            conn.destroy(new ConnectError("received GOAWAY without any open streams", Code.Canceled), http2.constants.NGHTTP2_NO_ERROR);
        }
    }
    function onClose() {
        var _a;
        cleanup();
        (_a = state.onClose) === null || _a === void 0 ? void 0 : _a.call(state);
    }
    function onError(err) {
        var _a, _b;
        cleanup();
        if (receivedGoAwayEnhanceYourCalmTooManyPings) {
            // We cannot prevent node from destroying session and streams with its own
            // error that does not carry debug data, but at least we can wrap the error
            // we surface on the manager.
            const ce = new ConnectError(`http/2 connection closed with error code ENHANCE_YOUR_CALM (0x${http2.constants.NGHTTP2_ENHANCE_YOUR_CALM.toString(16)}), too_many_pings, doubled the interval`, Code.ResourceExhausted);
            (_a = state.onError) === null || _a === void 0 ? void 0 : _a.call(state, ce);
        }
        else {
            (_b = state.onError) === null || _b === void 0 ? void 0 : _b.call(state, connectErrorFromNodeReason(err));
        }
    }
    function cleanup() {
        stopPingInterval();
        stopIdleTimeout();
        conn.off("error", onError);
        conn.off("close", onClose);
        conn.off("goaway", onGoaway);
    }
    conn.on("error", onError);
    conn.on("close", onClose);
    conn.on("goaway", onGoaway);
    return state;
}
/**
 * setTimeout(), but simply ignores values larger than the maximum supported
 * value (signed 32-bit integer) instead of calling the callback right away,
 * and does not block Node.js from exiting.
 */
function safeSetTimeout(callback, ms) {
    if (ms > 0x7fffffff) {
        return;
    }
    return setTimeout(callback, ms).unref();
}
function assertSessionOpen(conn) {
    if (conn.connecting) {
        throw new ConnectError("expected open session, but it is connecting", Code.Internal);
    }
    if (conn.destroyed) {
        throw new ConnectError("expected open session, but it is destroyed", Code.Internal);
    }
    if (conn.closed) {
        throw new ConnectError("expected open session, but it is closed", Code.Internal);
    }
}

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Create a universal client function, a minimal abstraction of an HTTP client,
 * using the Node.js `http`, `https`, or `http2` module.
 *
 * @private Internal code, does not follow semantic versioning.
 */
function createNodeHttpClient(options) {
    var _a;
    if (options.httpVersion == "1.1") {
        return createNodeHttp1Client(options.nodeOptions);
    }
    const sessionProvider = (_a = options.sessionProvider) !== null && _a !== void 0 ? _a : ((url) => new Http2SessionManager(url));
    return createNodeHttp2Client(sessionProvider);
}
/**
 * Create an HTTP client using the Node.js `http` or `https` package.
 *
 * The HTTP client is a simple function conforming to the type UniversalClientFn.
 * It takes an UniversalClientRequest as an argument, and returns a promise for
 * an UniversalClientResponse.
 */
function createNodeHttp1Client(httpOptions) {
    return async function request(req) {
        const sentinel = createSentinel(req.signal);
        return new Promise((resolve, reject) => {
            sentinel.onError((e) => {
                reject(e);
            });
            h1Request(sentinel, req.url, Object.assign(Object.assign({}, httpOptions), { headers: webHeaderToNodeHeaders(req.header, httpOptions === null || httpOptions === void 0 ? void 0 : httpOptions.headers), method: req.method }), (request) => {
                void sinkRequest(req, request, sentinel);
                request.on("response", (response) => {
                    var _a;
                    response.on("error", sentinel.error);
                    sentinel.onError((reason) => response.destroy(reason));
                    const trailer = new Headers();
                    resolve({
                        status: (_a = response.statusCode) !== null && _a !== void 0 ? _a : 0,
                        header: nodeHeaderToWebHeader(response.headers),
                        body: h1ResponseIterable(sentinel, response, trailer),
                        trailer,
                    });
                });
            });
        });
    };
}
/**
 * Create an HTTP client using the Node.js `http2` package.
 *
 * The HTTP client is a simple function conforming to the type UniversalClientFn.
 * It takes an UniversalClientRequest as an argument, and returns a promise for
 * an UniversalClientResponse.
 */
function createNodeHttp2Client(sessionProvider) {
    return function request(req) {
        const sentinel = createSentinel(req.signal);
        const sessionManager = sessionProvider(req.url);
        return new Promise((resolve, reject) => {
            sentinel.onError((e) => {
                reject(e);
            });
            h2Request(sentinel, sessionManager, req.url, req.method, webHeaderToNodeHeaders(req.header), {}, (stream) => {
                void sinkRequest(req, stream, sentinel);
                stream.on("response", (headers) => {
                    var _a;
                    const response = {
                        status: (_a = headers[":status"]) !== null && _a !== void 0 ? _a : 0,
                        header: nodeHeaderToWebHeader(headers),
                        body: h2ResponseIterable(sentinel, stream, sessionManager),
                        trailer: h2ResponseTrailer(stream),
                    };
                    resolve(response);
                });
            });
        });
    };
}
function h1Request(sentinel, url, options, onRequest) {
    let request;
    if (new URL(url).protocol.startsWith("https")) {
        request = https.request(url, options);
    }
    else {
        request = http.request(url, options);
    }
    sentinel.onError((reason) => request.destroy(reason));
    // Node.js will only send headers with the first request body byte by default.
    // We force it to send headers right away for consistent behavior between
    // HTTP/1.1 and HTTP/2.0 clients.
    request.flushHeaders();
    request.on("error", sentinel.error);
    request.on("socket", function onRequestSocket(socket) {
        function onSocketConnect() {
            socket.off("connect", onSocketConnect);
            onRequest(request);
        }
        // If readyState is open, then socket is already open due to keepAlive, so
        // the 'connect' event will never fire so call onRequest explicitly
        if (socket.readyState === "open") {
            onRequest(request);
        }
        else {
            socket.on("connect", onSocketConnect);
        }
    });
}
function h1ResponseIterable(sentinel, response, trailer) {
    const inner = response[Symbol.asyncIterator]();
    return {
        [Symbol.asyncIterator]() {
            return {
                async next() {
                    const r = await sentinel.race(inner.next());
                    if (r.done === true) {
                        nodeHeaderToWebHeader(response.trailers).forEach((value, key) => {
                            trailer.set(key, value);
                        });
                        sentinel.close();
                    }
                    return r;
                },
                throw(e) {
                    sentinel.error(e);
                    throw e;
                },
            };
        },
    };
}
function h2Request(sentinel, sm, url, method, headers, options, onStream) {
    const requestUrl = new URL(url);
    if (requestUrl.origin !== sm.authority) {
        const message = `cannot make a request to ${requestUrl.origin}: the http2 session is connected to ${sm.authority}`;
        sentinel.error(new ConnectError(message, Code.Internal));
        return;
    }
    sm.request(method, requestUrl.pathname + requestUrl.search, headers, {}).then((stream) => {
        sentinel.onError((reason) => {
            if (stream.closed) {
                return;
            }
            // Node.js http2 streams that are aborted via an AbortSignal close with
            // an RST_STREAM with code INTERNAL_ERROR.
            // To comply with the mapping between gRPC and HTTP/2 codes, we need to
            // close with code CANCEL.
            // See https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-HTTP2.md#errors
            // See https://www.rfc-editor.org/rfc/rfc7540#section-7
            const rstCode = reason.code == Code.Canceled ? H2Code.CANCEL : H2Code.INTERNAL_ERROR;
            return new Promise((resolve) => stream.close(rstCode, resolve));
        });
        stream.on("error", function h2StreamError(e) {
            if (stream.writableEnded &&
                unwrapNodeErrorChain(e)
                    .map(getNodeErrorProps)
                    .some((p) => p.code == "ERR_STREAM_WRITE_AFTER_END")) {
                return;
            }
            sentinel.error(e);
        });
        stream.on("close", function h2StreamClose() {
            const err = connectErrorFromH2ResetCode(stream.rstCode);
            if (err) {
                sentinel.error(err);
            }
        });
        onStream(stream);
    }, (reason) => {
        sentinel.error(reason);
    });
}
function h2ResponseTrailer(response) {
    const trailer = new Headers();
    response.on("trailers", (args) => {
        nodeHeaderToWebHeader(args).forEach((value, key) => {
            trailer.set(key, value);
        });
    });
    return trailer;
}
function h2ResponseIterable(sentinel, response, sm) {
    const inner = response[Symbol.asyncIterator]();
    return {
        [Symbol.asyncIterator]() {
            return {
                async next() {
                    const r = await sentinel.race(inner.next());
                    if (r.done === true) {
                        sentinel.close();
                    }
                    sm === null || sm === void 0 ? void 0 : sm.notifyResponseByteRead(response);
                    return r;
                },
                throw(e) {
                    sentinel.error(e);
                    throw e;
                },
            };
        },
    };
}
async function sinkRequest(request, nodeRequest, sentinel) {
    if (request.body === undefined) {
        await new Promise((resolve) => nodeRequest.end(resolve));
        return;
    }
    const it = request.body[Symbol.asyncIterator]();
    return new Promise((resolve) => {
        writeNext();
        function writeNext() {
            if (sentinel.isClosed()) {
                return;
            }
            it.next().then((r) => {
                if (r.done === true) {
                    nodeRequest.end(resolve);
                    return;
                }
                nodeRequest.write(r.value, "binary", (e) => {
                    if (e === null || e === undefined) {
                        writeNext();
                        return;
                    }
                    if (it.throw !== undefined) {
                        it.throw(connectErrorFromNodeReason(e)).catch(() => {
                            //
                        });
                    }
                    // If the server responds and closes the connection before the client has written the entire response
                    // body, we get an ERR_STREAM_WRITE_AFTER_END error code from Node.js here.
                    // We do want to notify the iterable of the error condition, but we do not want to reject our sentinel,
                    // because that would also affect the reading side.
                    if (nodeRequest.writableEnded &&
                        unwrapNodeErrorChain(e)
                            .map(getNodeErrorProps)
                            .some((p) => p.code == "ERR_STREAM_WRITE_AFTER_END")) {
                        return;
                    }
                    sentinel.error(e);
                });
            }, (e) => {
                sentinel.error(e);
            });
        }
    });
}
function createSentinel(signal) {
    let rejectRace;
    let closed = false;
    let closedError = undefined;
    let onErrorListeners = [];
    const sentinel = {
        error(error) {
            if (closed) {
                return;
            }
            closed = true;
            closedError =
                error instanceof ConnectError
                    ? error
                    : connectErrorFromNodeReason(error);
            rejectRace === null || rejectRace === void 0 ? void 0 : rejectRace(closedError);
            for (const onRejected of onErrorListeners) {
                onRejected(closedError);
            }
            cleanup();
        },
        close() {
            if (closed) {
                return;
            }
            closed = true;
            if (rejectRace) {
                rejectRace(new ConnectError("sentinel completed early", Code.Internal));
            }
            cleanup();
        },
        isClosed() {
            return closed;
        },
        onError(onError) {
            if (closed) {
                if (closedError !== undefined) {
                    onError(closedError);
                }
            }
            else {
                onErrorListeners.push(onError);
            }
        },
        race(promise) {
            let resolveRace;
            const race = new Promise((resolve, reject) => {
                resolveRace = resolve;
                rejectRace = reject;
            });
            promise.then((value) => {
                resolveRace === null || resolveRace === void 0 ? void 0 : resolveRace(value);
            }, (reason) => {
                rejectRace === null || rejectRace === void 0 ? void 0 : rejectRace(reason);
            });
            if (closed) {
                rejectRace === null || rejectRace === void 0 ? void 0 : rejectRace(closedError !== null && closedError !== void 0 ? closedError : new ConnectError("sentinel completed early", Code.Internal));
            }
            return race;
        },
    };
    function cleanup() {
        if (signal) {
            signal.removeEventListener("abort", onSignalAbort);
        }
        onErrorListeners = [];
        rejectRace = undefined;
    }
    function onSignalAbort() {
        sentinel.error(getAbortSignalReason(this));
    }
    if (signal) {
        signal.addEventListener("abort", onSignalAbort);
        if (signal.aborted) {
            sentinel.error(getAbortSignalReason(signal));
        }
    }
    return sentinel;
}

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Asserts that the options are within sane limits, and returns default values
 * where no value is provided.
 *
 * @private Internal code, does not follow semantic versioning.
 */
function validateNodeTransportOptions(options) {
    var _a, _b, _c, _d;
    let httpClient;
    if (options.httpVersion == "2") {
        let sessionManager;
        if (options.sessionManager) {
            sessionManager = options.sessionManager;
        }
        else {
            sessionManager = new Http2SessionManager(options.baseUrl, {
                pingIntervalMs: options.pingIntervalMs,
                pingIdleConnection: options.pingIdleConnection,
                pingTimeoutMs: options.pingTimeoutMs,
                idleConnectionTimeoutMs: options.idleConnectionTimeoutMs,
            }, options.nodeOptions);
        }
        httpClient = createNodeHttpClient({
            httpVersion: "2",
            sessionProvider: () => sessionManager,
        });
    }
    else {
        httpClient = createNodeHttpClient({
            httpVersion: "1.1",
            nodeOptions: options.nodeOptions,
        });
    }
    return Object.assign(Object.assign(Object.assign({}, options), { httpClient, useBinaryFormat: (_a = options.useBinaryFormat) !== null && _a !== void 0 ? _a : true, interceptors: (_b = options.interceptors) !== null && _b !== void 0 ? _b : [], sendCompression: (_c = options.sendCompression) !== null && _c !== void 0 ? _c : null, acceptCompression: (_d = options.acceptCompression) !== null && _d !== void 0 ? _d : [
            compressionGzip,
            compressionBrotli,
        ] }), validateReadWriteMaxBytes(options.readMaxBytes, options.writeMaxBytes, options.compressMinBytes));
}

// Copyright 2021-2025 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/**
 * Create a Transport for the gRPC protocol using the Node.js `http2` module.
 */
function createGrpcTransport(options) {
    return createTransport(validateNodeTransportOptions(Object.assign(Object.assign({}, options), { httpVersion: "2" })));
}

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0


/**
 * Describes the file agntcy/dir/core/v1/record.proto.
 */
const file_agntcy_dir_core_v1_record = /*@__PURE__*/
  fileDesc("Ch9hZ250Y3kvZGlyL2NvcmUvdjEvcmVjb3JkLnByb3RvEhJhZ250Y3kuZGlyLmNvcmUudjEiGAoJUmVjb3JkUmVmEgsKA2NpZBgBIAEoCSI8Cg5OYW1lZFJlY29yZFJlZhIMCgRuYW1lGAEgASgJEg8KB3ZlcnNpb24YAiABKAkSCwoDY2lkGAMgASgJIr8BCgpSZWNvcmRNZXRhEgsKA2NpZBgBIAEoCRJECgthbm5vdGF0aW9ucxgCIAMoCzIvLmFnbnRjeS5kaXIuY29yZS52MS5SZWNvcmRNZXRhLkFubm90YXRpb25zRW50cnkSFgoOc2NoZW1hX3ZlcnNpb24YAyABKAkSEgoKY3JlYXRlZF9hdBgEIAEoCRoyChBBbm5vdGF0aW9uc0VudHJ5EgsKA2tleRgBIAEoCRINCgV2YWx1ZRgCIAEoCToCOAEiLwoGUmVjb3JkEiUKBGRhdGEYASABKAsyFy5nb29nbGUucHJvdG9idWYuU3RydWN0IooCCg5SZWNvcmRSZWZlcnJlchIMCgR0eXBlGAEgASgJEjEKCnJlY29yZF9yZWYYAiABKAsyHS5hZ250Y3kuZGlyLmNvcmUudjEuUmVjb3JkUmVmEkgKC2Fubm90YXRpb25zGAMgAygLMjMuYWdudGN5LmRpci5jb3JlLnYxLlJlY29yZFJlZmVycmVyLkFubm90YXRpb25zRW50cnkSEgoKY3JlYXRlZF9hdBgEIAEoCRIlCgRkYXRhGAUgASgLMhcuZ29vZ2xlLnByb3RvYnVmLlN0cnVjdBoyChBBbm5vdGF0aW9uc0VudHJ5EgsKA2tleRgBIAEoCRINCgV2YWx1ZRgCIAEoCToCOAFCswEKFmNvbS5hZ250Y3kuZGlyLmNvcmUudjFCC1JlY29yZFByb3RvUAFaIWdpdGh1Yi5jb20vYWdudGN5L2Rpci9hcGkvY29yZS92MaICA0FEQ6oCEkFnbnRjeS5EaXIuQ29yZS5WMcoCEkFnbnRjeVxEaXJcQ29yZVxWMeICHkFnbnRjeVxEaXJcQ29yZVxWMVxHUEJNZXRhZGF0YeoCFUFnbnRjeTo6RGlyOjpDb3JlOjpWMWIGcHJvdG8z", [file_google_protobuf_struct]);

/**
 * Describes the message agntcy.dir.core.v1.RecordRef.
 * Use `create(RecordRefSchema)` to create a new message.
 */
const RecordRefSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_core_v1_record, 0);

/**
 * Describes the message agntcy.dir.core.v1.NamedRecordRef.
 * Use `create(NamedRecordRefSchema)` to create a new message.
 */
const NamedRecordRefSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_core_v1_record, 1);

/**
 * Describes the message agntcy.dir.core.v1.RecordMeta.
 * Use `create(RecordMetaSchema)` to create a new message.
 */
const RecordMetaSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_core_v1_record, 2);

/**
 * Describes the message agntcy.dir.core.v1.Record.
 * Use `create(RecordSchema)` to create a new message.
 */
const RecordSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_core_v1_record, 3);

/**
 * Describes the message agntcy.dir.core.v1.RecordReferrer.
 * Use `create(RecordReferrerSchema)` to create a new message.
 */
const RecordReferrerSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_core_v1_record, 4);

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

var core_v1 = /*#__PURE__*/Object.freeze({
    __proto__: null,
    NamedRecordRefSchema: NamedRecordRefSchema,
    RecordMetaSchema: RecordMetaSchema,
    RecordRefSchema: RecordRefSchema,
    RecordReferrerSchema: RecordReferrerSchema,
    RecordSchema: RecordSchema,
    file_agntcy_dir_core_v1_record: file_agntcy_dir_core_v1_record
});

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0


/**
 * Describes the file agntcy/dir/naming/v1/name_verification.proto.
 */
const file_agntcy_dir_naming_v1_name_verification = /*@__PURE__*/
  fileDesc("CixhZ250Y3kvZGlyL25hbWluZy92MS9uYW1lX3ZlcmlmaWNhdGlvbi5wcm90bxIUYWdudGN5LmRpci5uYW1pbmcudjEiUgoMVmVyaWZpY2F0aW9uEjoKBmRvbWFpbhgBIAEoCzIoLmFnbnRjeS5kaXIubmFtaW5nLnYxLkRvbWFpblZlcmlmaWNhdGlvbkgAQgYKBGluZm8idQoSRG9tYWluVmVyaWZpY2F0aW9uEg4KBmRvbWFpbhgBIAEoCRIOCgZtZXRob2QYAiABKAkSDgoGa2V5X2lkGAMgASgJEi8KC3ZlcmlmaWVkX2F0GAQgASgLMhouZ29vZ2xlLnByb3RvYnVmLlRpbWVzdGFtcELJAQoYY29tLmFnbnRjeS5kaXIubmFtaW5nLnYxQhVOYW1lVmVyaWZpY2F0aW9uUHJvdG9QAVojZ2l0aHViLmNvbS9hZ250Y3kvZGlyL2FwaS9uYW1pbmcvdjGiAgNBRE6qAhRBZ250Y3kuRGlyLk5hbWluZy5WMcoCFEFnbnRjeVxEaXJcTmFtaW5nXFYx4gIgQWdudGN5XERpclxOYW1pbmdcVjFcR1BCTWV0YWRhdGHqAhdBZ250Y3k6OkRpcjo6TmFtaW5nOjpWMWIGcHJvdG8z", [file_google_protobuf_timestamp]);

/**
 * Describes the message agntcy.dir.naming.v1.Verification.
 * Use `create(VerificationSchema)` to create a new message.
 */
const VerificationSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_naming_v1_name_verification, 0);

/**
 * Describes the message agntcy.dir.naming.v1.DomainVerification.
 * Use `create(DomainVerificationSchema)` to create a new message.
 */
const DomainVerificationSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_naming_v1_name_verification, 1);

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0


/**
 * Describes the file agntcy/dir/naming/v1/naming_service.proto.
 */
const file_agntcy_dir_naming_v1_naming_service = /*@__PURE__*/
  fileDesc("CilhZ250Y3kvZGlyL25hbWluZy92MS9uYW1pbmdfc2VydmljZS5wcm90bxIUYWdudGN5LmRpci5uYW1pbmcudjEidAoaR2V0VmVyaWZpY2F0aW9uSW5mb1JlcXVlc3QSEAoDY2lkGAEgASgJSACIAQESEQoEbmFtZRgCIAEoCUgBiAEBEhQKB3ZlcnNpb24YAyABKAlIAogBAUIGCgRfY2lkQgcKBV9uYW1lQgoKCF92ZXJzaW9uIpcBChtHZXRWZXJpZmljYXRpb25JbmZvUmVzcG9uc2USEAoIdmVyaWZpZWQYASABKAgSOAoMdmVyaWZpY2F0aW9uGAIgASgLMiIuYWdudGN5LmRpci5uYW1pbmcudjEuVmVyaWZpY2F0aW9uEhoKDWVycm9yX21lc3NhZ2UYAyABKAlIAIgBAUIQCg5fZXJyb3JfbWVzc2FnZSJACg5SZXNvbHZlUmVxdWVzdBIMCgRuYW1lGAEgASgJEhQKB3ZlcnNpb24YAiABKAlIAIgBAUIKCghfdmVyc2lvbiJGCg9SZXNvbHZlUmVzcG9uc2USMwoHcmVjb3JkcxgBIAMoCzIiLmFnbnRjeS5kaXIuY29yZS52MS5OYW1lZFJlY29yZFJlZjLjAQoNTmFtaW5nU2VydmljZRJ6ChNHZXRWZXJpZmljYXRpb25JbmZvEjAuYWdudGN5LmRpci5uYW1pbmcudjEuR2V0VmVyaWZpY2F0aW9uSW5mb1JlcXVlc3QaMS5hZ250Y3kuZGlyLm5hbWluZy52MS5HZXRWZXJpZmljYXRpb25JbmZvUmVzcG9uc2USVgoHUmVzb2x2ZRIkLmFnbnRjeS5kaXIubmFtaW5nLnYxLlJlc29sdmVSZXF1ZXN0GiUuYWdudGN5LmRpci5uYW1pbmcudjEuUmVzb2x2ZVJlc3BvbnNlQsYBChhjb20uYWdudGN5LmRpci5uYW1pbmcudjFCEk5hbWluZ1NlcnZpY2VQcm90b1ABWiNnaXRodWIuY29tL2FnbnRjeS9kaXIvYXBpL25hbWluZy92MaICA0FETqoCFEFnbnRjeS5EaXIuTmFtaW5nLlYxygIUQWdudGN5XERpclxOYW1pbmdcVjHiAiBBZ250Y3lcRGlyXE5hbWluZ1xWMVxHUEJNZXRhZGF0YeoCF0FnbnRjeTo6RGlyOjpOYW1pbmc6OlYxYgZwcm90bzM", [file_agntcy_dir_core_v1_record, file_agntcy_dir_naming_v1_name_verification]);

/**
 * Describes the message agntcy.dir.naming.v1.GetVerificationInfoRequest.
 * Use `create(GetVerificationInfoRequestSchema)` to create a new message.
 */
const GetVerificationInfoRequestSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_naming_v1_naming_service, 0);

/**
 * Describes the message agntcy.dir.naming.v1.GetVerificationInfoResponse.
 * Use `create(GetVerificationInfoResponseSchema)` to create a new message.
 */
const GetVerificationInfoResponseSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_naming_v1_naming_service, 1);

/**
 * Describes the message agntcy.dir.naming.v1.ResolveRequest.
 * Use `create(ResolveRequestSchema)` to create a new message.
 */
const ResolveRequestSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_naming_v1_naming_service, 2);

/**
 * Describes the message agntcy.dir.naming.v1.ResolveResponse.
 * Use `create(ResolveResponseSchema)` to create a new message.
 */
const ResolveResponseSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_naming_v1_naming_service, 3);

/**
 * NamingService provides methods for name resolution and verification.
 * Note: Verification is performed automatically by the backend scheduler
 * for signed records with verifiable names (http://, https:// prefixes).
 *
 * @generated from service agntcy.dir.naming.v1.NamingService
 */
const NamingService = /*@__PURE__*/
  serviceDesc(file_agntcy_dir_naming_v1_naming_service, 0);

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0
// Note: domain_verification_pb is already re-exported by name_verification_pb

var naming_v1 = /*#__PURE__*/Object.freeze({
    __proto__: null,
    DomainVerificationSchema: DomainVerificationSchema,
    GetVerificationInfoRequestSchema: GetVerificationInfoRequestSchema,
    GetVerificationInfoResponseSchema: GetVerificationInfoResponseSchema,
    NamingService: NamingService,
    ResolveRequestSchema: ResolveRequestSchema,
    ResolveResponseSchema: ResolveResponseSchema,
    VerificationSchema: VerificationSchema,
    file_agntcy_dir_naming_v1_name_verification: file_agntcy_dir_naming_v1_name_verification,
    file_agntcy_dir_naming_v1_naming_service: file_agntcy_dir_naming_v1_naming_service
});

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0


/**
 * Describes the file agntcy/dir/routing/v1/peer.proto.
 */
const file_agntcy_dir_routing_v1_peer = /*@__PURE__*/
  fileDesc("CiBhZ250Y3kvZGlyL3JvdXRpbmcvdjEvcGVlci5wcm90bxIVYWdudGN5LmRpci5yb3V0aW5nLnYxItcBCgRQZWVyEgoKAmlkGAEgASgJEg0KBWFkZHJzGAIgAygJEkEKC2Fubm90YXRpb25zGAMgAygLMiwuYWdudGN5LmRpci5yb3V0aW5nLnYxLlBlZXIuQW5ub3RhdGlvbnNFbnRyeRI9Cgpjb25uZWN0aW9uGAQgASgOMikuYWdudGN5LmRpci5yb3V0aW5nLnYxLlBlZXJDb25uZWN0aW9uVHlwZRoyChBBbm5vdGF0aW9uc0VudHJ5EgsKA2tleRgBIAEoCRINCgV2YWx1ZRgCIAEoCToCOAEqrwEKElBlZXJDb25uZWN0aW9uVHlwZRImCiJQRUVSX0NPTk5FQ1RJT05fVFlQRV9OT1RfQ09OTkVDVEVEEAASIgoeUEVFUl9DT05ORUNUSU9OX1RZUEVfQ09OTkVDVEVEEAESJAogUEVFUl9DT05ORUNUSU9OX1RZUEVfQ0FOX0NPTk5FQ1QQAhInCiNQRUVSX0NPTk5FQ1RJT05fVFlQRV9DQU5OT1RfQ09OTkVDVBADQsMBChljb20uYWdudGN5LmRpci5yb3V0aW5nLnYxQglQZWVyUHJvdG9QAVokZ2l0aHViLmNvbS9hZ250Y3kvZGlyL2FwaS9yb3V0aW5nL3YxogIDQURSqgIVQWdudGN5LkRpci5Sb3V0aW5nLlYxygIVQWdudGN5XERpclxSb3V0aW5nXFYx4gIhQWdudGN5XERpclxSb3V0aW5nXFYxXEdQQk1ldGFkYXRh6gIYQWdudGN5OjpEaXI6OlJvdXRpbmc6OlYxYgZwcm90bzM");

/**
 * Describes the message agntcy.dir.routing.v1.Peer.
 * Use `create(PeerSchema)` to create a new message.
 */
const PeerSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_routing_v1_peer, 0);

/**
 * Describes the enum agntcy.dir.routing.v1.PeerConnectionType.
 */
const PeerConnectionTypeSchema = /*@__PURE__*/
  enumDesc(file_agntcy_dir_routing_v1_peer, 0);

/**
 * @generated from enum agntcy.dir.routing.v1.PeerConnectionType
 */
const PeerConnectionType = /*@__PURE__*/
  tsEnum(PeerConnectionTypeSchema);

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0


/**
 * Describes the file agntcy/dir/routing/v1/record_query.proto.
 */
const file_agntcy_dir_routing_v1_record_query = /*@__PURE__*/
  fileDesc("CihhZ250Y3kvZGlyL3JvdXRpbmcvdjEvcmVjb3JkX3F1ZXJ5LnByb3RvEhVhZ250Y3kuZGlyLnJvdXRpbmcudjEiUgoLUmVjb3JkUXVlcnkSNAoEdHlwZRgBIAEoDjImLmFnbnRjeS5kaXIucm91dGluZy52MS5SZWNvcmRRdWVyeVR5cGUSDQoFdmFsdWUYAiABKAkqrAEKD1JlY29yZFF1ZXJ5VHlwZRIhCh1SRUNPUkRfUVVFUllfVFlQRV9VTlNQRUNJRklFRBAAEhsKF1JFQ09SRF9RVUVSWV9UWVBFX1NLSUxMEAESHQoZUkVDT1JEX1FVRVJZX1RZUEVfTE9DQVRPUhACEhwKGFJFQ09SRF9RVUVSWV9UWVBFX0RPTUFJThADEhwKGFJFQ09SRF9RVUVSWV9UWVBFX01PRFVMRRAEQsoBChljb20uYWdudGN5LmRpci5yb3V0aW5nLnYxQhBSZWNvcmRRdWVyeVByb3RvUAFaJGdpdGh1Yi5jb20vYWdudGN5L2Rpci9hcGkvcm91dGluZy92MaICA0FEUqoCFUFnbnRjeS5EaXIuUm91dGluZy5WMcoCFUFnbnRjeVxEaXJcUm91dGluZ1xWMeICIUFnbnRjeVxEaXJcUm91dGluZ1xWMVxHUEJNZXRhZGF0YeoCGEFnbnRjeTo6RGlyOjpSb3V0aW5nOjpWMWIGcHJvdG8z");

/**
 * Describes the message agntcy.dir.routing.v1.RecordQuery.
 * Use `create(RecordQuerySchema)` to create a new message.
 */
const RecordQuerySchema$1 = /*@__PURE__*/
  messageDesc(file_agntcy_dir_routing_v1_record_query, 0);

/**
 * Describes the enum agntcy.dir.routing.v1.RecordQueryType.
 */
const RecordQueryTypeSchema$1 = /*@__PURE__*/
  enumDesc(file_agntcy_dir_routing_v1_record_query, 0);

/**
 * Defines a list of supported record query types.
 *
 * @generated from enum agntcy.dir.routing.v1.RecordQueryType
 */
const RecordQueryType$1 = /*@__PURE__*/
  tsEnum(RecordQueryTypeSchema$1);

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0


/**
 * Describes the file agntcy/dir/search/v1/record_query.proto.
 */
const file_agntcy_dir_search_v1_record_query = /*@__PURE__*/
  fileDesc("CidhZ250Y3kvZGlyL3NlYXJjaC92MS9yZWNvcmRfcXVlcnkucHJvdG8SFGFnbnRjeS5kaXIuc2VhcmNoLnYxIlEKC1JlY29yZFF1ZXJ5EjMKBHR5cGUYASABKA4yJS5hZ250Y3kuZGlyLnNlYXJjaC52MS5SZWNvcmRRdWVyeVR5cGUSDQoFdmFsdWUYAiABKAkq3gMKD1JlY29yZFF1ZXJ5VHlwZRIhCh1SRUNPUkRfUVVFUllfVFlQRV9VTlNQRUNJRklFRBAAEhoKFlJFQ09SRF9RVUVSWV9UWVBFX05BTUUQARIdChlSRUNPUkRfUVVFUllfVFlQRV9WRVJTSU9OEAISHgoaUkVDT1JEX1FVRVJZX1RZUEVfU0tJTExfSUQQAxIgChxSRUNPUkRfUVVFUllfVFlQRV9TS0lMTF9OQU1FEAQSHQoZUkVDT1JEX1FVRVJZX1RZUEVfTE9DQVRPUhAFEiEKHVJFQ09SRF9RVUVSWV9UWVBFX01PRFVMRV9OQU1FEAYSHwobUkVDT1JEX1FVRVJZX1RZUEVfRE9NQUlOX0lEEAcSIQodUkVDT1JEX1FVRVJZX1RZUEVfRE9NQUlOX05BTUUQCBIgChxSRUNPUkRfUVVFUllfVFlQRV9DUkVBVEVEX0FUEAkSHAoYUkVDT1JEX1FVRVJZX1RZUEVfQVVUSE9SEAoSJAogUkVDT1JEX1FVRVJZX1RZUEVfU0NIRU1BX1ZFUlNJT04QCxIfChtSRUNPUkRfUVVFUllfVFlQRV9NT0RVTEVfSUQQDBIeChpSRUNPUkRfUVVFUllfVFlQRV9WRVJJRklFRBANQsQBChhjb20uYWdudGN5LmRpci5zZWFyY2gudjFCEFJlY29yZFF1ZXJ5UHJvdG9QAVojZ2l0aHViLmNvbS9hZ250Y3kvZGlyL2FwaS9zZWFyY2gvdjGiAgNBRFOqAhRBZ250Y3kuRGlyLlNlYXJjaC5WMcoCFEFnbnRjeVxEaXJcU2VhcmNoXFYx4gIgQWdudGN5XERpclxTZWFyY2hcVjFcR1BCTWV0YWRhdGHqAhdBZ250Y3k6OkRpcjo6U2VhcmNoOjpWMWIGcHJvdG8z");

/**
 * Describes the message agntcy.dir.search.v1.RecordQuery.
 * Use `create(RecordQuerySchema)` to create a new message.
 */
const RecordQuerySchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_search_v1_record_query, 0);

/**
 * Describes the enum agntcy.dir.search.v1.RecordQueryType.
 */
const RecordQueryTypeSchema = /*@__PURE__*/
  enumDesc(file_agntcy_dir_search_v1_record_query, 0);

/**
 * Defines a list of supported record query types.
 *
 * @generated from enum agntcy.dir.search.v1.RecordQueryType
 */
const RecordQueryType = /*@__PURE__*/
  tsEnum(RecordQueryTypeSchema);

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0


/**
 * Describes the file agntcy/dir/routing/v1/routing_service.proto.
 */
const file_agntcy_dir_routing_v1_routing_service = /*@__PURE__*/
  fileDesc("CithZ250Y3kvZGlyL3JvdXRpbmcvdjEvcm91dGluZ19zZXJ2aWNlLnByb3RvEhVhZ250Y3kuZGlyLnJvdXRpbmcudjEijgEKDlB1Ymxpc2hSZXF1ZXN0EjgKC3JlY29yZF9yZWZzGAEgASgLMiEuYWdudGN5LmRpci5yb3V0aW5nLnYxLlJlY29yZFJlZnNIABI3CgdxdWVyaWVzGAIgASgLMiQuYWdudGN5LmRpci5yb3V0aW5nLnYxLlJlY29yZFF1ZXJpZXNIAEIJCgdyZXF1ZXN0IpABChBVbnB1Ymxpc2hSZXF1ZXN0EjgKC3JlY29yZF9yZWZzGAEgASgLMiEuYWdudGN5LmRpci5yb3V0aW5nLnYxLlJlY29yZFJlZnNIABI3CgdxdWVyaWVzGAIgASgLMiQuYWdudGN5LmRpci5yb3V0aW5nLnYxLlJlY29yZFF1ZXJpZXNIAEIJCgdyZXF1ZXN0IjkKClJlY29yZFJlZnMSKwoEcmVmcxgBIAMoCzIdLmFnbnRjeS5kaXIuY29yZS52MS5SZWNvcmRSZWYiQwoNUmVjb3JkUXVlcmllcxIyCgdxdWVyaWVzGAEgAygLMiEuYWdudGN5LmRpci5zZWFyY2gudjEuUmVjb3JkUXVlcnkilAEKDVNlYXJjaFJlcXVlc3QSMwoHcXVlcmllcxgBIAMoCzIiLmFnbnRjeS5kaXIucm91dGluZy52MS5SZWNvcmRRdWVyeRIcCg9taW5fbWF0Y2hfc2NvcmUYAiABKA1IAIgBARISCgVsaW1pdBgDIAEoDUgBiAEBQhIKEF9taW5fbWF0Y2hfc2NvcmVCCAoGX2xpbWl0Ir4BCg5TZWFyY2hSZXNwb25zZRIxCgpyZWNvcmRfcmVmGAEgASgLMh0uYWdudGN5LmRpci5jb3JlLnYxLlJlY29yZFJlZhIpCgRwZWVyGAIgASgLMhsuYWdudGN5LmRpci5yb3V0aW5nLnYxLlBlZXISOQoNbWF0Y2hfcXVlcmllcxgDIAMoCzIiLmFnbnRjeS5kaXIucm91dGluZy52MS5SZWNvcmRRdWVyeRITCgttYXRjaF9zY29yZRgEIAEoDSJgCgtMaXN0UmVxdWVzdBIzCgdxdWVyaWVzGAEgAygLMiIuYWdudGN5LmRpci5yb3V0aW5nLnYxLlJlY29yZFF1ZXJ5EhIKBWxpbWl0GAIgASgNSACIAQFCCAoGX2xpbWl0IlEKDExpc3RSZXNwb25zZRIxCgpyZWNvcmRfcmVmGAEgASgLMh0uYWdudGN5LmRpci5jb3JlLnYxLlJlY29yZFJlZhIOCgZsYWJlbHMYAiADKAky1AIKDlJvdXRpbmdTZXJ2aWNlEkgKB1B1Ymxpc2gSJS5hZ250Y3kuZGlyLnJvdXRpbmcudjEuUHVibGlzaFJlcXVlc3QaFi5nb29nbGUucHJvdG9idWYuRW1wdHkSTAoJVW5wdWJsaXNoEicuYWdudGN5LmRpci5yb3V0aW5nLnYxLlVucHVibGlzaFJlcXVlc3QaFi5nb29nbGUucHJvdG9idWYuRW1wdHkSVwoGU2VhcmNoEiQuYWdudGN5LmRpci5yb3V0aW5nLnYxLlNlYXJjaFJlcXVlc3QaJS5hZ250Y3kuZGlyLnJvdXRpbmcudjEuU2VhcmNoUmVzcG9uc2UwARJRCgRMaXN0EiIuYWdudGN5LmRpci5yb3V0aW5nLnYxLkxpc3RSZXF1ZXN0GiMuYWdudGN5LmRpci5yb3V0aW5nLnYxLkxpc3RSZXNwb25zZTABQs0BChljb20uYWdudGN5LmRpci5yb3V0aW5nLnYxQhNSb3V0aW5nU2VydmljZVByb3RvUAFaJGdpdGh1Yi5jb20vYWdudGN5L2Rpci9hcGkvcm91dGluZy92MaICA0FEUqoCFUFnbnRjeS5EaXIuUm91dGluZy5WMcoCFUFnbnRjeVxEaXJcUm91dGluZ1xWMeICIUFnbnRjeVxEaXJcUm91dGluZ1xWMVxHUEJNZXRhZGF0YeoCGEFnbnRjeTo6RGlyOjpSb3V0aW5nOjpWMWIGcHJvdG8z", [file_agntcy_dir_core_v1_record, file_agntcy_dir_routing_v1_peer, file_agntcy_dir_routing_v1_record_query, file_agntcy_dir_search_v1_record_query, file_google_protobuf_empty]);

/**
 * Describes the message agntcy.dir.routing.v1.PublishRequest.
 * Use `create(PublishRequestSchema)` to create a new message.
 */
const PublishRequestSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_routing_v1_routing_service, 0);

/**
 * Describes the message agntcy.dir.routing.v1.UnpublishRequest.
 * Use `create(UnpublishRequestSchema)` to create a new message.
 */
const UnpublishRequestSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_routing_v1_routing_service, 1);

/**
 * Describes the message agntcy.dir.routing.v1.RecordRefs.
 * Use `create(RecordRefsSchema)` to create a new message.
 */
const RecordRefsSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_routing_v1_routing_service, 2);

/**
 * Describes the message agntcy.dir.routing.v1.RecordQueries.
 * Use `create(RecordQueriesSchema)` to create a new message.
 */
const RecordQueriesSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_routing_v1_routing_service, 3);

/**
 * Describes the message agntcy.dir.routing.v1.SearchRequest.
 * Use `create(SearchRequestSchema)` to create a new message.
 */
const SearchRequestSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_routing_v1_routing_service, 4);

/**
 * Describes the message agntcy.dir.routing.v1.SearchResponse.
 * Use `create(SearchResponseSchema)` to create a new message.
 */
const SearchResponseSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_routing_v1_routing_service, 5);

/**
 * Describes the message agntcy.dir.routing.v1.ListRequest.
 * Use `create(ListRequestSchema)` to create a new message.
 */
const ListRequestSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_routing_v1_routing_service, 6);

/**
 * Describes the message agntcy.dir.routing.v1.ListResponse.
 * Use `create(ListResponseSchema)` to create a new message.
 */
const ListResponseSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_routing_v1_routing_service, 7);

/**
 * Defines an interface for announcement and discovery
 * of records across interconnected network.
 *
 * Middleware should be used to control who can perform these RPCs.
 * Policies for the middleware can be handled via separate service.
 *
 * @generated from service agntcy.dir.routing.v1.RoutingService
 */
const RoutingService = /*@__PURE__*/
  serviceDesc(file_agntcy_dir_routing_v1_routing_service, 0);

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0


/**
 * Describes the file agntcy/dir/routing/v1/publication_service.proto.
 */
const file_agntcy_dir_routing_v1_publication_service = /*@__PURE__*/
  fileDesc("Ci9hZ250Y3kvZGlyL3JvdXRpbmcvdjEvcHVibGljYXRpb25fc2VydmljZS5wcm90bxIVYWdudGN5LmRpci5yb3V0aW5nLnYxIjMKGUNyZWF0ZVB1YmxpY2F0aW9uUmVzcG9uc2USFgoOcHVibGljYXRpb25faWQYASABKAkiVwoXTGlzdFB1YmxpY2F0aW9uc1JlcXVlc3QSEgoFbGltaXQYAiABKA1IAIgBARITCgZvZmZzZXQYAyABKA1IAYgBAUIICgZfbGltaXRCCQoHX29mZnNldCKYAQoUTGlzdFB1YmxpY2F0aW9uc0l0ZW0SFgoOcHVibGljYXRpb25faWQYASABKAkSOAoGc3RhdHVzGAIgASgOMiguYWdudGN5LmRpci5yb3V0aW5nLnYxLlB1YmxpY2F0aW9uU3RhdHVzEhQKDGNyZWF0ZWRfdGltZRgDIAEoCRIYChBsYXN0X3VwZGF0ZV90aW1lGAQgASgJIi8KFUdldFB1YmxpY2F0aW9uUmVxdWVzdBIWCg5wdWJsaWNhdGlvbl9pZBgBIAEoCSKaAQoWR2V0UHVibGljYXRpb25SZXNwb25zZRIWCg5wdWJsaWNhdGlvbl9pZBgBIAEoCRI4CgZzdGF0dXMYAiABKA4yKC5hZ250Y3kuZGlyLnJvdXRpbmcudjEuUHVibGljYXRpb25TdGF0dXMSFAoMY3JlYXRlZF90aW1lGAMgASgJEhgKEGxhc3RfdXBkYXRlX3RpbWUYBCABKAkqvAEKEVB1YmxpY2F0aW9uU3RhdHVzEiIKHlBVQkxJQ0FUSU9OX1NUQVRVU19VTlNQRUNJRklFRBAAEh4KGlBVQkxJQ0FUSU9OX1NUQVRVU19QRU5ESU5HEAESIgoeUFVCTElDQVRJT05fU1RBVFVTX0lOX1BST0dSRVNTEAISIAocUFVCTElDQVRJT05fU1RBVFVTX0NPTVBMRVRFRBADEh0KGVBVQkxJQ0FUSU9OX1NUQVRVU19GQUlMRUQQBDLkAgoSUHVibGljYXRpb25TZXJ2aWNlEmwKEUNyZWF0ZVB1YmxpY2F0aW9uEiUuYWdudGN5LmRpci5yb3V0aW5nLnYxLlB1Ymxpc2hSZXF1ZXN0GjAuYWdudGN5LmRpci5yb3V0aW5nLnYxLkNyZWF0ZVB1YmxpY2F0aW9uUmVzcG9uc2UScQoQTGlzdFB1YmxpY2F0aW9ucxIuLmFnbnRjeS5kaXIucm91dGluZy52MS5MaXN0UHVibGljYXRpb25zUmVxdWVzdBorLmFnbnRjeS5kaXIucm91dGluZy52MS5MaXN0UHVibGljYXRpb25zSXRlbTABEm0KDkdldFB1YmxpY2F0aW9uEiwuYWdudGN5LmRpci5yb3V0aW5nLnYxLkdldFB1YmxpY2F0aW9uUmVxdWVzdBotLmFnbnRjeS5kaXIucm91dGluZy52MS5HZXRQdWJsaWNhdGlvblJlc3BvbnNlQtEBChljb20uYWdudGN5LmRpci5yb3V0aW5nLnYxQhdQdWJsaWNhdGlvblNlcnZpY2VQcm90b1ABWiRnaXRodWIuY29tL2FnbnRjeS9kaXIvYXBpL3JvdXRpbmcvdjGiAgNBRFKqAhVBZ250Y3kuRGlyLlJvdXRpbmcuVjHKAhVBZ250Y3lcRGlyXFJvdXRpbmdcVjHiAiFBZ250Y3lcRGlyXFJvdXRpbmdcVjFcR1BCTWV0YWRhdGHqAhhBZ250Y3k6OkRpcjo6Um91dGluZzo6VjFiBnByb3RvMw", [file_agntcy_dir_routing_v1_routing_service]);

/**
 * Describes the message agntcy.dir.routing.v1.CreatePublicationResponse.
 * Use `create(CreatePublicationResponseSchema)` to create a new message.
 */
const CreatePublicationResponseSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_routing_v1_publication_service, 0);

/**
 * Describes the message agntcy.dir.routing.v1.ListPublicationsRequest.
 * Use `create(ListPublicationsRequestSchema)` to create a new message.
 */
const ListPublicationsRequestSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_routing_v1_publication_service, 1);

/**
 * Describes the message agntcy.dir.routing.v1.ListPublicationsItem.
 * Use `create(ListPublicationsItemSchema)` to create a new message.
 */
const ListPublicationsItemSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_routing_v1_publication_service, 2);

/**
 * Describes the message agntcy.dir.routing.v1.GetPublicationRequest.
 * Use `create(GetPublicationRequestSchema)` to create a new message.
 */
const GetPublicationRequestSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_routing_v1_publication_service, 3);

/**
 * Describes the message agntcy.dir.routing.v1.GetPublicationResponse.
 * Use `create(GetPublicationResponseSchema)` to create a new message.
 */
const GetPublicationResponseSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_routing_v1_publication_service, 4);

/**
 * Describes the enum agntcy.dir.routing.v1.PublicationStatus.
 */
const PublicationStatusSchema = /*@__PURE__*/
  enumDesc(file_agntcy_dir_routing_v1_publication_service, 0);

/**
 * PublicationStatus represents the current state of a publication request.
 * Publications progress from pending to processing to completed or failed states.
 *
 * @generated from enum agntcy.dir.routing.v1.PublicationStatus
 */
const PublicationStatus = /*@__PURE__*/
  tsEnum(PublicationStatusSchema);

/**
 * PublicationService manages publication requests for announcing records to the DHT.
 *
 * Publications are stored in the database and processed by a worker that runs every hour.
 * The publication workflow:
 * 1. Publications are created via routing's Publish RPC by specifying either a query, a list of CIDs, or all records
 * 2. Publication requests are added to the database
 * 3. PublicationWorker queries the data using the publication request from the database to get the list of CIDs to be published
 * 4. PublicationWorker announces the records with these CIDs to the DHT
 *
 * @generated from service agntcy.dir.routing.v1.PublicationService
 */
const PublicationService = /*@__PURE__*/
  serviceDesc(file_agntcy_dir_routing_v1_publication_service, 0);

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

var routing_v1 = /*#__PURE__*/Object.freeze({
    __proto__: null,
    CreatePublicationResponseSchema: CreatePublicationResponseSchema,
    GetPublicationRequestSchema: GetPublicationRequestSchema,
    GetPublicationResponseSchema: GetPublicationResponseSchema,
    ListPublicationsItemSchema: ListPublicationsItemSchema,
    ListPublicationsRequestSchema: ListPublicationsRequestSchema,
    ListRequestSchema: ListRequestSchema,
    ListResponseSchema: ListResponseSchema,
    PeerConnectionType: PeerConnectionType,
    PeerConnectionTypeSchema: PeerConnectionTypeSchema,
    PeerSchema: PeerSchema,
    PublicationService: PublicationService,
    PublicationStatus: PublicationStatus,
    PublicationStatusSchema: PublicationStatusSchema,
    PublishRequestSchema: PublishRequestSchema,
    RecordQueriesSchema: RecordQueriesSchema,
    RecordQuerySchema: RecordQuerySchema$1,
    RecordQueryType: RecordQueryType$1,
    RecordQueryTypeSchema: RecordQueryTypeSchema$1,
    RecordRefsSchema: RecordRefsSchema,
    RoutingService: RoutingService,
    SearchRequestSchema: SearchRequestSchema,
    SearchResponseSchema: SearchResponseSchema,
    UnpublishRequestSchema: UnpublishRequestSchema,
    file_agntcy_dir_routing_v1_peer: file_agntcy_dir_routing_v1_peer,
    file_agntcy_dir_routing_v1_publication_service: file_agntcy_dir_routing_v1_publication_service,
    file_agntcy_dir_routing_v1_record_query: file_agntcy_dir_routing_v1_record_query,
    file_agntcy_dir_routing_v1_routing_service: file_agntcy_dir_routing_v1_routing_service
});

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0


/**
 * Describes the file agntcy/dir/search/v1/search_service.proto.
 */
const file_agntcy_dir_search_v1_search_service = /*@__PURE__*/
  fileDesc("CilhZ250Y3kvZGlyL3NlYXJjaC92MS9zZWFyY2hfc2VydmljZS5wcm90bxIUYWdudGN5LmRpci5zZWFyY2gudjEihQEKEVNlYXJjaENJRHNSZXF1ZXN0EjIKB3F1ZXJpZXMYASADKAsyIS5hZ250Y3kuZGlyLnNlYXJjaC52MS5SZWNvcmRRdWVyeRISCgVsaW1pdBgCIAEoDUgAiAEBEhMKBm9mZnNldBgDIAEoDUgBiAEBQggKBl9saW1pdEIJCgdfb2Zmc2V0IogBChRTZWFyY2hSZWNvcmRzUmVxdWVzdBIyCgdxdWVyaWVzGAEgAygLMiEuYWdudGN5LmRpci5zZWFyY2gudjEuUmVjb3JkUXVlcnkSEgoFbGltaXQYAiABKA1IAIgBARITCgZvZmZzZXQYAyABKA1IAYgBAUIICgZfbGltaXRCCQoHX29mZnNldCIoChJTZWFyY2hDSURzUmVzcG9uc2USEgoKcmVjb3JkX2NpZBgBIAEoCSJDChVTZWFyY2hSZWNvcmRzUmVzcG9uc2USKgoGcmVjb3JkGAEgASgLMhouYWdudGN5LmRpci5jb3JlLnYxLlJlY29yZDLeAQoNU2VhcmNoU2VydmljZRJhCgpTZWFyY2hDSURzEicuYWdudGN5LmRpci5zZWFyY2gudjEuU2VhcmNoQ0lEc1JlcXVlc3QaKC5hZ250Y3kuZGlyLnNlYXJjaC52MS5TZWFyY2hDSURzUmVzcG9uc2UwARJqCg1TZWFyY2hSZWNvcmRzEiouYWdudGN5LmRpci5zZWFyY2gudjEuU2VhcmNoUmVjb3Jkc1JlcXVlc3QaKy5hZ250Y3kuZGlyLnNlYXJjaC52MS5TZWFyY2hSZWNvcmRzUmVzcG9uc2UwAULGAQoYY29tLmFnbnRjeS5kaXIuc2VhcmNoLnYxQhJTZWFyY2hTZXJ2aWNlUHJvdG9QAVojZ2l0aHViLmNvbS9hZ250Y3kvZGlyL2FwaS9zZWFyY2gvdjGiAgNBRFOqAhRBZ250Y3kuRGlyLlNlYXJjaC5WMcoCFEFnbnRjeVxEaXJcU2VhcmNoXFYx4gIgQWdudGN5XERpclxTZWFyY2hcVjFcR1BCTWV0YWRhdGHqAhdBZ250Y3k6OkRpcjo6U2VhcmNoOjpWMWIGcHJvdG8z", [file_agntcy_dir_core_v1_record, file_agntcy_dir_search_v1_record_query]);

/**
 * Describes the message agntcy.dir.search.v1.SearchCIDsRequest.
 * Use `create(SearchCIDsRequestSchema)` to create a new message.
 */
const SearchCIDsRequestSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_search_v1_search_service, 0);

/**
 * Describes the message agntcy.dir.search.v1.SearchRecordsRequest.
 * Use `create(SearchRecordsRequestSchema)` to create a new message.
 */
const SearchRecordsRequestSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_search_v1_search_service, 1);

/**
 * Describes the message agntcy.dir.search.v1.SearchCIDsResponse.
 * Use `create(SearchCIDsResponseSchema)` to create a new message.
 */
const SearchCIDsResponseSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_search_v1_search_service, 2);

/**
 * Describes the message agntcy.dir.search.v1.SearchRecordsResponse.
 * Use `create(SearchRecordsResponseSchema)` to create a new message.
 */
const SearchRecordsResponseSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_search_v1_search_service, 3);

/**
 * @generated from service agntcy.dir.search.v1.SearchService
 */
const SearchService = /*@__PURE__*/
  serviceDesc(file_agntcy_dir_search_v1_search_service, 0);

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

var search_v1 = /*#__PURE__*/Object.freeze({
    __proto__: null,
    RecordQuerySchema: RecordQuerySchema,
    RecordQueryType: RecordQueryType,
    RecordQueryTypeSchema: RecordQueryTypeSchema,
    SearchCIDsRequestSchema: SearchCIDsRequestSchema,
    SearchCIDsResponseSchema: SearchCIDsResponseSchema,
    SearchRecordsRequestSchema: SearchRecordsRequestSchema,
    SearchRecordsResponseSchema: SearchRecordsResponseSchema,
    SearchService: SearchService,
    file_agntcy_dir_search_v1_record_query: file_agntcy_dir_search_v1_record_query,
    file_agntcy_dir_search_v1_search_service: file_agntcy_dir_search_v1_search_service
});

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0


/**
 * Describes the file agntcy/dir/sign/v1/signature.proto.
 */
const file_agntcy_dir_sign_v1_signature = /*@__PURE__*/
  fileDesc("CiJhZ250Y3kvZGlyL3NpZ24vdjEvc2lnbmF0dXJlLnByb3RvEhJhZ250Y3kuZGlyLnNpZ24udjEigAIKCVNpZ25hdHVyZRJDCgthbm5vdGF0aW9ucxgBIAMoCzIuLmFnbnRjeS5kaXIuc2lnbi52MS5TaWduYXR1cmUuQW5ub3RhdGlvbnNFbnRyeRIRCglzaWduZWRfYXQYAiABKAkSEQoJYWxnb3JpdGhtGAMgASgJEhEKCXNpZ25hdHVyZRgEIAEoCRITCgtjZXJ0aWZpY2F0ZRgFIAEoCRIUCgxjb250ZW50X3R5cGUYBiABKAkSFgoOY29udGVudF9idW5kbGUYByABKAkaMgoQQW5ub3RhdGlvbnNFbnRyeRILCgNrZXkYASABKAkSDQoFdmFsdWUYAiABKAk6AjgBQrYBChZjb20uYWdudGN5LmRpci5zaWduLnYxQg5TaWduYXR1cmVQcm90b1ABWiFnaXRodWIuY29tL2FnbnRjeS9kaXIvYXBpL3NpZ24vdjGiAgNBRFOqAhJBZ250Y3kuRGlyLlNpZ24uVjHKAhJBZ250Y3lcRGlyXFNpZ25cVjHiAh5BZ250Y3lcRGlyXFNpZ25cVjFcR1BCTWV0YWRhdGHqAhVBZ250Y3k6OkRpcjo6U2lnbjo6VjFiBnByb3RvMw");

/**
 * Describes the message agntcy.dir.sign.v1.Signature.
 * Use `create(SignatureSchema)` to create a new message.
 */
const SignatureSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_sign_v1_signature, 0);

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0


/**
 * Describes the file agntcy/dir/sign/v1/sign_service.proto.
 */
const file_agntcy_dir_sign_v1_sign_service = /*@__PURE__*/
  fileDesc("CiVhZ250Y3kvZGlyL3NpZ24vdjEvc2lnbl9zZXJ2aWNlLnByb3RvEhJhZ250Y3kuZGlyLnNpZ24udjEiewoLU2lnblJlcXVlc3QSMQoKcmVjb3JkX3JlZhgBIAEoCzIdLmFnbnRjeS5kaXIuY29yZS52MS5SZWNvcmRSZWYSOQoIcHJvdmlkZXIYAiABKAsyJy5hZ250Y3kuZGlyLnNpZ24udjEuU2lnblJlcXVlc3RQcm92aWRlciKCAQoTU2lnblJlcXVlc3RQcm92aWRlchIwCgRvaWRjGAEgASgLMiAuYWdudGN5LmRpci5zaWduLnYxLlNpZ25XaXRoT0lEQ0gAEi4KA2tleRgCIAEoCzIfLmFnbnRjeS5kaXIuc2lnbi52MS5TaWduV2l0aEtleUgAQgkKB3JlcXVlc3QimwIKDFNpZ25XaXRoT0lEQxIQCghpZF90b2tlbhgBIAEoCRI6CgdvcHRpb25zGAIgASgLMikuYWdudGN5LmRpci5zaWduLnYxLlNpZ25XaXRoT0lEQy5TaWduT3B0cxq8AQoIU2lnbk9wdHMSFwoKZnVsY2lvX3VybBgBIAEoCUgAiAEBEhYKCXJla29yX3VybBgCIAEoCUgBiAEBEhoKDXRpbWVzdGFtcF91cmwYAyABKAlIAogBARIeChFvaWRjX3Byb3ZpZGVyX3VybBgEIAEoCUgDiAEBQg0KC19mdWxjaW9fdXJsQgwKCl9yZWtvcl91cmxCEAoOX3RpbWVzdGFtcF91cmxCFAoSX29pZGNfcHJvdmlkZXJfdXJsIkYKC1NpZ25XaXRoS2V5EhMKC3ByaXZhdGVfa2V5GAEgASgMEhUKCHBhc3N3b3JkGAIgASgMSACIAQFCCwoJX3Bhc3N3b3JkIkAKDFNpZ25SZXNwb25zZRIwCglzaWduYXR1cmUYASABKAsyHS5hZ250Y3kuZGlyLnNpZ24udjEuU2lnbmF0dXJlIkIKDVZlcmlmeVJlcXVlc3QSMQoKcmVjb3JkX3JlZhgBIAEoCzIdLmFnbnRjeS5kaXIuY29yZS52MS5SZWNvcmRSZWYiTwoOVmVyaWZ5UmVzcG9uc2USDwoHc3VjY2VzcxgBIAEoCBIaCg1lcnJvcl9tZXNzYWdlGAIgASgJSACIAQFCEAoOX2Vycm9yX21lc3NhZ2UyqQEKC1NpZ25TZXJ2aWNlEkkKBFNpZ24SHy5hZ250Y3kuZGlyLnNpZ24udjEuU2lnblJlcXVlc3QaIC5hZ250Y3kuZGlyLnNpZ24udjEuU2lnblJlc3BvbnNlEk8KBlZlcmlmeRIhLmFnbnRjeS5kaXIuc2lnbi52MS5WZXJpZnlSZXF1ZXN0GiIuYWdudGN5LmRpci5zaWduLnYxLlZlcmlmeVJlc3BvbnNlQrgBChZjb20uYWdudGN5LmRpci5zaWduLnYxQhBTaWduU2VydmljZVByb3RvUAFaIWdpdGh1Yi5jb20vYWdudGN5L2Rpci9hcGkvc2lnbi92MaICA0FEU6oCEkFnbnRjeS5EaXIuU2lnbi5WMcoCEkFnbnRjeVxEaXJcU2lnblxWMeICHkFnbnRjeVxEaXJcU2lnblxWMVxHUEJNZXRhZGF0YeoCFUFnbnRjeTo6RGlyOjpTaWduOjpWMWIGcHJvdG8z", [file_agntcy_dir_core_v1_record, file_agntcy_dir_sign_v1_signature]);

/**
 * Describes the message agntcy.dir.sign.v1.SignRequest.
 * Use `create(SignRequestSchema)` to create a new message.
 */
const SignRequestSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_sign_v1_sign_service, 0);

/**
 * Describes the message agntcy.dir.sign.v1.SignRequestProvider.
 * Use `create(SignRequestProviderSchema)` to create a new message.
 */
const SignRequestProviderSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_sign_v1_sign_service, 1);

/**
 * Describes the message agntcy.dir.sign.v1.SignWithOIDC.
 * Use `create(SignWithOIDCSchema)` to create a new message.
 */
const SignWithOIDCSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_sign_v1_sign_service, 2);

/**
 * Describes the message agntcy.dir.sign.v1.SignWithOIDC.SignOpts.
 * Use `create(SignWithOIDC_SignOptsSchema)` to create a new message.
 */
const SignWithOIDC_SignOptsSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_sign_v1_sign_service, 2, 0);

/**
 * Describes the message agntcy.dir.sign.v1.SignWithKey.
 * Use `create(SignWithKeySchema)` to create a new message.
 */
const SignWithKeySchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_sign_v1_sign_service, 3);

/**
 * Describes the message agntcy.dir.sign.v1.SignResponse.
 * Use `create(SignResponseSchema)` to create a new message.
 */
const SignResponseSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_sign_v1_sign_service, 4);

/**
 * Describes the message agntcy.dir.sign.v1.VerifyRequest.
 * Use `create(VerifyRequestSchema)` to create a new message.
 */
const VerifyRequestSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_sign_v1_sign_service, 5);

/**
 * Describes the message agntcy.dir.sign.v1.VerifyResponse.
 * Use `create(VerifyResponseSchema)` to create a new message.
 */
const VerifyResponseSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_sign_v1_sign_service, 6);

/**
 * SignService provides methods to sign and verify records.
 *
 * @generated from service agntcy.dir.sign.v1.SignService
 */
const SignService = /*@__PURE__*/
  serviceDesc(file_agntcy_dir_sign_v1_sign_service, 0);

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0


/**
 * Describes the file agntcy/dir/sign/v1/public_key.proto.
 */
const file_agntcy_dir_sign_v1_public_key = /*@__PURE__*/
  fileDesc("CiNhZ250Y3kvZGlyL3NpZ24vdjEvcHVibGljX2tleS5wcm90bxISYWdudGN5LmRpci5zaWduLnYxIhgKCVB1YmxpY0tleRILCgNrZXkYASABKAlCtgEKFmNvbS5hZ250Y3kuZGlyLnNpZ24udjFCDlB1YmxpY0tleVByb3RvUAFaIWdpdGh1Yi5jb20vYWdudGN5L2Rpci9hcGkvc2lnbi92MaICA0FEU6oCEkFnbnRjeS5EaXIuU2lnbi5WMcoCEkFnbnRjeVxEaXJcU2lnblxWMeICHkFnbnRjeVxEaXJcU2lnblxWMVxHUEJNZXRhZGF0YeoCFUFnbnRjeTo6RGlyOjpTaWduOjpWMWIGcHJvdG8z");

/**
 * Describes the message agntcy.dir.sign.v1.PublicKey.
 * Use `create(PublicKeySchema)` to create a new message.
 */
const PublicKeySchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_sign_v1_public_key, 0);

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

var sign_v1 = /*#__PURE__*/Object.freeze({
    __proto__: null,
    PublicKeySchema: PublicKeySchema,
    SignRequestProviderSchema: SignRequestProviderSchema,
    SignRequestSchema: SignRequestSchema,
    SignResponseSchema: SignResponseSchema,
    SignService: SignService,
    SignWithKeySchema: SignWithKeySchema,
    SignWithOIDCSchema: SignWithOIDCSchema,
    SignWithOIDC_SignOptsSchema: SignWithOIDC_SignOptsSchema,
    SignatureSchema: SignatureSchema,
    VerifyRequestSchema: VerifyRequestSchema,
    VerifyResponseSchema: VerifyResponseSchema,
    file_agntcy_dir_sign_v1_public_key: file_agntcy_dir_sign_v1_public_key,
    file_agntcy_dir_sign_v1_sign_service: file_agntcy_dir_sign_v1_sign_service,
    file_agntcy_dir_sign_v1_signature: file_agntcy_dir_sign_v1_signature
});

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0


/**
 * Describes the file agntcy/dir/store/v1/store_service.proto.
 */
const file_agntcy_dir_store_v1_store_service = /*@__PURE__*/
  fileDesc("CidhZ250Y3kvZGlyL3N0b3JlL3YxL3N0b3JlX3NlcnZpY2UucHJvdG8SE2FnbnRjeS5kaXIuc3RvcmUudjEifgoTUHVzaFJlZmVycmVyUmVxdWVzdBIxCgpyZWNvcmRfcmVmGAEgASgLMh0uYWdudGN5LmRpci5jb3JlLnYxLlJlY29yZFJlZhI0CghyZWZlcnJlchgCIAEoCzIiLmFnbnRjeS5kaXIuY29yZS52MS5SZWNvcmRSZWZlcnJlciJVChRQdXNoUmVmZXJyZXJSZXNwb25zZRIPCgdzdWNjZXNzGAEgASgIEhoKDWVycm9yX21lc3NhZ2UYAiABKAlIAIgBAUIQCg5fZXJyb3JfbWVzc2FnZSJ2ChNQdWxsUmVmZXJyZXJSZXF1ZXN0EjEKCnJlY29yZF9yZWYYASABKAsyHS5hZ250Y3kuZGlyLmNvcmUudjEuUmVjb3JkUmVmEhoKDXJlZmVycmVyX3R5cGUYAiABKAlIAIgBAUIQCg5fcmVmZXJyZXJfdHlwZSJMChRQdWxsUmVmZXJyZXJSZXNwb25zZRI0CghyZWZlcnJlchgBIAEoCzIiLmFnbnRjeS5kaXIuY29yZS52MS5SZWNvcmRSZWZlcnJlcjL+AwoMU3RvcmVTZXJ2aWNlEkUKBFB1c2gSGi5hZ250Y3kuZGlyLmNvcmUudjEuUmVjb3JkGh0uYWdudGN5LmRpci5jb3JlLnYxLlJlY29yZFJlZigBMAESRQoEUHVsbBIdLmFnbnRjeS5kaXIuY29yZS52MS5SZWNvcmRSZWYaGi5hZ250Y3kuZGlyLmNvcmUudjEuUmVjb3JkKAEwARJLCgZMb29rdXASHS5hZ250Y3kuZGlyLmNvcmUudjEuUmVjb3JkUmVmGh4uYWdudGN5LmRpci5jb3JlLnYxLlJlY29yZE1ldGEoATABEkEKBkRlbGV0ZRIdLmFnbnRjeS5kaXIuY29yZS52MS5SZWNvcmRSZWYaFi5nb29nbGUucHJvdG9idWYuRW1wdHkoARJnCgxQdXNoUmVmZXJyZXISKC5hZ250Y3kuZGlyLnN0b3JlLnYxLlB1c2hSZWZlcnJlclJlcXVlc3QaKS5hZ250Y3kuZGlyLnN0b3JlLnYxLlB1c2hSZWZlcnJlclJlc3BvbnNlKAEwARJnCgxQdWxsUmVmZXJyZXISKC5hZ250Y3kuZGlyLnN0b3JlLnYxLlB1bGxSZWZlcnJlclJlcXVlc3QaKS5hZ250Y3kuZGlyLnN0b3JlLnYxLlB1bGxSZWZlcnJlclJlc3BvbnNlKAEwAUK/AQoXY29tLmFnbnRjeS5kaXIuc3RvcmUudjFCEVN0b3JlU2VydmljZVByb3RvUAFaImdpdGh1Yi5jb20vYWdudGN5L2Rpci9hcGkvc3RvcmUvdjGiAgNBRFOqAhNBZ250Y3kuRGlyLlN0b3JlLlYxygITQWdudGN5XERpclxTdG9yZVxWMeICH0FnbnRjeVxEaXJcU3RvcmVcVjFcR1BCTWV0YWRhdGHqAhZBZ250Y3k6OkRpcjo6U3RvcmU6OlYxYgZwcm90bzM", [file_agntcy_dir_core_v1_record, file_google_protobuf_empty]);

/**
 * Describes the message agntcy.dir.store.v1.PushReferrerRequest.
 * Use `create(PushReferrerRequestSchema)` to create a new message.
 */
const PushReferrerRequestSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_store_v1_store_service, 0);

/**
 * Describes the message agntcy.dir.store.v1.PushReferrerResponse.
 * Use `create(PushReferrerResponseSchema)` to create a new message.
 */
const PushReferrerResponseSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_store_v1_store_service, 1);

/**
 * Describes the message agntcy.dir.store.v1.PullReferrerRequest.
 * Use `create(PullReferrerRequestSchema)` to create a new message.
 */
const PullReferrerRequestSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_store_v1_store_service, 2);

/**
 * Describes the message agntcy.dir.store.v1.PullReferrerResponse.
 * Use `create(PullReferrerResponseSchema)` to create a new message.
 */
const PullReferrerResponseSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_store_v1_store_service, 3);

/**
 * Defines an interface for content-addressable storage
 * service for objects.
 *
 * Max object size: 4MB (to fully fit in a single request)
 * Max metadata size: 100KB
 *
 * Store service can be implemented by various storage backends,
 * such as local file system, OCI registry, etc.
 *
 * Middleware should be used to control who can perform these RPCs.
 * Policies for the middleware can be handled via separate service.
 *
 * Each operation is performed sequentially, meaning that
 * for the N-th request, N-th response will be returned.
 * If an error occurs, the stream will be cancelled.
 *
 * @generated from service agntcy.dir.store.v1.StoreService
 */
const StoreService = /*@__PURE__*/
  serviceDesc(file_agntcy_dir_store_v1_store_service, 0);

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0


/**
 * Describes the file agntcy/dir/store/v1/sync_service.proto.
 */
const file_agntcy_dir_store_v1_sync_service = /*@__PURE__*/
  fileDesc("CiZhZ250Y3kvZGlyL3N0b3JlL3YxL3N5bmNfc2VydmljZS5wcm90bxITYWdudGN5LmRpci5zdG9yZS52MSI/ChFDcmVhdGVTeW5jUmVxdWVzdBIcChRyZW1vdGVfZGlyZWN0b3J5X3VybBgBIAEoCRIMCgRjaWRzGAIgAygJIiUKEkNyZWF0ZVN5bmNSZXNwb25zZRIPCgdzeW5jX2lkGAEgASgJIlAKEExpc3RTeW5jc1JlcXVlc3QSEgoFbGltaXQYAiABKA1IAIgBARITCgZvZmZzZXQYAyABKA1IAYgBAUIICgZfbGltaXRCCQoHX29mZnNldCJvCg1MaXN0U3luY3NJdGVtEg8KB3N5bmNfaWQYASABKAkSLwoGc3RhdHVzGAIgASgOMh8uYWdudGN5LmRpci5zdG9yZS52MS5TeW5jU3RhdHVzEhwKFHJlbW90ZV9kaXJlY3RvcnlfdXJsGAMgASgJIiEKDkdldFN5bmNSZXF1ZXN0Eg8KB3N5bmNfaWQYASABKAkioQEKD0dldFN5bmNSZXNwb25zZRIPCgdzeW5jX2lkGAEgASgJEi8KBnN0YXR1cxgCIAEoDjIfLmFnbnRjeS5kaXIuc3RvcmUudjEuU3luY1N0YXR1cxIcChRyZW1vdGVfZGlyZWN0b3J5X3VybBgDIAEoCRIUCgxjcmVhdGVkX3RpbWUYBCABKAkSGAoQbGFzdF91cGRhdGVfdGltZRgFIAEoCSIkChFEZWxldGVTeW5jUmVxdWVzdBIPCgdzeW5jX2lkGAEgASgJIhQKEkRlbGV0ZVN5bmNSZXNwb25zZSIjCiFSZXF1ZXN0UmVnaXN0cnlDcmVkZW50aWFsc1JlcXVlc3Qi4QEKIlJlcXVlc3RSZWdpc3RyeUNyZWRlbnRpYWxzUmVzcG9uc2USDwoHc3VjY2VzcxgBIAEoCBIVCg1lcnJvcl9tZXNzYWdlGAIgASgJEhgKEHJlZ2lzdHJ5X2FkZHJlc3MYAyABKAkSFwoPcmVwb3NpdG9yeV9uYW1lGAYgASgJEj8KCmJhc2ljX2F1dGgYBCABKAsyKS5hZ250Y3kuZGlyLnN0b3JlLnYxLkJhc2ljQXV0aENyZWRlbnRpYWxzSAASEAoIaW5zZWN1cmUYByABKAhCDQoLY3JlZGVudGlhbHMiOgoUQmFzaWNBdXRoQ3JlZGVudGlhbHMSEAoIdXNlcm5hbWUYASABKAkSEAoIcGFzc3dvcmQYAiABKAkqywEKClN5bmNTdGF0dXMSGwoXU1lOQ19TVEFUVVNfVU5TUEVDSUZJRUQQABIXChNTWU5DX1NUQVRVU19QRU5ESU5HEAESGwoXU1lOQ19TVEFUVVNfSU5fUFJPR1JFU1MQAhIWChJTWU5DX1NUQVRVU19GQUlMRUQQAxIeChpTWU5DX1NUQVRVU19ERUxFVEVfUEVORElORxAEEhcKE1NZTkNfU1RBVFVTX0RFTEVURUQQBRIZChVTWU5DX1NUQVRVU19DT01QTEVURUQQBjKLBAoLU3luY1NlcnZpY2USXQoKQ3JlYXRlU3luYxImLmFnbnRjeS5kaXIuc3RvcmUudjEuQ3JlYXRlU3luY1JlcXVlc3QaJy5hZ250Y3kuZGlyLnN0b3JlLnYxLkNyZWF0ZVN5bmNSZXNwb25zZRJYCglMaXN0U3luY3MSJS5hZ250Y3kuZGlyLnN0b3JlLnYxLkxpc3RTeW5jc1JlcXVlc3QaIi5hZ250Y3kuZGlyLnN0b3JlLnYxLkxpc3RTeW5jc0l0ZW0wARJUCgdHZXRTeW5jEiMuYWdudGN5LmRpci5zdG9yZS52MS5HZXRTeW5jUmVxdWVzdBokLmFnbnRjeS5kaXIuc3RvcmUudjEuR2V0U3luY1Jlc3BvbnNlEl0KCkRlbGV0ZVN5bmMSJi5hZ250Y3kuZGlyLnN0b3JlLnYxLkRlbGV0ZVN5bmNSZXF1ZXN0GicuYWdudGN5LmRpci5zdG9yZS52MS5EZWxldGVTeW5jUmVzcG9uc2USjQEKGlJlcXVlc3RSZWdpc3RyeUNyZWRlbnRpYWxzEjYuYWdudGN5LmRpci5zdG9yZS52MS5SZXF1ZXN0UmVnaXN0cnlDcmVkZW50aWFsc1JlcXVlc3QaNy5hZ250Y3kuZGlyLnN0b3JlLnYxLlJlcXVlc3RSZWdpc3RyeUNyZWRlbnRpYWxzUmVzcG9uc2VCvgEKF2NvbS5hZ250Y3kuZGlyLnN0b3JlLnYxQhBTeW5jU2VydmljZVByb3RvUAFaImdpdGh1Yi5jb20vYWdudGN5L2Rpci9hcGkvc3RvcmUvdjGiAgNBRFOqAhNBZ250Y3kuRGlyLlN0b3JlLlYxygITQWdudGN5XERpclxTdG9yZVxWMeICH0FnbnRjeVxEaXJcU3RvcmVcVjFcR1BCTWV0YWRhdGHqAhZBZ250Y3k6OkRpcjo6U3RvcmU6OlYxYgZwcm90bzM");

/**
 * Describes the message agntcy.dir.store.v1.CreateSyncRequest.
 * Use `create(CreateSyncRequestSchema)` to create a new message.
 */
const CreateSyncRequestSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_store_v1_sync_service, 0);

/**
 * Describes the message agntcy.dir.store.v1.CreateSyncResponse.
 * Use `create(CreateSyncResponseSchema)` to create a new message.
 */
const CreateSyncResponseSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_store_v1_sync_service, 1);

/**
 * Describes the message agntcy.dir.store.v1.ListSyncsRequest.
 * Use `create(ListSyncsRequestSchema)` to create a new message.
 */
const ListSyncsRequestSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_store_v1_sync_service, 2);

/**
 * Describes the message agntcy.dir.store.v1.ListSyncsItem.
 * Use `create(ListSyncsItemSchema)` to create a new message.
 */
const ListSyncsItemSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_store_v1_sync_service, 3);

/**
 * Describes the message agntcy.dir.store.v1.GetSyncRequest.
 * Use `create(GetSyncRequestSchema)` to create a new message.
 */
const GetSyncRequestSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_store_v1_sync_service, 4);

/**
 * Describes the message agntcy.dir.store.v1.GetSyncResponse.
 * Use `create(GetSyncResponseSchema)` to create a new message.
 */
const GetSyncResponseSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_store_v1_sync_service, 5);

/**
 * Describes the message agntcy.dir.store.v1.DeleteSyncRequest.
 * Use `create(DeleteSyncRequestSchema)` to create a new message.
 */
const DeleteSyncRequestSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_store_v1_sync_service, 6);

/**
 * Describes the message agntcy.dir.store.v1.DeleteSyncResponse.
 * Use `create(DeleteSyncResponseSchema)` to create a new message.
 */
const DeleteSyncResponseSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_store_v1_sync_service, 7);

/**
 * Describes the message agntcy.dir.store.v1.RequestRegistryCredentialsRequest.
 * Use `create(RequestRegistryCredentialsRequestSchema)` to create a new message.
 */
const RequestRegistryCredentialsRequestSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_store_v1_sync_service, 8);

/**
 * Describes the message agntcy.dir.store.v1.RequestRegistryCredentialsResponse.
 * Use `create(RequestRegistryCredentialsResponseSchema)` to create a new message.
 */
const RequestRegistryCredentialsResponseSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_store_v1_sync_service, 9);

/**
 * Describes the message agntcy.dir.store.v1.BasicAuthCredentials.
 * Use `create(BasicAuthCredentialsSchema)` to create a new message.
 */
const BasicAuthCredentialsSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_store_v1_sync_service, 10);

/**
 * Describes the enum agntcy.dir.store.v1.SyncStatus.
 */
const SyncStatusSchema = /*@__PURE__*/
  enumDesc(file_agntcy_dir_store_v1_sync_service, 0);

/**
 * SyncStatus enumeration defines the possible states of a synchronization operation.
 *
 * @generated from enum agntcy.dir.store.v1.SyncStatus
 */
const SyncStatus = /*@__PURE__*/
  tsEnum(SyncStatusSchema);

/**
 * SyncService provides functionality for synchronizing objects between Directory nodes.
 *
 * This service enables one-way synchronization from a remote Directory node to the local node,
 * allowing distributed Directory instances to share and replicate objects. The service supports
 * both on-demand synchronization and tracking of sync operations through their lifecycle.
 *
 * @generated from service agntcy.dir.store.v1.SyncService
 */
const SyncService = /*@__PURE__*/
  serviceDesc(file_agntcy_dir_store_v1_sync_service, 0);

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

var store_v1 = /*#__PURE__*/Object.freeze({
    __proto__: null,
    BasicAuthCredentialsSchema: BasicAuthCredentialsSchema,
    CreateSyncRequestSchema: CreateSyncRequestSchema,
    CreateSyncResponseSchema: CreateSyncResponseSchema,
    DeleteSyncRequestSchema: DeleteSyncRequestSchema,
    DeleteSyncResponseSchema: DeleteSyncResponseSchema,
    GetSyncRequestSchema: GetSyncRequestSchema,
    GetSyncResponseSchema: GetSyncResponseSchema,
    ListSyncsItemSchema: ListSyncsItemSchema,
    ListSyncsRequestSchema: ListSyncsRequestSchema,
    PullReferrerRequestSchema: PullReferrerRequestSchema,
    PullReferrerResponseSchema: PullReferrerResponseSchema,
    PushReferrerRequestSchema: PushReferrerRequestSchema,
    PushReferrerResponseSchema: PushReferrerResponseSchema,
    RequestRegistryCredentialsRequestSchema: RequestRegistryCredentialsRequestSchema,
    RequestRegistryCredentialsResponseSchema: RequestRegistryCredentialsResponseSchema,
    StoreService: StoreService,
    SyncService: SyncService,
    SyncStatus: SyncStatus,
    SyncStatusSchema: SyncStatusSchema,
    file_agntcy_dir_store_v1_store_service: file_agntcy_dir_store_v1_store_service,
    file_agntcy_dir_store_v1_sync_service: file_agntcy_dir_store_v1_sync_service
});

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0


/**
 * Describes the file agntcy/dir/events/v1/event_service.proto.
 */
const file_agntcy_dir_events_v1_event_service = /*@__PURE__*/
  fileDesc("CihhZ250Y3kvZGlyL2V2ZW50cy92MS9ldmVudF9zZXJ2aWNlLnByb3RvEhRhZ250Y3kuZGlyLmV2ZW50cy52MSJxCg1MaXN0ZW5SZXF1ZXN0EjQKC2V2ZW50X3R5cGVzGAEgAygOMh8uYWdudGN5LmRpci5ldmVudHMudjEuRXZlbnRUeXBlEhUKDWxhYmVsX2ZpbHRlcnMYAiADKAkSEwoLY2lkX2ZpbHRlcnMYAyADKAkiPAoOTGlzdGVuUmVzcG9uc2USKgoFZXZlbnQYASABKAsyGy5hZ250Y3kuZGlyLmV2ZW50cy52MS5FdmVudCKEAgoFRXZlbnQSCgoCaWQYASABKAkSLQoEdHlwZRgCIAEoDjIfLmFnbnRjeS5kaXIuZXZlbnRzLnYxLkV2ZW50VHlwZRItCgl0aW1lc3RhbXAYAyABKAsyGi5nb29nbGUucHJvdG9idWYuVGltZXN0YW1wEhMKC3Jlc291cmNlX2lkGAQgASgJEg4KBmxhYmVscxgFIAMoCRI7CghtZXRhZGF0YRgHIAMoCzIpLmFnbnRjeS5kaXIuZXZlbnRzLnYxLkV2ZW50Lk1ldGFkYXRhRW50cnkaLwoNTWV0YWRhdGFFbnRyeRILCgNrZXkYASABKAkSDQoFdmFsdWUYAiABKAk6AjgBKoADCglFdmVudFR5cGUSGgoWRVZFTlRfVFlQRV9VTlNQRUNJRklFRBAAEhwKGEVWRU5UX1RZUEVfUkVDT1JEX1BVU0hFRBABEhwKGEVWRU5UX1RZUEVfUkVDT1JEX1BVTExFRBACEh0KGUVWRU5UX1RZUEVfUkVDT1JEX0RFTEVURUQQAxIfChtFVkVOVF9UWVBFX1JFQ09SRF9QVUJMSVNIRUQQBBIhCh1FVkVOVF9UWVBFX1JFQ09SRF9VTlBVQkxJU0hFRBAFEhsKF0VWRU5UX1RZUEVfU1lOQ19DUkVBVEVEEAYSHQoZRVZFTlRfVFlQRV9TWU5DX0NPTVBMRVRFRBAHEhoKFkVWRU5UX1RZUEVfU1lOQ19GQUlMRUQQCBIcChhFVkVOVF9UWVBFX1JFQ09SRF9TSUdORUQQCRIeChpFVkVOVF9UWVBFX1JFQ09SRF9WRVJJRklFRBAKEiIKHkVWRU5UX1RZUEVfUFVCTElDX0tFWV9VUExPQURFRBALMmUKDEV2ZW50U2VydmljZRJVCgZMaXN0ZW4SIy5hZ250Y3kuZGlyLmV2ZW50cy52MS5MaXN0ZW5SZXF1ZXN0GiQuYWdudGN5LmRpci5ldmVudHMudjEuTGlzdGVuUmVzcG9uc2UwAULFAQoYY29tLmFnbnRjeS5kaXIuZXZlbnRzLnYxQhFFdmVudFNlcnZpY2VQcm90b1ABWiNnaXRodWIuY29tL2FnbnRjeS9kaXIvYXBpL2V2ZW50cy92MaICA0FERaoCFEFnbnRjeS5EaXIuRXZlbnRzLlYxygIUQWdudGN5XERpclxFdmVudHNcVjHiAiBBZ250Y3lcRGlyXEV2ZW50c1xWMVxHUEJNZXRhZGF0YeoCF0FnbnRjeTo6RGlyOjpFdmVudHM6OlYxYgZwcm90bzM", [file_google_protobuf_timestamp]);

/**
 * Describes the message agntcy.dir.events.v1.ListenRequest.
 * Use `create(ListenRequestSchema)` to create a new message.
 */
const ListenRequestSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_events_v1_event_service, 0);

/**
 * Describes the message agntcy.dir.events.v1.ListenResponse.
 * Use `create(ListenResponseSchema)` to create a new message.
 */
const ListenResponseSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_events_v1_event_service, 1);

/**
 * Describes the message agntcy.dir.events.v1.Event.
 * Use `create(EventSchema)` to create a new message.
 */
const EventSchema = /*@__PURE__*/
  messageDesc(file_agntcy_dir_events_v1_event_service, 2);

/**
 * Describes the enum agntcy.dir.events.v1.EventType.
 */
const EventTypeSchema = /*@__PURE__*/
  enumDesc(file_agntcy_dir_events_v1_event_service, 0);

/**
 * EventType represents all valid event types in the system.
 * Each value represents a specific operation that can occur.
 *
 * Supported Events:
 * - Store: RECORD_PUSHED, RECORD_PULLED, RECORD_DELETED
 * - Routing: RECORD_PUBLISHED, RECORD_UNPUBLISHED
 * - Sync: SYNC_CREATED, SYNC_COMPLETED, SYNC_FAILED
 * - Sign: RECORD_SIGNED
 *
 * @generated from enum agntcy.dir.events.v1.EventType
 */
const EventType = /*@__PURE__*/
  tsEnum(EventTypeSchema);

/**
 * EventService provides real-time event streaming for all system operations.
 * Events are delivered from subscription time forward with no history or replay.
 * This service enables external applications to react to system changes in real-time.
 *
 * @generated from service agntcy.dir.events.v1.EventService
 */
const EventService = /*@__PURE__*/
  serviceDesc(file_agntcy_dir_events_v1_event_service, 0);

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

var events_v1 = /*#__PURE__*/Object.freeze({
    __proto__: null,
    EventSchema: EventSchema,
    EventService: EventService,
    EventType: EventType,
    EventTypeSchema: EventTypeSchema,
    ListenRequestSchema: ListenRequestSchema,
    ListenResponseSchema: ListenResponseSchema,
    file_agntcy_dir_events_v1_event_service: file_agntcy_dir_events_v1_event_service
});

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

var index = /*#__PURE__*/Object.freeze({
    __proto__: null,
    core_v1: core_v1,
    events_v1: events_v1,
    naming_v1: naming_v1,
    routing_v1: routing_v1,
    search_v1: search_v1,
    sign_v1: sign_v1,
    store_v1: store_v1
});

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0
/**
 * Configuration class for the AGNTCY Directory client.
 *
 * This class manages configuration settings for connecting to the Directory service
 * and provides default values and environment-based configuration loading.
 */
class Config {
    /**
     * Creates a new Config instance.
     *
     * @param serverAddress - The server address to connect to. Defaults to '127.0.0.1:8888'
     * @param dirctlPath - Path to the dirctl executable. Defaults to 'dirctl'
     * @param spiffeEndpointSocket - Path to the spire server socket. Defaults to empty string.
     * @param authMode - Authentication mode: '' for insecure, 'x509', 'jwt' or 'tls'. Defaults to ''
     * @param jwtAudience - JWT audience for JWT authentication. Required when authMode is 'jwt'
     */
    constructor(serverAddress = Config.DEFAULT_SERVER_ADDRESS, dirctlPath = Config.DEFAULT_DIRCTL_PATH, spiffeEndpointSocket = Config.DEFAULT_SPIFFE_ENDPOINT_SOCKET, authMode = Config.DEFAULT_AUTH_MODE, jwtAudience = Config.DEFAULT_JWT_AUDIENCE, tlsCaFile = Config.DEFAULT_TLS_CA_FILE, tlsCertFile = Config.DEFAULT_TLS_CERT_FILE, tlsKeyFile = Config.DEFAULT_TLS_KEY_FILE) {
        // add protocol prefix if not set
        // use unsafe http unless spire/auth is used
        if (!serverAddress.startsWith('http://') &&
            !serverAddress.startsWith('https://')) {
            // use https protocol when X.509, JWT, or TLS auth is used
            if (authMode === 'x509' || authMode === 'jwt' || authMode === 'tls') {
                serverAddress = `https://${serverAddress}`;
            }
            else {
                serverAddress = `http://${serverAddress}`;
            }
        }
        this.serverAddress = serverAddress;
        this.dirctlPath = dirctlPath;
        this.spiffeEndpointSocket = spiffeEndpointSocket;
        this.authMode = authMode;
        this.jwtAudience = jwtAudience;
        this.tlsCaFile = tlsCaFile;
        this.tlsCertFile = tlsCertFile;
        this.tlsKeyFile = tlsKeyFile;
    }
    /**
     * Load configuration from environment variables.
     *
     * @param prefix - Environment variable prefix. Defaults to 'DIRECTORY_CLIENT_'
     * @returns A new Config instance with values loaded from environment variables
     *
     * @example
     * ```typescript
     * // Load with default prefix
     * const config = Config.loadFromEnv();
     *
     * // Load with custom prefix
     * const config = Config.loadFromEnv("MY_APP_");
     * ```
     */
    static loadFromEnv(prefix = 'DIRECTORY_CLIENT_') {
        // Load dirctl path from env without env prefix
        const dirctlPath = env['DIRCTL_PATH'] || Config.DEFAULT_DIRCTL_PATH;
        // Load other config values with env prefix
        const serverAddress = env[`${prefix}SERVER_ADDRESS`] || Config.DEFAULT_SERVER_ADDRESS;
        const spiffeEndpointSocketPath = env[`${prefix}SPIFFE_SOCKET_PATH`] || Config.DEFAULT_SPIFFE_ENDPOINT_SOCKET;
        const authMode = (env[`${prefix}AUTH_MODE`] || Config.DEFAULT_AUTH_MODE);
        const jwtAudience = env[`${prefix}JWT_AUDIENCE`] || Config.DEFAULT_JWT_AUDIENCE;
        const tlsCaFile = env[`${prefix}TLS_CA_FILE`] || Config.DEFAULT_TLS_CA_FILE;
        const tlsCertFile = env[`${prefix}TLS_CERT_FILE`] || Config.DEFAULT_TLS_CERT_FILE;
        const tlsKeyFile = env[`${prefix}TLS_KEY_FILE`] || Config.DEFAULT_TLS_KEY_FILE;
        return new Config(serverAddress, dirctlPath, spiffeEndpointSocketPath, authMode, jwtAudience, tlsCaFile, tlsCertFile, tlsKeyFile);
    }
}
Config.DEFAULT_SERVER_ADDRESS = '127.0.0.1:8888';
Config.DEFAULT_DIRCTL_PATH = 'dirctl';
Config.DEFAULT_SPIFFE_ENDPOINT_SOCKET = '';
Config.DEFAULT_AUTH_MODE = '';
Config.DEFAULT_JWT_AUDIENCE = '';
Config.DEFAULT_TLS_CA_FILE = '';
Config.DEFAULT_TLS_CERT_FILE = '';
Config.DEFAULT_TLS_KEY_FILE = '';
/**
 * High-level client for interacting with AGNTCY Directory services.
 *
 * This client provides a unified interface for operations across the Directory API.
 * It handles gRPC communication and provides convenient methods for common operations
 * including storage, routing, search, signing, and synchronization.
 *
 * @example
 * ```typescript
 * // Create client with default configuration
 * const client = new Client();
 *
 * // Create client with custom configuration
 * const config = new Config('localhost:8888', '/usr/local/bin/dirctl');
 * const client = new Client(config);
 *
 * // Use client for operations
 * const records = await client.push([record]);
 * ```
 */
class Client {
    constructor(config, grpcTransport) {
        // Load config from environment if not provided
        if (!config) {
            config = Config.loadFromEnv();
        }
        this.config = config;
        // if no transport provided, use insecure transport
        if (!grpcTransport) {
            grpcTransport = createGrpcTransport({
                baseUrl: config.serverAddress,
            });
        }
        // Set clients for all services
        this.storeClient = createClient(StoreService, grpcTransport);
        this.routingClient = createClient(RoutingService, grpcTransport);
        this.publicationClient = createClient(PublicationService, grpcTransport);
        this.searchClient = createClient(SearchService, grpcTransport);
        this.signClient = createClient(SignService, grpcTransport);
        this.syncClient = createClient(SyncService, grpcTransport);
        this.eventClient = createClient(EventService, grpcTransport);
        this.namingClient = createClient(NamingService, grpcTransport);
    }
    static convertToPEM(bytes, label) {
        // Convert Uint8Array to base64 string
        let binary = '';
        const len = bytes.byteLength;
        for (let i = 0; i < len; i++) {
            binary += String.fromCharCode(bytes[i]);
        }
        const base64String = btoa(binary);
        // Split base64 string into 64-character lines
        const lines = base64String.match(/.{1,64}/g) || [];
        // Build PEM formatted string with headers and footers
        const pem = [
            `-----BEGIN ${label}-----`,
            ...lines,
            `-----END ${label}-----`
        ].join('\n');
        return pem;
    }
    static async createGRPCTransport(config) {
        // Handle different authentication modes
        switch (config.authMode) {
            case '':
                return createGrpcTransport({
                    baseUrl: config.serverAddress,
                });
            case 'jwt':
                return await this.createJWTTransport(config);
            case 'x509':
                return await this.createX509Transport(config);
            case 'tls':
                return await this.createTLSTransport(config);
            default:
                throw new Error(`Unsupported auth mode: ${config.authMode}`);
        }
    }
    static async createX509Transport(config) {
        if (config.spiffeEndpointSocket === '') {
            throw new Error('SPIFFE socket path is required for X.509 authentication');
        }
        // Create secure transport with SPIFFE X.509
        const client = createClient$1(config.spiffeEndpointSocket);
        let svid = {
            x509Svid: new Uint8Array(),
            x509SvidKey: new Uint8Array(),
            bundle: new Uint8Array(),
        };
        const svidStream = client.fetchX509SVID({});
        for await (const message of svidStream.responses) {
            message.svids.forEach((_svid) => {
                svid = _svid;
            });
            if (message.svids.length > 0) {
                break;
            }
        }
        // Create transport settings for gRPC client
        const transport = createGrpcTransport({
            baseUrl: config.serverAddress,
            nodeOptions: {
                ca: this.convertToPEM(svid.bundle, "TRUSTED CERTIFICATE"),
                cert: this.convertToPEM(svid.x509Svid, "CERTIFICATE"),
                key: this.convertToPEM(svid.x509SvidKey, "PRIVATE KEY"),
            },
        });
        return transport;
    }
    static async createJWTTransport(config) {
        if (config.spiffeEndpointSocket === '') {
            throw new Error('SPIFFE socket path is required for JWT authentication');
        }
        if (config.jwtAudience === '') {
            throw new Error('JWT audience is required for JWT authentication');
        }
        // Create SPIFFE client
        const client = createClient$1(config.spiffeEndpointSocket);
        // Fetch X.509 bundle for verifying server's TLS certificate
        // In JWT mode, the server presents its X.509-SVID via TLS for transport security
        let bundle = null;
        const bundleStream = client.fetchX509Bundles({});
        for await (const message of bundleStream.responses) {
            // Get the first bundle from the bundles map
            // bundles is a map<string, bytes> where bytes is ASN.1 DER encoded
            for (const [_, bundleData] of Object.entries(message.bundles)) {
                // Convert to a new Uint8Array to ensure type compatibility
                bundle = new Uint8Array(bundleData);
                break;
            }
            if (bundle !== null) {
                break;
            }
        }
        if (bundle === null || bundle.length === 0) {
            throw new Error('Failed to fetch X.509 bundle from SPIRE: no bundles returned');
        }
        // Create JWT interceptor that fetches and injects JWT tokens
        const jwtInterceptor = (next) => async (req) => {
            // Fetch JWT-SVID from SPIRE
            // Note: spiffeId is empty string to use the workload's default identity
            const jwtCall = client.fetchJWTSVID({
                spiffeId: '',
                audience: [config.jwtAudience]
            });
            const response = await jwtCall.response;
            if (!response.svids || response.svids.length === 0) {
                throw new Error('Failed to fetch JWT-SVID from SPIRE: no SVIDs returned');
            }
            const jwtToken = response.svids[0].svid;
            // Add JWT token to request headers
            req.header.set('authorization', `Bearer ${jwtToken}`);
            return await next(req);
        };
        // Create transport with JWT interceptor and TLS using SPIFFE bundle
        // For JWT mode: Server presents X.509-SVID via TLS, clients authenticate with JWT-SVID
        const transport = createGrpcTransport({
            baseUrl: config.serverAddress,
            interceptors: [jwtInterceptor],
            nodeOptions: {
                ca: this.convertToPEM(bundle, "CERTIFICATE"),
            },
        });
        return transport;
    }
    static async createTLSTransport(config) {
        if (config.tlsCaFile === '') {
            throw new Error('TLS CA file is required for TLS authentication');
        }
        if (config.tlsCertFile === '') {
            throw new Error('TLS certificate file is required for TLS authentication');
        }
        if (config.tlsKeyFile === '') {
            throw new Error('TLS key file is required for TLS authentication');
        }
        let root_ca;
        let cert_chain;
        let private_key;
        try {
            root_ca = readFileSync(config.tlsCaFile).toString();
            cert_chain = readFileSync(config.tlsCertFile).toString();
            private_key = readFileSync(config.tlsKeyFile).toString();
        }
        catch (e) {
            console.error('Error reading file:', e.message);
            throw e;
        }
        const transport = createGrpcTransport({
            baseUrl: config.serverAddress,
            nodeOptions: {
                ca: root_ca,
                cert: cert_chain,
                key: private_key,
            },
        });
        return transport;
    }
    /**
     * Request generator helper function for streaming requests.
     */
    async *requestGenerator(reqs) {
        for (const req of reqs) {
            yield req;
        }
    }
    /**
     * Push records to the Store API.
     *
     * Uploads one or more records to the content store, making them available
     * for retrieval and reference. Each record is assigned a unique content
     * identifier (CID) based on its content hash.
     *
     * @param records - Array of Record objects to push to the store
     * @returns Promise that resolves to an array of RecordRef objects containing the CIDs of the pushed records
     *
     * @throws {Error} If the gRPC call fails or the push operation fails
     *
     * @example
     * ```typescript
     * const records = [createRecord("example")];
     * const refs = await client.push(records);
     * console.log(`Pushed with CID: ${refs[0].cid}`);
     * ```
     */
    async push(records) {
        const responses = [];
        for await (const response of this.storeClient.push(this.requestGenerator(records))) {
            responses.push(response);
        }
        return responses;
    }
    /**
     * Push records with referrer metadata to the Store API.
     *
     * Uploads records along with optional artifacts and referrer information.
     * This is useful for pushing complex objects that include additional
     * metadata or associated artifacts.
     *
     * @param requests - Array of PushReferrerRequest objects containing records and optional artifacts
     * @returns Promise that resolves to an array of PushReferrerResponse objects containing the details of pushed artifacts
     *
     * @throws {Error} If the gRPC call fails or the push operation fails
     *
     * @example
     * ```typescript
     * const requests = [new models.store_v1.PushReferrerRequest({record: record})];
     * const responses = await client.push_referrer(requests);
     * ```
     */
    async push_referrer(requests) {
        const responses = [];
        for await (const response of this.storeClient.pushReferrer(this.requestGenerator(requests))) {
            responses.push(response);
        }
        return responses;
    }
    /**
     * Pull records from the Store API by their references.
     *
     * Retrieves one or more records from the content store using their
     * content identifiers (CIDs).
     *
     * @param refs - Array of RecordRef objects containing the CIDs to retrieve
     * @returns Promise that resolves to an array of Record objects retrieved from the store
     *
     * @throws {Error} If the gRPC call fails or the pull operation fails
     *
     * @example
     * ```typescript
     * const refs = [new models.core_v1.RecordRef({cid: "QmExample123"})];
     * const records = await client.pull(refs);
     * for (const record of records) {
     *   console.log(`Retrieved record: ${record}`);
     * }
     * ```
     */
    async pull(refs) {
        const records = [];
        for await (const response of this.storeClient.pull(this.requestGenerator(refs))) {
            records.push(response);
        }
        return records;
    }
    /**
     * Pull records with referrer metadata from the Store API.
     *
     * Retrieves records along with their associated artifacts and referrer
     * information. This provides access to complex objects that include
     * additional metadata or associated artifacts.
     *
     * @param requests - Array of PullReferrerRequest objects containing records and optional artifacts for pull operations
     * @returns Promise that resolves to an array of PullReferrerResponse objects containing the retrieved records
     *
     * @throws {Error} If the gRPC call fails or the pull operation fails
     *
     * @example
     * ```typescript
     * const requests = [new models.store_v1.PullReferrerRequest({ref: ref})];
     * const responses = await client.pull_referrer(requests);
     * for (const response of responses) {
     *   console.log(`Retrieved: ${response}`);
     * }
     * ```
     */
    async pull_referrer(requests) {
        const responses = [];
        for await (const response of this.storeClient.pullReferrer(this.requestGenerator(requests))) {
            responses.push(response);
        }
        return responses;
    }
    /**
     * Search objects from the Store API matching the specified queries.
     *
     * Performs a search across the storage using the provided search queries
     * and returns a list of matching CIDs. This is efficient for lookups
     * where only the CIDs are needed.
     *
     * @param request - SearchCIDsRequest containing queries, filters, and search options
     * @returns Promise that resolves to an array of SearchCIDsResponse objects matching the queries
     *
     * @throws {Error} If the gRPC call fails or the search operation fails
     *
     * @example
     * ```typescript
     * const request = create(models.search_v1.SearchCIDsRequestSchema, {queries: [query], limit: 10});
     * const responses = await client.searchCIDs(request);
     * for (const response of responses) {
     *   console.log(`Found CID: ${response.recordCid}`);
     * }
     * ```
     */
    async searchCIDs(request) {
        const responses = [];
        for await (const response of this.searchClient.searchCIDs(request)) {
            responses.push(response);
        }
        return responses;
    }
    /**
     * Search for full records from the Store API matching the specified queries.
     *
     * Performs a search across the storage using the provided search queries
     * and returns a list of full records with all metadata.
     *
     * @param request - SearchRecordsRequest containing queries, filters, and search options
     * @returns Promise that resolves to an array of SearchRecordsResponse objects matching the queries
     *
     * @throws {Error} If the gRPC call fails or the search operation fails
     *
     * @example
     * ```typescript
     * const request = create(models.search_v1.SearchRecordsRequestSchema, {queries: [query], limit: 10});
     * const responses = await client.searchRecords(request);
     * for (const response of responses) {
     *   console.log(`Found: ${response.record?.name}`);
     * }
     * ```
     */
    async searchRecords(request) {
        const responses = [];
        for await (const response of this.searchClient.searchRecords(request)) {
            responses.push(response);
        }
        return responses;
    }
    /**
     * Look up metadata for records in the Store API.
     *
     * Retrieves metadata information for one or more records without
     * downloading the full record content. This is useful for checking
     * if records exist and getting basic information about them.
     *
     * @param refs - Array of RecordRef objects containing the CIDs to look up
     * @returns Promise that resolves to an array of RecordMeta objects containing metadata for the records
     *
     * @throws {Error} If the gRPC call fails or the lookup operation fails
     *
     * @example
     * ```typescript
     * const refs = [new models.core_v1.RecordRef({cid: "QmExample123"})];
     * const metadatas = await client.lookup(refs);
     * for (const meta of metadatas) {
     *   console.log(`Record size: ${meta.size}`);
     * }
     * ```
     */
    async lookup(refs) {
        const recordMetas = [];
        for await (const response of this.storeClient.lookup(this.requestGenerator(refs))) {
            recordMetas.push(response);
        }
        return recordMetas;
    }
    /**
     * List objects from the Routing API matching the specified criteria.
     *
     * Returns a list of objects that match the filtering and
     * query criteria specified in the request.
     *
     * @param request - ListRequest specifying filtering criteria, pagination, etc.
     * @returns Promise that resolves to an array of ListResponse objects matching the criteria
     *
     * @throws {Error} If the gRPC call fails or the list operation fails
     *
     * @example
     * ```typescript
     * const request = new models.routing_v1.ListRequest({limit: 10});
     * const responses = await client.list(request);
     * for (const response of responses) {
     *   console.log(`Found object: ${response.cid}`);
     * }
     * ```
     */
    async list(request) {
        const results = [];
        for await (const response of this.routingClient.list(request)) {
            results.push(response);
        }
        return results;
    }
    /**
     * Publish objects to the Routing API matching the specified criteria.
     *
     * Makes the specified objects available for discovery and retrieval by other
     * clients in the network. The objects must already exist in the store before
     * they can be published.
     *
     * @param request - PublishRequest containing the query for the objects to publish
     * @returns Promise that resolves when the publish operation is complete
     *
     * @throws {Error} If the gRPC call fails or the object cannot be published
     *
     * @example
     * ```typescript
     * const ref = new models.routing_v1.RecordRef({cid: "QmExample123"});
     * const request = new models.routing_v1.PublishRequest({recordRefs: [ref]});
     * await client.publish(request);
     * ```
     */
    async publish(request) {
        await this.routingClient.publish(request);
    }
    /**
     * Unpublish objects from the Routing API matching the specified criteria.
     *
     * Removes the specified objects from the public network, making them no
     * longer discoverable by other clients. The objects remain in the local
     * store but are not available for network discovery.
     *
     * @param request - UnpublishRequest containing the query for the objects to unpublish
     * @returns Promise that resolves when the unpublish operation is complete
     *
     * @throws {Error} If the gRPC call fails or the objects cannot be unpublished
     *
     * @example
     * ```typescript
     * const ref = new models.routing_v1.RecordRef({cid: "QmExample123"});
     * const request = new models.routing_v1.UnpublishRequest({recordRefs: [ref]});
     * await client.unpublish(request);
     * ```
     */
    async unpublish(request) {
        await this.routingClient.unpublish(request);
    }
    /**
     * Delete records from the Store API.
     *
     * Permanently removes one or more records from the content store using
     * their content identifiers (CIDs). This operation cannot be undone.
     *
     * @param refs - Array of RecordRef objects containing the CIDs to delete
     * @returns Promise that resolves when the deletion is complete
     *
     * @throws {Error} If the gRPC call fails or the delete operation fails
     *
     * @example
     * ```typescript
     * const refs = [new models.core_v1.RecordRef({cid: "QmExample123"})];
     * await client.delete(refs);
     * ```
     */
    async delete(refs) {
        await this.storeClient.delete(this.requestGenerator(refs));
    }
    /**
     * Sign a record with a cryptographic signature.
     *
     * Creates a cryptographic signature for a record using either a private
     * key or OIDC-based signing. The signing process uses the external dirctl
     * command-line tool to perform the actual cryptographic operations.
     *
     * @param req - SignRequest containing the record reference and signing provider
     *              configuration. The provider can specify either key-based signing
     *              (with a private key) or OIDC-based signing
     * @param oidc_client_id - OIDC client identifier for OIDC-based signing. Defaults to "sigstore"
     * @returns SignResponse containing the signature
     *
     * @throws {Error} If the signing operation fails or unsupported provider is supplied
     *
     * @example
     * ```typescript
     * const req = new models.sign_v1.SignRequest({
     *   recordRef: new models.core_v1.RecordRef({cid: "QmExample123"}),
     *   provider: new models.sign_v1.SignProvider({key: keyConfig})
     * });
     * const response = client.sign(req);
     * console.log(`Signature: ${response.signature}`);
     * ```
     */
    sign(req, oidc_client_id = 'sigstore') {
        var _a, _b, _c;
        var output;
        switch ((_a = req.provider) === null || _a === void 0 ? void 0 : _a.request.case) {
            case 'oidc':
                output = this.__sign_with_oidc(((_b = req.recordRef) === null || _b === void 0 ? void 0 : _b.cid) || '', req.provider.request.value, oidc_client_id);
                break;
            case 'key':
                output = this.__sign_with_key(((_c = req.recordRef) === null || _c === void 0 ? void 0 : _c.cid) || '', req.provider.request.value);
                break;
            default:
                throw new Error('unsupported provider was supplied');
        }
        if (output.status !== 0) {
            throw output.error;
        }
    }
    /**
     * Verify a cryptographic signature on a record.
     *
     * Validates the cryptographic signature of a previously signed record
     * to ensure its authenticity and integrity. This operation verifies
     * that the record has not been tampered with since signing.
     *
     * @param request - VerifyRequest containing the record reference and verification parameters
     * @returns Promise that resolves to a VerifyResponse containing the verification result and details
     *
     * @throws {Error} If the gRPC call fails or the verification operation fails
     *
     * @example
     * ```typescript
     * const request = new models.sign_v1.VerifyRequest({
     *   recordRef: new models.core_v1.RecordRef({cid: "QmExample123"})
     * });
     * const response = await client.verify(request);
     * console.log(`Signature valid: ${response.valid}`);
     * ```
     */
    async verify(request) {
        return await this.signClient.verify(request);
    }
    /**
     * Create a new synchronization configuration.
     *
     * Creates a new sync configuration that defines how data should be
     * synchronized between different Directory servers. This allows for
     * automated data replication and consistency across multiple locations.
     *
     * @param request - CreateSyncRequest containing the sync configuration details
     *                  including source, target, and synchronization parameters
     * @returns Promise that resolves to a CreateSyncResponse containing the created sync details
     *          including the sync ID and configuration
     *
     * @throws {Error} If the gRPC call fails or the sync creation fails
     *
     * @example
     * ```typescript
     * const request = new models.store_v1.CreateSyncRequest();
     * const response = await client.create_sync(request);
     * console.log(`Created sync with ID: ${response.syncId}`);
     * ```
     */
    async create_sync(request) {
        return await this.syncClient.createSync(request);
    }
    /**
     * List existing synchronization configurations.
     *
     * Retrieves a list of all sync configurations that have been created,
     * with optional filtering and pagination support. This allows you to
     * monitor and manage multiple synchronization processes.
     *
     * @param request - ListSyncsRequest containing filtering criteria, pagination options,
     *                  and other query parameters
     * @returns Promise that resolves to an array of ListSyncsItem objects with
     *          their details including ID, name, status, and configuration parameters
     *
     * @throws {Error} If the gRPC call fails or the list operation fails
     *
     * @example
     * ```typescript
     * const request = new models.store_v1.ListSyncsRequest({limit: 10});
     * const syncs = await client.list_syncs(request);
     * for (const sync of syncs) {
     *   console.log(`Sync: ${sync}`);
     * }
     * ```
     */
    async list_syncs(request) {
        const results = [];
        for await (const response of this.syncClient.listSyncs(request)) {
            results.push(response);
        }
        return results;
    }
    /**
     * Retrieve detailed information about a specific synchronization configuration.
     *
     * Gets comprehensive details about a specific sync configuration including
     * its current status, configuration parameters, performance metrics,
     * and any recent errors or warnings.
     *
     * @param request - GetSyncRequest containing the sync ID or identifier to retrieve
     * @returns Promise that resolves to a GetSyncResponse with detailed information about the sync configuration
     *          including status, metrics, configuration, and logs
     *
     * @throws {Error} If the gRPC call fails or the get operation fails
     *
     * @example
     * ```typescript
     * const request = new models.store_v1.GetSyncRequest({syncId: "sync-123"});
     * const response = await client.get_sync(request);
     * console.log(`Sync status: ${response.status}`);
     * console.log(`Last update: ${response.lastUpdateTime}`);
     * ```
     */
    async get_sync(request) {
        return await this.syncClient.getSync(request);
    }
    /**
     * Delete a synchronization configuration.
     *
     * Permanently removes a sync configuration and stops any ongoing
     * synchronization processes. This operation cannot be undone and
     * will halt all data synchronization for the specified configuration.
     *
     * @param request - DeleteSyncRequest containing the sync ID or identifier to delete
     * @returns Promise that resolves to a DeleteSyncResponse when the deletion is complete
     *
     * @throws {Error} If the gRPC call fails or the delete operation fails
     *
     * @example
     * ```typescript
     * const request = new models.store_v1.DeleteSyncRequest({syncId: "sync-123"});
     * await client.delete_sync(request);
     * console.log("Sync deleted");
     * ```
     */
    async delete_sync(request) {
        return await this.syncClient.deleteSync(request);
    }
    /**
     * Get events from the Event API matching the specified criteria.
     *
     * Retrieves a list of events that match the filtering and query criteria
     * specified in the request.
     *
     * @param request - ListenRequest specifying filtering criteria, pagination, etc.
     * @returns Promise that resolves to an array of ListenResponse objects matching the criteria
     *
     * @throws {Error} If the gRPC call fails or the get events operation fails
     */
    listen(request) {
        return this.eventClient.listen(request);
    }
    /**
     * CreatePublication creates a new publication request that will be processed by the PublicationWorker.
     * The publication request can specify either a query, a list of specific CIDs,
     * or all records to be announced to the DHT.
     *
     * @param request - PublishRequest containing record references and queries options.
     *
     * @returns CreatePublicationResponse returns the result of creating a publication request.
     * This includes the publication ID and any relevant metadata.
     *
     * @throws {Error} If the gRPC call fails or the list operation fails
     */
    async create_publication(request) {
        return await this.publicationClient.createPublication(request);
    }
    /**
     * ListPublications returns a stream of all publication requests in the system.
     * This allows monitoring of pending, processing, and completed publication requests.
     *
     * @param request - ListPublicationsRequest contains optional filters for listing publication requests.
     *
     * @returns Promise that resolves to an array of ListPublicationsItem represents
     * a single publication request in the list response.
     * Contains publication details including ID, status, and creation timestamp.
     *
     * @throws {Error} If the gRPC call fails or the list operation fails
     */
    async list_publication(request) {
        const results = [];
        for await (const response of this.publicationClient.listPublications(request)) {
            results.push(response);
        }
        return results;
    }
    /**
     * GetPublication retrieves details of a specific publication request by its identifier.
     * This includes the current status and any associated metadata.
     *
     * @param request - GetPublicationRequest specifies which publication to retrieve by its identifier.
     *
     * @returns GetPublicationResponse contains the full details of a specific publication request.
     * Includes status, progress information, and any error details if applicable.
     *
     * @throws {Error} If the gRPC call fails or the get operation fails
     */
    async get_publication(request) {
        return await this.publicationClient.getPublication(request);
    }
    /**
     * Resolve a record name to CIDs.
     *
     * Resolves a record reference (name with optional version) to content identifiers (CIDs).
     * When no version is specified, returns all versions sorted by creation time (newest first).
     *
     * @param request - ResolveRequest containing the name and optional version
     * @returns Promise that resolves to a ResolveResponse containing the resolved record references
     *
     * @throws {Error} If the gRPC call fails or the resolve operation fails
     *
     * @example
     * ```typescript
     * import { create } from "@bufbuild/protobuf";
     *
     * // Resolve latest version
     * const request = create(models.naming_v1.ResolveRequestSchema, { name: "cisco.com/agent" });
     * const response = await client.resolve(request);
     * console.log(`Latest CID: ${response.records[0].cid}`);
     *
     * // Resolve specific version
     * const request = create(models.naming_v1.ResolveRequestSchema, { name: "cisco.com/agent", version: "v1.0.0" });
     * const response = await client.resolve(request);
     * ```
     */
    async resolve(request) {
        return await this.namingClient.resolve(request);
    }
    /**
     * Get verification info for a record.
     *
     * Retrieves the name verification status for a record. Can look up by CID directly
     * or by name (with optional version) which will be resolved first.
     *
     * @param request - GetVerificationInfoRequest containing cid, name, and/or version
     * @returns Promise that resolves to a GetVerificationInfoResponse containing verification status
     *
     * @throws {Error} If the gRPC call fails or the operation fails
     *
     * @example
     * ```typescript
     * import { create } from "@bufbuild/protobuf";
     *
     * // Check by CID
     * const request = create(models.naming_v1.GetVerificationInfoRequestSchema, { cid: "bafyreib..." });
     * const response = await client.getVerificationInfo(request);
     *
     * // Check by name (latest version)
     * const request = create(models.naming_v1.GetVerificationInfoRequestSchema, { name: "cisco.com/agent" });
     * const response = await client.getVerificationInfo(request);
     *
     * // Check by name with specific version
     * const request = create(models.naming_v1.GetVerificationInfoRequestSchema, { name: "cisco.com/agent", version: "v1.0.0" });
     * const response = await client.getVerificationInfo(request);
     * ```
     */
    async getVerificationInfo(request) {
        return await this.namingClient.getVerificationInfo(request);
    }
    /**
     * Sign a record using a private key.
     *
     * This private method handles key-based signing by writing the private key
     * to a temporary file and executing the dirctl command with the key file.
     *
     * @param cid - Content identifier of the record to sign
     * @param req - SignWithKey request containing the private key
     * @returns SignResponse containing the signature
     *
     * @throws {Error} If any error occurs during signing
     *
     * @private
     */
    __sign_with_key(cid, req) {
        const tmpDir = mkdtempSync(join(tmpdir(), 'dirctl-sign-'));
        const tmp_key_filename = join(tmpDir, 'private.key');
        try {
            // Write private key to the temporary file with secure permissions (owner read/write only)
            writeFileSync(tmp_key_filename, String(req.privateKey), { mode: 0o600 });
            // Prepare environment for command
            const shell_env = env;
            shell_env['COSIGN_PASSWORD'] = String(req.password);
            let commandArgs = ["sign", cid, "--key", tmp_key_filename];
            if (this.config.spiffeEndpointSocket !== '') {
                commandArgs.push(...["--spiffe-socket-path", this.config.spiffeEndpointSocket]);
            }
            // Execute command
            let output = spawnSync(`${this.config.dirctlPath}`, commandArgs, { env: { ...shell_env }, encoding: 'utf8', stdio: 'pipe' });
            return output;
        }
        finally {
            // Clean up: remove the temporary directory and its contents
            try {
                rmSync(tmpDir, { recursive: true, force: true });
            }
            catch (cleanupError) {
                // Log cleanup error but don't fail the operation
                console.warn(`Failed to clean up temporary directory ${tmpDir}:`, cleanupError);
            }
        }
    }
    /**
     * Sign a record using OIDC-based authentication.
     *
     * This private method handles OIDC-based signing by building the appropriate
     * dirctl command with OIDC parameters and executing it.
     *
     * @param cid - Content identifier of the record to sign
     * @param req - SignWithOIDC request containing the OIDC configuration
     * @param oidc_client_id - OIDC client identifier for authentication
     * @returns SignResponse containing the signature
     *
     * @throws {Error} If any error occurs during signing
     *
     * @private
     */
    __sign_with_oidc(cid, req, oidc_client_id) {
        var _a, _b, _c, _d;
        // Prepare command
        let commandArgs = ["sign", cid];
        if (req.idToken !== '') {
            commandArgs.push(...["--oidc-token", req.idToken]);
        }
        if (((_a = req.options) === null || _a === void 0 ? void 0 : _a.oidcProviderUrl) !== undefined &&
            req.options.oidcProviderUrl !== '') {
            commandArgs.push(...["--oidc-provider-url", req.options.oidcProviderUrl]);
        }
        if (((_b = req.options) === null || _b === void 0 ? void 0 : _b.fulcioUrl) !== undefined && req.options.fulcioUrl !== '') {
            commandArgs.push(...["--fulcio-url", req.options.fulcioUrl]);
        }
        if (((_c = req.options) === null || _c === void 0 ? void 0 : _c.rekorUrl) !== undefined && req.options.rekorUrl !== '') {
            commandArgs.push(...["--rekor-url", req.options.rekorUrl]);
        }
        if (((_d = req.options) === null || _d === void 0 ? void 0 : _d.timestampUrl) !== undefined &&
            req.options.timestampUrl !== '') {
            commandArgs.push(...["--timestamp-url", req.options.timestampUrl]);
        }
        if (this.config.spiffeEndpointSocket !== '') {
            commandArgs.push(...["--spiffe-socket-path", this.config.spiffeEndpointSocket]);
        }
        // Execute command
        let output = spawnSync(`${this.config.dirctlPath}`, commandArgs, {
            env: { ...env },
            encoding: 'utf8',
            stdio: 'pipe',
        });
        return output;
    }
}

export { Client, Config, index as models };
//# sourceMappingURL=index.mjs.map
