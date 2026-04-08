// Copyright (c) 2026 Buun Group
// SPDX-License-Identifier: MPL-2.0

package pod

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

// Resource implements the cryptflare_pod resource.
type Resource struct {
	client *client.Client
}

// Model maps the Terraform schema to Go types.
type Model struct {
	ID            types.String `tfsdk:"id"`
	WorkspaceID   types.String `tfsdk:"workspace_id"`
	EnvironmentID types.String `tfsdk:"environment_id"`
	ParentID      types.String `tfsdk:"parent_id"`
	Name          types.String `tfsdk:"name"`
	Slug          types.String `tfsdk:"slug"`
	Description   types.String `tfsdk:"description"`
	CreatedAt     types.String `tfsdk:"created_at"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
}

// NewResource returns a new pod resource constructor.
func NewResource() resource.Resource {
	return &Resource{}
}

func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pod"
}

func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a CryptFlare pod (folder) for organizing secrets within an environment. Supports up to 5 levels of nesting.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true, Description: "Pod ID.",
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
			"parent_id": schema.StringAttribute{
				Optional: true, Description: "Parent pod ID for nesting. Omit for root-level pods.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required: true, Description: "Display name of the pod.",
			},
			"slug": schema.StringAttribute{
				Required: true, Description: "URL-safe slug (lowercase + hyphens). Unique per parent level.",
			},
			"description": schema.StringAttribute{
				Optional: true, Description: "Optional description.",
			},
			"created_at": schema.StringAttribute{
				Computed: true, Description: "ISO 8601 creation timestamp.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Computed: true, Description: "ISO 8601 last update timestamp.",
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

	input := client.CreatePodInput{
		Name: data.Name.ValueString(),
		Slug: data.Slug.ValueString(),
	}
	if !data.ParentID.IsNull() && !data.ParentID.IsUnknown() {
		parentID := data.ParentID.ValueString()
		input.ParentID = &parentID
	}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		desc := data.Description.ValueString()
		input.Description = &desc
	}

	pod, err := r.client.CreatePod(ctx, data.WorkspaceID.ValueString(), data.EnvironmentID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating pod", err.Error())
		return
	}

	data.ID = types.StringValue(pod.ID)
	data.CreatedAt = types.StringValue(pod.CreatedAt)
	data.UpdatedAt = types.StringValue(pod.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data Model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pod, err := r.client.GetPod(ctx, data.WorkspaceID.ValueString(), data.EnvironmentID.ValueString(), data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading pod", err.Error())
		return
	}

	data.Name = types.StringValue(pod.Name)
	data.Slug = types.StringValue(pod.Slug)
	if pod.Description != nil {
		data.Description = types.StringValue(*pod.Description)
	} else {
		data.Description = types.StringNull()
	}
	if pod.ParentID != nil {
		data.ParentID = types.StringValue(*pod.ParentID)
	} else {
		data.ParentID = types.StringNull()
	}
	data.CreatedAt = types.StringValue(pod.CreatedAt)
	data.UpdatedAt = types.StringValue(pod.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.UpdatePodInput{}
	name := plan.Name.ValueString()
	input.Name = &name
	slug := plan.Slug.ValueString()
	input.Slug = &slug
	if !plan.Description.IsNull() {
		desc := plan.Description.ValueString()
		input.Description = &desc
	}

	pod, err := r.client.UpdatePod(ctx, plan.WorkspaceID.ValueString(), plan.EnvironmentID.ValueString(), plan.ID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating pod", err.Error())
		return
	}

	plan.UpdatedAt = types.StringValue(pod.UpdatedAt)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data Model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeletePod(ctx, data.WorkspaceID.ValueString(), data.EnvironmentID.ValueString(), data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting pod", err.Error())
	}
}

// ImportState supports importing by "workspace_id/environment_id/pod_id".
func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: workspace_id/environment_id/pod_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}
