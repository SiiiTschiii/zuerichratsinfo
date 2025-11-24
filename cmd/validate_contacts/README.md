# Validate Contacts Script

This script validates the `data/contacts.yaml` file to ensure:

1. **Valid YAML syntax** - The file must be parseable YAML
2. **Supported platforms** - Only allowed platforms: x, facebook, instagram, linkedin, bluesky, tiktok
3. **Valid URLs** - All URLs must:
   - Use http or https scheme
   - Match the expected domain for their platform
   - Be properly formatted

## Usage

### Manual validation

Run the validation script directly:

```bash
go run cmd/validate_contacts/main.go data/contacts.yaml
```

Or use the Makefile:

```bash
make validate-contacts
```

### Exit codes

- `0` - Validation successful
- `1` - Validation failed (errors will be printed to stdout)

## Examples

### Successful validation

```bash
$ go run cmd/validate_contacts/main.go data/contacts.yaml
✅ Validation successful! contacts.yaml is valid.
```

### Failed validation

```bash
$ go run cmd/validate_contacts/main.go data/contacts.yaml
❌ Validation failed with 2 error(s):

1. Contact 'John Doe', platform 'x', URL 'www.x.com/johndoe': URL must use http or https scheme, got:
2. Contact 'Jane Smith', platform 'facebook', URL 'https://twitter.com/janesmith': URL domain 'twitter.com' does not match platform 'facebook' (expected one of: [facebook.com www.facebook.com])
```

## Validation rules

### Supported platforms

- `x` (formerly Twitter) - URLs must be from x.com or twitter.com
- `facebook` - URLs must be from facebook.com or www.facebook.com
- `instagram` - URLs must be from instagram.com or www.instagram.com
- `linkedin` - URLs must be from linkedin.com or www.linkedin.com
- `bluesky` - URLs must be from bsky.app or web-cdn.bsky.app
- `tiktok` - URLs must be from tiktok.com or www.tiktok.com

### URL requirements

- All URLs must start with `http://` or `https://`
- URLs must match the platform they're listed under
- URLs cannot be empty strings

### Contact requirements

- Each contact must have a unique name
- Contact names cannot be empty
- At least the `version` field must be present in the file
- At least one contact must be defined

## GitHub Actions

The validation runs automatically on:

- Push to `main` branch (when contacts.yaml or validation code changes)
- Pull requests to `main` branch
- Manual workflow dispatch

See `.github/workflows/validate-contacts.yml` for the workflow configuration.
