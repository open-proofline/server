# Historical Prompt: Initial Docker Support

This prompt is archived.

Current Docker behaviour should use plural bind variables:

- `SAFE_PRIVATE_BIND_ADDRS`
- `SAFE_PUBLIC_BIND_ADDRS`

Container defaults should bind inside the container, usually:

```env
SAFE_PRIVATE_BIND_ADDRS=0.0.0.0:8080
SAFE_PUBLIC_BIND_ADDRS=0.0.0.0:8081
```

Restrict exposure using Docker port publishing, firewall rules, WireGuard, or a reverse proxy.

Do not use this historical prompt as-is.
