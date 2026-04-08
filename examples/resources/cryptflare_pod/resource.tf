resource "cryptflare_pod" "databases" {
  workspace_id   = cryptflare_workspace.backend.id
  environment_id = cryptflare_environment.production.id
  name           = "Databases"
  slug           = "databases"
  description    = "Database connection strings"
}

# Nested pod
resource "cryptflare_pod" "postgres" {
  workspace_id   = cryptflare_workspace.backend.id
  environment_id = cryptflare_environment.production.id
  parent_id      = cryptflare_pod.databases.id
  name           = "PostgreSQL"
  slug           = "postgres"
}
