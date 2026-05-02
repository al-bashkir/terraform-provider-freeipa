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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	ipa "github.com/infra-monkey/go-freeipa/freeipa"
)

var _ resource.Resource = &dnsServer{}

type dnsServer struct {
	client *ipa.Client
}

type dnsServerModel struct {
	Id               types.String `tfsdk:"id"`
	ServerName       types.String `tfsdk:"server_name"`
	SOAMnameOverride types.String `tfsdk:"soa_mname_override"`
	Forwarders       types.List   `tfsdk:"forwarders"`
	ForwardPolicy    types.String `tfsdk:"forward_policy"`
}

func NewDNSServerResource() resource.Resource {
	return &dnsServer{}
}

func (r *dnsServer) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_server"
}

func (r *dnsServer) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *dnsServer) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "FreeIPA per-server DNS configuration. The `dnsserver` row must already exist in FreeIPA (created automatically by `ipa-dns-install`). This resource manages per-replica forwarders, forward policy, and SOA mname override in-place; Delete clears those managed attrs.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID of the resource (= server FQDN).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"server_name": schema.StringAttribute{
				MarkdownDescription: "DNS server name (FQDN of the IPA replica running named).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"soa_mname_override": schema.StringAttribute{
				MarkdownDescription: "SOA mname (authoritative server) override for zones served by this replica.",
				Optional:            true,
			},
			"forwarders": schema.ListAttribute{
				MarkdownDescription: "Per-server forwarders. A custom port can be specified using a standard format `IP_ADDRESS port PORT`.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"forward_policy": schema.StringAttribute{
				MarkdownDescription: "Per-server conditional forwarding policy. One of `only`, `first`, `none`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("only", "first", "none"),
				},
			},
		},
	}
}

func (r *dnsServer) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
}
func (r *dnsServer) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data dnsServerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.readServer(ctx, &data, &resp.Diagnostics); err != nil {
		if strings.Contains(err.Error(), "NotFound") {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] dns server %s not found, removing from state", data.ServerName.ValueString()))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Error reading freeipa dns server: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
func (r *dnsServer) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}
func (r *dnsServer) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}

func (r *dnsServer) readServer(ctx context.Context, data *dnsServerModel, diags *diag.Diagnostics) (*ipa.Dnsserver, error) {
	all := true
	args := ipa.DnsserverShowArgs{Idnsserverid: data.ServerName.ValueString()}
	res, err := r.client.DnsserverShow(&args, &ipa.DnsserverShowOptionalArgs{All: &all})
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Read freeipa dns server %s: %s", data.ServerName.ValueString(), res.String()))
	srv := &res.Result

	data.Id = types.StringValue(srv.Idnsserverid)
	data.ServerName = types.StringValue(srv.Idnsserverid)

	if !data.SOAMnameOverride.IsNull() {
		if srv.Idnssoamname != nil {
			if v, ok := (*srv.Idnssoamname).(string); ok {
				data.SOAMnameOverride = types.StringValue(v)
			} else {
				data.SOAMnameOverride = types.StringNull()
			}
		} else {
			data.SOAMnameOverride = types.StringNull()
		}
	}

	if !data.Forwarders.IsNull() {
		if srv.Idnsforwarders != nil {
			v, d := types.ListValueFrom(ctx, types.StringType, *srv.Idnsforwarders)
			diags.Append(d...)
			data.Forwarders = v
		} else {
			data.Forwarders = types.ListValueMust(types.StringType, []attr.Value{})
		}
	}

	if !data.ForwardPolicy.IsNull() {
		if srv.Idnsforwardpolicy != nil {
			data.ForwardPolicy = types.StringValue(*srv.Idnsforwardpolicy)
		} else {
			data.ForwardPolicy = types.StringNull()
		}
	}

	return srv, nil
}

// suppress unused-import warnings until Task 6 lands buildModOptArgs/Import.
var (
	_ = strconv.Quote
	_ = path.Root
)
