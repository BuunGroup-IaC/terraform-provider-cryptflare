<p align="center">
  <img src="assets/logo.png" width="80" alt="CryptFlare" />
</p>

<h1 align="center">Terraform Provider for CryptFlare</h1>

<p align="center">
  Manage secrets, workspaces, environments, and pods as infrastructure-as-code.
</p>

<p align="center">
  <a href="https://registry.terraform.io/providers/buun-group/cryptflare/latest/docs">Registry</a> |
  <a href="https://docs.cryptflare.com">Documentation</a> |
  <a href="https://github.com/buun-group/terraform-provider-cryptflare/issues">Issues</a>
</p>

---

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.13
- [Go](https://golang.org/doc/install) >= 1.23 (for building from source)

## Installation

```hcl
terraform {
  required_providers {
    cryptflare = {
      source  = "buun-group/cryptflare"
      version = "~> 0.1"
    }
  }
}

provider "cryptflare" {
  # Set via CF_TOKEN and CF_ORG environment variables, or configure here:
  # api_token = "cf_live_..."
  # org_id    = "org_..."
}
```

## Authentication

The provider needs an API token and organisation ID.

```bash
# Recommended: environment variables
export CF_TOKEN=cf_live_...
export CF_ORG=org_...
```

Or configure directly in the provider block (not recommended for version control):

```hcl
provider "cryptflare" {
  api_token = var.cryptflare_token
  org_id    = var.cryptflare_org
}
```

## Usage

### Create a workspace with environments and secrets

```hcl
resource "cryptflare_workspace" "backend" {
  name = "Backend API"
  slug = "backend-api"
}

resource "cryptflare_environment" "production" {
  workspace_id = cryptflare_workspace.backend.id
  name         = "Production"
  slug         = "production"
}

resource "cryptflare_secret" "database_url" {
  workspace_id   = cryptflare_workspace.backend.id
  environment_id = cryptflare_environment.production.id
  key            = "DATABASE_URL"
  value          = var.database_url
}
```

### Organize secrets with pods

```hcl
resource "cryptflare_pod" "databases" {
  workspace_id   = cryptflare_workspace.backend.id
  environment_id = cryptflare_environment.production.id
  name           = "Databases"
  slug           = "databases"
}

resource "cryptflare_secret" "redis_url" {
  workspace_id   = cryptflare_workspace.backend.id
  environment_id = cryptflare_environment.production.id
  key            = "REDIS_URL"
  value          = var.redis_url
  pod_id         = cryptflare_pod.databases.id
}
```

## Resources

| Resource | Description |
|---|---|
| `cryptflare_workspace` | Manages a workspace within an organisation |
| `cryptflare_environment` | Manages an environment within a workspace |
| `cryptflare_secret` | Manages an encrypted secret in an environment |
| `cryptflare_pod` | Manages a pod (folder) for organizing secrets |

## Import

All resources support `terraform import`:

```bash
# Workspace
terraform import cryptflare_workspace.example ws_abc123

# Environment (workspace_id/env_id)
terraform import cryptflare_environment.example ws_abc123/env_def456

# Secret (workspace_id/env_id/key)
terraform import cryptflare_secret.example ws_abc123/env_def456/DATABASE_URL

# Pod (workspace_id/env_id/pod_id)
terraform import cryptflare_pod.example ws_abc123/env_def456/pod_ghi789
```

## Development

```bash
# Build
make build

# Run tests
make test

# Run acceptance tests (requires API access)
export CRYPTFLARE_API_TOKEN=cf_live_...
export CRYPTFLARE_ORG_ID=org_...
make testacc

# Generate docs
make generate

# Lint
make lint
```

### Local development

To test against a local CryptFlare API:

```bash
export CF_API_URL=http://localhost:5488
export CF_TOKEN=cf_live_...
export CF_ORG=org_...
make testacc
```

## License

[MPL-2.0](LICENSE)
