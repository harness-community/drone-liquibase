---
name: sync-docs
description: Update AGENTS.md when plugin code changes
---

# Sync Documentation

Update AGENTS.md to reflect code changes in the Go plugin.

## When to Run

- After modifying `plugin/*.go` files
- After adding/removing environment variables
- After changing Docker image structure
- After modifying certificate handling logic

## Key Sections to Update

### Repository Structure
Update file tree if new Go files are added:
```
├── main.go
├── plugin/
│   ├── plugin.go
│   ├── args.go
│   ├── command_builder.go
│   ├── cert_manager.go
│   ├── kerberos.go
│   ├── global_options.go
│   └── exec.go
```

### Key Components
Update package descriptions if functionality changes:
- `plugin/plugin.go` - Main execution flow
- `plugin/command_builder.go` - CLI argument construction
- `plugin/cert_manager.go` - SSL/TLS handling
- `plugin/kerberos.go` - Kerberos auth

### Environment Variables
Update tables if new env vars are added to `plugin/args.go`:
- Required variables
- Database connection variables
- Harness integration variables
- SSL/TLS configuration

### Certificate Handling
Update if `cert_manager.go` logic changes:
- TrustStore setup
- KeyStore setup
- JAVA_OPTS configuration

## Checklist

- [ ] Repository structure matches actual files
- [ ] Key Components describes current packages
- [ ] Environment variable tables match `args.go`
- [ ] Certificate handling matches `cert_manager.go`
- [ ] Debugging tips are accurate
