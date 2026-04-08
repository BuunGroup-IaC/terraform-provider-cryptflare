// Copyright (c) 2026 Buun Group
// SPDX-License-Identifier: MPL-2.0

package secret

import (
	"context"
	"fmt"
	"strings"

	"github.com/buun-group/terraform-provider-cryptflare/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &Resource{}
	_ resource.ResourceWithImportState = &Resource{}
)

// Resource implements the cryptflare_secret resource.
type Resource struct {
	client *client.Client
}

// Model maps the Terraform schema to Go types.
type Model struct {
	ID            types.String `tfsdk:"id"`
	WorkspaceID   types.String `tfsdk:"workspace_id"`
	EnvironmentID types.String `tfsdk:"environment_id"`
	Key           types.String `tfsdk:"key"`
	Value         types.String `tfsdk:"value"`
	Version       types.Int64  `tfsdk:"version"`
	PodID         types.String `tfsdk:"pod_id"`
}

// NewResource returns a new secret resource constructor.
func NewResource() resource.Resource {
	return &Resource{}
}

func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret"
}

func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an encrypted secret in a CryptFlare environment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true, Description: "Composite ID (workspace_id/environment_id/key).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"workspace_id": schema.StringAttribute{
				Required: true, Description: "Workspace ID or slug.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"environment_id": schema.StringAttribute{
				Required: true, Description: "Environment ID or slug.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"key": schema.StringAttribute{
				Required: true, Description: "Secret key name (UPPER_SNAKE_CASE).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"value": schema.StringAttribute{
				Required: true, Sensitive: true,
				Description: "Secret value. Encrypted with AES-256-GCM before storage.",
			},
			"version": schema.Int64Attribute{
				Computed: true, Description: "Current version number.",
			},
			"pod_id": schema.StringAttribute{
				Optional: true, Description: "Pod ID to organize this secret into.",
			},
		},
	}
}

func (r *Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected *client.Client, got %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.CreateSecretInput{
		Key:   data.Key.ValueString(),
		Value: data.Value.ValueString(),
	}
	if !data.PodID.IsNull() && !data.PodID.IsUnknown() {
		podID := data.PodID.ValueString()
		input.PodID = &podID
	}

	result, err := r.client.CreateSecret(ctx, data.WorkspaceID.ValueString(), data.EnvironmentID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating secret", err.Error())
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s/%s", data.WorkspaceID.ValueString(), data.EnvironmentID.ValueString(), data.Key.ValueString()))
	data.Version = types.Int64Value(int64(result.Version))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data Model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sv, err := r.client.GetSecret(ctx, data.WorkspaceID.ValueString(), data.EnvironmentID.ValueString(), data.Key.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading secret", err.Error())
		return
	}

	data.Value = types.StringValue(sv.Value)
	data.Version = types.Int64Value(int64(sv.Version))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Value changed — rotate the secret
	if !plan.Value.Equal(state.Value) {
		result, err := r.client.RotateSecret(ctx, plan.WorkspaceID.ValueString(), plan.EnvironmentID.ValueString(), plan.Key.ValueString(), plan.Value.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error rotating secret", err.Error())
			return
		}
		plan.Version = types.Int64Value(int64(result.Version))
	}

	plan.ID = state.ID
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data Model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSecret(ctx, data.WorkspaceID.ValueString(), data.EnvironmentID.ValueString(), data.Key.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting secret", err.Error())
	}
}

// ImportState supports importing by "workspace_id/environment_id/key".
func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: workspace_id/environment_id/key")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("key"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
