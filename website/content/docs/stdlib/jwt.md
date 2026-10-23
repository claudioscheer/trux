---
title: "jwt"
description: "Sign and verify HS256 JSON Web Tokens with libjwt."
icon: "key"
weight: 370
---

```trux
import "jwt"
```

**What it is.** HS256 JSON Web Tokens for API authentication. Signing and verification go through [libjwt](https://github.com/benmcollins/libjwt) 3. Claim objects are parsed with the same yyjson runtime as [`json`](/docs/stdlib/json/).

**Why.** HTTP handlers need to issue and check bearer tokens without a hand-rolled HMAC. The package pins the algorithm to HS256, so a token that says `alg` is `none` is rejected.

**When you don't need it.** Programs that do not authenticate requests should not import `jwt`. Importing it links libjwt, jansson, and OpenSSL.

---

## Build requirement

`jwt` is the one standard package that needs libraries outside the Trux tree:

- libjwt 3
- jansson
- OpenSSL 3

Point `TRUX_LIBJWT_PREFIX` at a directory that contains `include/jwt.h` and `lib/libjwt.a` when those are not on the compiler's default search path. On Windows, shared OpenSSL and jansson DLLs next to that prefix are copied beside the executable.

An HS256 secret must be at least 32 bytes. A shorter secret fails `jwt.sign` and `jwt.verify`.

---

## API Reference

| Function | Returns | Description |
| :--- | :--- | :--- |
| `jwt.sign(claims string, secret string)` | `(string, error)` | Signs a JSON object with HS256. `claims` must be a JSON object. Issued-at is not added unless it is already in that object. |
| `jwt.verify(token string, secret string)` | `(*jwt.Claims, error)` | Checks the HS256 signature and returns the claims. A bad signature returns an error and no claims. |
| `jwt.getString(claims *jwt.Claims, key string)` | `string` | Reads a string claim, or `""` when the claim is missing. |
| `jwt.getInt(claims *jwt.Claims, key string)` | `int` | Reads an integer claim, or `0` when the claim is missing. |
| `jwt.getBool(claims *jwt.Claims, key string)` | `bool` | Reads a boolean claim, or `false` when the claim is missing. |
| `jwt.getFloat(claims *jwt.Claims, key string)` | `float` | Reads a numeric claim, or `0.0` when the claim is missing. |

`jwt.Claims` is created only by `jwt.verify`. A struct literal is rejected.

---

## Example

```trux
package main

import "jwt"

func main() int {
    let claims string = "{\"sub\":\"ada\",\"admin\":true}"
    let secret string = "0123456789abcdef0123456789abcdef"
    let token string, err error = jwt.sign(claims, secret)
    if err != nil {
        return 1
    }
    let parsed *jwt.Claims, verr error = jwt.verify(token, secret)
    if verr != nil {
        return 1
    }
    print(jwt.getString(parsed, "sub"))
    return 0
}
```
