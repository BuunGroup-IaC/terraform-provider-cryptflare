// Copyright (c) 2026 Buun Group
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"log"

	"github.com/buun-group/terraform-provider-cryptflare/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

var version = "dev"

func main() {
	err := providerserver.Serve(
		context.Background(),
		provider.New(version),
		providerserver.ServeOpts{
			Address: "registry.terraform.io/buun-group/cryptflare",
		},
	)
	if err != nil {
		log.Fatal(err)
	}
}
