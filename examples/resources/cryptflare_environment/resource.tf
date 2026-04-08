resource "cryptflare_environment" "production" {
  workspace_id = cryptflare_workspace.backend.id
  name         = "Production"
  slug         = "production"
}
