// Authors:
//   Pavel Aksenov <41126916+al-bashkir@users.noreply.github.com>
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
var _ resource.ResourceWithImportState = &dnsServer{}

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
	var data dnsServerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Create freeipa dns server %s", data.ServerName.ValueString()))

	all := true
	if _, err := r.client.DnsserverShow(
		&ipa.DnsserverShowArgs{Idnsserverid: data.ServerName.ValueString()},
		&ipa.DnsserverShowOptionalArgs{All: &all},
	); err != nil {
		if strings.Contains(err.Error(), "NotFound") {
			resp.Diagnostics.AddError(
				"DNS server not found",
				fmt.Sprintf("Run ipa-dns-install on '%s' before managing it via Terraform", data.ServerName.ValueString()),
			)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Error verifying freeipa dns server existence: %s", err))
		return
	}

	optArgs, hasChange := r.buildModOptArgs(ctx, &data, nil)
	if hasChange {
		args := ipa.DnsserverModArgs{Idnsserverid: data.ServerName.ValueString()}
		if _, err := r.client.DnsserverMod(&args, optArgs); err != nil {
			if strings.Contains(err.Error(), "EmptyModlist") {
				resp.Diagnostics.AddWarning("Client Warning", err.Error())
			} else {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Error applying freeipa dns server config: %s", err))
				return
			}
		}
	}

	if _, err := r.readServer(ctx, &data, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Error refreshing freeipa dns server after create: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
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
	var data, state dnsServerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Update freeipa dns server %s", data.ServerName.ValueString()))

	optArgs, hasChange := r.buildModOptArgs(ctx, &data, &state)
	if hasChange {
		args := ipa.DnsserverModArgs{Idnsserverid: data.ServerName.ValueString()}
		if _, err := r.client.DnsserverMod(&args, optArgs); err != nil {
			if strings.Contains(err.Error(), "EmptyModlist") {
				resp.Diagnostics.AddWarning("Client Warning", err.Error())
			} else {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Error updating freeipa dns server: %s", err))
				return
			}
		}
	}

	if _, err := r.readServer(ctx, &data, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Error refreshing freeipa dns server after update: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *dnsServer) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data dnsServerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Delete freeipa dns server %s — clearing managed attrs", data.ServerName.ValueString()))

	delAttrs := []string{}
	if !data.SOAMnameOverride.IsNull() {
		delAttrs = append(delAttrs, "idnssoamname")
	}
	if !data.Forwarders.IsNull() {
		delAttrs = append(delAttrs, "idnsforwarders")
	}
	if !data.ForwardPolicy.IsNull() {
		delAttrs = append(delAttrs, "idnsforwardpolicy")
	}
	if len(delAttrs) == 0 {
		return
	}

	optArgs := ipa.DnsserverModOptionalArgs{Delattr: &delAttrs}
	args := ipa.DnsserverModArgs{Idnsserverid: data.ServerName.ValueString()}
	if _, err := r.client.DnsserverMod(&args, &optArgs); err != nil {
		if strings.Contains(err.Error(), "EmptyModlist") {
			resp.Diagnostics.AddWarning("Client Warning", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Error clearing freeipa dns server config on delete: %s", err))
	}
}

func (r *dnsServer) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_name"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *dnsServer) buildModOptArgs(ctx context.Context, data *dnsServerModel, prev *dnsServerModel) (*ipa.DnsserverModOptionalArgs, bool) {
	optArgs := ipa.DnsserverModOptionalArgs{}
	hasChange := false
	delAttrs := []string{}

	if prev == nil || !data.SOAMnameOverride.Equal(prev.SOAMnameOverride) {
		if data.SOAMnameOverride.IsNull() {
			delAttrs = append(delAttrs, "idnssoamname")
		} else {
			var v interface{} = data.SOAMnameOverride.ValueString()
			optArgs.Idnssoamname = &v
		}
		hasChange = true
	}

	if prev == nil || !data.Forwarders.Equal(prev.Forwarders) {
		if data.Forwarders.IsNull() {
			delAttrs = append(delAttrs, "idnsforwarders")
		} else {
			var v []string
			for _, value := range data.Forwarders.Elements() {
				val, _ := strconv.Unquote(value.String())
				v = append(v, val)
			}
			if v == nil {
				v = []string{}
			}
			optArgs.Idnsforwarders = &v
		}
		hasChange = true
	}

	if prev == nil || !data.ForwardPolicy.Equal(prev.ForwardPolicy) {
		if !data.ForwardPolicy.IsNull() {
			s := data.ForwardPolicy.ValueString()
			optArgs.Idnsforwardpolicy = &s
		}
		hasChange = true
	}

	if len(delAttrs) > 0 {
		optArgs.Delattr = &delAttrs
	}
	return &optArgs, hasChange
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
