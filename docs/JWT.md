# JWT Package

Trux signs and verifies HS256 JSON Web Tokens through libjwt 3 when a program imports `jwt`.

```trux
import "jwt"

jwt.sign(claims string, secret string) (string, error)
jwt.verify(token string, secret string) (*jwt.Claims, error)
jwt.getString(claims *jwt.Claims, key string) string
jwt.getInt(claims *jwt.Claims, key string) int
jwt.getBool(claims *jwt.Claims, key string) bool
jwt.getFloat(claims *jwt.Claims, key string) float
```

`claims` is a JSON object. `jwt.sign` does not add an `iat` claim unless that field is already in the object. `jwt.verify` accepts only HS256. A token with `alg` `none`, or a signature that does not match `secret`, returns an error and does not return claims.

A secret shorter than 32 bytes is accepted. HMAC-SHA256 pads it with zero bytes, which is the same key OpenSSL uses for that secret.

The package links libjwt, jansson, and OpenSSL. It does not vendor those libraries. Set `TRUX_LIBJWT_PREFIX` to a prefix that contains `include/jwt.h` and `lib/libjwt.a` when they are not on the default compiler search path. On Windows, `libcrypto`, `libssl`, and `libjansson` DLLs from that prefix are copied next to the executable.

`jwt` and `gpu` cannot be combined in one program yet. Claim values are read back with yyjson, the same JSON library used by `import "json"`.
