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

HS256 secrets must be at least 32 bytes. libjwt rejects shorter HMAC keys.

The package links libjwt, jansson, and OpenSSL. It does not vendor those libraries. Set `TRUX_LIBJWT_PREFIX` to a prefix that contains `include/jwt.h` and `lib/libjwt.a` when they are not on the default compiler search path.

A program can import `jwt` and declare a `kernel func` in the same file. `nvcc` compiles that program and links the same libjwt, jansson, and OpenSSL libraries. Claim values are read back with yyjson, the same JSON library used by `import "json"`.
