# X API Package

This package handles posting tweets to X (formerly Twitter) using the X API v2 with OAuth 1.0a authentication.

https://docs.x.com/x-api/introduction

## Troubleshooting

```
{"detail":"You are not permitted to perform this action.","type":"about:blank","title":"Forbidden","status":403}
```

Possible cause: Tweet exceeds 280 characters for non-verified accounts. https://docs.x.com/fundamentals/counting-characters#definition-of-a-character

**429 Too Many Requests** -

```
{"title":"Too Many Requests","detail":"Too Many Requests","type":"about:blank","status":429}
```

Rate limit reached (17 POST requests per 24 hours). https://developer.x.com/en/portal/products
