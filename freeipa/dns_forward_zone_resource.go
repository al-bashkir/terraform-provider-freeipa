// Authors:
//   Pavel Aksenov <aksenov.pavel.v@gmail.com>
//
// SPDX-License-Identifier: GPL-3.0-only

package freeipa

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	ipa "github.com/infra-monkey/go-freeipa/freeipa"
)

var _ resource.Resource = &dnsForwardZone{}

type dnsForwardZone struct {
	client *ipa.Client
}

type dnsForwardZoneModel struct {
	Id               types.String `tfsdk:"id"`
	ZoneName         types.String `tfsdk:"zone_name"`
	Forwarders       types.List   `tfsdk:"forwarders"`
	ForwardPolicy    types.String `tfsdk:"forward_policy"`
	DisableZone      types.Bool   `tfsdk:"disable_zone"`
	SkipOverlapCheck types.Bool   `tfsdk:"skip_overlap_check"`
	ComputedZoneName types.String `tfsdk:"computed_zone_name"`
}

func NewDNSForwardZoneResource() resource.Resource {
	return &dnsForwardZone{}
}

func (r *dnsForwardZone) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_forward_zone"
}

func (r *dnsForwardZone) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*ipa.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ipa.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = client
}

func (r *dnsForwardZone) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "FreeIPA DNS forward zone resource. Manages per-zone forwarders, forward policy, and enable/disable state.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID of the resource (canonical zone name with trailing dot).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"zone_name": schema.StringAttribute{
				MarkdownDescription: "Forward zone name (FQDN).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"forwarders": schema.ListAttribute{
				MarkdownDescription: "Per-zone forwarders. A custom port can be specified using a standard format `IP_ADDRESS port PORT`.",
				Required:            true,
				ElementType:         types.StringType,
			},
			"forward_policy": schema.StringAttribute{
				MarkdownDescription: "Per-zone conditional forwarding policy. One of `only`, `first`, `none`. Defaults to `first`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("first"),
				Validators: []validator.String{
					stringvalidator.OneOf("only", "first", "none"),
				},
			},
			"disable_zone": schema.BoolAttribute{
				MarkdownDescription: "Disable the forward zone (inverse of FreeIPA `idnszoneactive`). Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"skip_overlap_check": schema.BoolAttribute{
				MarkdownDescription: "Force creation even if the zone overlaps with an existing zone. Create-time only; ignored on update. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"computed_zone_name": schema.StringAttribute{
				MarkdownDescription: "Canonical zone name returned by FreeIPA (with trailing dot).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *dnsForwardZone) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
}

func (r *dnsForwardZone) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data dnsForwardZoneModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.readZone(ctx, &data, &resp.Diagnostics); err != nil {
		if strings.Contains(err.Error(), "NotFound") {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] dns forward zone %s not found, removing from state", data.ZoneName.ValueString()))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Error reading freeipa dns forward zone: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *dnsForwardZone) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}
func (r *dnsForwardZone) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}

func (r *dnsForwardZone) readZone(ctx context.Context, data *dnsForwardZoneModel, diags *diag.Diagnostics) (*ipa.Dnsforwardzone, error) {
	all := true
	var name interface{} = data.Id.ValueString()
	if data.Id.IsNull() || data.Id.ValueString() == "" {
		name = data.ZoneName.ValueString()
	}
	res, err := r.client.DnsforwardzoneShow(
		&ipa.DnsforwardzoneShowArgs{},
		&ipa.DnsforwardzoneShowOptionalArgs{Idnsname: &name, All: &all},
	)
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Read freeipa dns forward zone %s: %s", data.ZoneName.ValueString(), res.String()))
	z := &res.Result

	if z.Idnsname != nil {
		dnsnames := z.Idnsname.([]interface{})
		dnsname := dnsnames[0].(map[string]interface{})["__dns_name__"].(string)
		data.ComputedZoneName = types.StringValue(dnsname)
		data.Id = types.StringValue(dnsname)
	}

	if z.Idnsforwarders != nil {
		v, d := types.ListValueFrom(ctx, types.StringType, *z.Idnsforwarders)
		diags.Append(d...)
		data.Forwarders = v
	} else {
		data.Forwarders = types.ListValueMust(types.StringType, []attr.Value{})
	}

	if z.Idnsforwardpolicy != nil {
		data.ForwardPolicy = types.StringValue(*z.Idnsforwardpolicy)
	}

	if z.Idnszoneactive != nil {
		data.DisableZone = types.BoolValue(!*z.Idnszoneactive)
	}

	return z, nil
}

// suppress unused-import warnings until Task 9 lands Create/Update/Delete/Import.
var (
	_ = strconv.Quote
	_ = path.Root
)
