terraform {
  required_providers {
    cryptflare = {
      source  = "buun-group/cryptflare"
      version = "~> 0.1"
    }
  }
}

provider "cryptflare" {
  # api_token = var.cryptflare_token  # Or set CF_TOKEN env var
  # org_id    = var.cryptflare_org    # Or set CF_ORG env var
}
