resource "cryptflare_secret" "database_url" {
  workspace_id   = cryptflare_workspace.backend.id
  environment_id = cryptflare_environment.production.id
  key            = "DATABASE_URL"
  value          = var.database_url
}
