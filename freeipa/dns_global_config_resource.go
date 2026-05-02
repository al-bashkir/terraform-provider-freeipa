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

var _ resource.Resource = &dnsGlobalConfig{}

type dnsGlobalConfig struct {
	client *ipa.Client
}

type dnsGlobalConfigModel struct {
	Id              types.String `tfsdk:"id"`
	Forwarders      types.List   `tfsdk:"forwarders"`
	ForwardPolicy   types.String `tfsdk:"forward_policy"`
	AllowSyncPTR    types.Bool   `tfsdk:"allow_sync_ptr"`
	ZoneRefresh     types.Int64  `tfsdk:"zone_refresh"`
	DNSServers      types.List   `tfsdk:"dns_servers"`
	DNSSECKeyMaster types.String `tfsdk:"dnssec_key_master"`
	IPADNSVersion   types.Int64  `tfsdk:"ipa_dns_version"`
}

func NewDNSGlobalConfigResource() resource.Resource {
	return &dnsGlobalConfig{}
}

func (r *dnsGlobalConfig) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_global_config"
}

func (r *dnsGlobalConfig) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "FreeIPA global DNS configuration. Singleton resource — only one instance per IPA realm. Create reads existing values then applies modifications; Delete reverts managed attributes to FreeIPA documented defaults (forwarders cleared, forward_policy=first, allow_sync_ptr=false, zone_refresh unset).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID of the resource. Always `global` for this singleton.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"forwarders": schema.ListAttribute{
				MarkdownDescription: "Global forwarders. A custom port can be specified for each forwarder using a standard format `IP_ADDRESS port PORT`.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"forward_policy": schema.StringAttribute{
				MarkdownDescription: "Global forwarding policy. One of `only`, `first`, `none`. Set to `none` to disable any configured global forwarders.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("only", "first", "none"),
				},
			},
			"allow_sync_ptr": schema.BoolAttribute{
				MarkdownDescription: "Allow synchronization of forward (A, AAAA) and reverse (PTR) records.",
				Optional:            true,
			},
			"zone_refresh": schema.Int64Attribute{
				MarkdownDescription: "Interval (in seconds) between regular polls of the name server for new DNS zones.",
				Optional:            true,
			},
			"dns_servers": schema.ListAttribute{
				MarkdownDescription: "List of IPA masters configured as DNS servers. Read-only.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"dnssec_key_master": schema.StringAttribute{
				MarkdownDescription: "IPA server configured as DNSSec key master. Read-only.",
				Computed:            true,
			},
			"ipa_dns_version": schema.Int64Attribute{
				MarkdownDescription: "IPA DNS version. Read-only.",
				Computed:            true,
			},
		},
	}
}

func (r *dnsGlobalConfig) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *dnsGlobalConfig) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
}

func (r *dnsGlobalConfig) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data dnsGlobalConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.readGlobalConfig(ctx, &data, &resp.Diagnostics); err != nil {
		if strings.Contains(err.Error(), "NotFound") {
			resp.Diagnostics.AddError("DNS not installed", "dnsconfig_show returned NotFound — IPA DNS subsystem is not installed on this realm")
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Error reading freeipa dns global config: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *dnsGlobalConfig) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}
func (r *dnsGlobalConfig) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}

// readGlobalConfig calls dnsconfig_show and copies remote attrs into the model.
// User-managed attrs are only overwritten if they are non-null in the model
// (so Read does not invent values for attrs the user has not declared).
func (r *dnsGlobalConfig) readGlobalConfig(ctx context.Context, data *dnsGlobalConfigModel, diags *diag.Diagnostics) (*ipa.Dnsconfig, error) {
	all := true
	res, err := r.client.DnsconfigShow(&ipa.DnsconfigShowArgs{}, &ipa.DnsconfigShowOptionalArgs{All: &all})
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Read freeipa dns global config: %s", res.String()))

	cfg := &res.Result

	if cfg.DNSServerServer != nil {
		v, d := types.ListValueFrom(ctx, types.StringType, *cfg.DNSServerServer)
		diags.Append(d...)
		data.DNSServers = v
	} else {
		data.DNSServers = types.ListNull(types.StringType)
	}
	if cfg.DnssecKeyMasterServer != nil {
		data.DNSSECKeyMaster = types.StringValue(*cfg.DnssecKeyMasterServer)
	} else {
		data.DNSSECKeyMaster = types.StringNull()
	}
	if cfg.Ipadnsversion != nil {
		data.IPADNSVersion = types.Int64Value(int64(*cfg.Ipadnsversion))
	} else {
		data.IPADNSVersion = types.Int64Null()
	}

	if !data.Forwarders.IsNull() {
		if cfg.Idnsforwarders != nil {
			v, d := types.ListValueFrom(ctx, types.StringType, *cfg.Idnsforwarders)
			diags.Append(d...)
			data.Forwarders = v
		} else {
			data.Forwarders = types.ListValueMust(types.StringType, []attr.Value{})
		}
	}
	if !data.ForwardPolicy.IsNull() {
		if cfg.Idnsforwardpolicy != nil {
			data.ForwardPolicy = types.StringValue(*cfg.Idnsforwardpolicy)
		} else {
			data.ForwardPolicy = types.StringNull()
		}
	}
	if !data.AllowSyncPTR.IsNull() {
		if cfg.Idnsallowsyncptr != nil {
			data.AllowSyncPTR = types.BoolValue(*cfg.Idnsallowsyncptr)
		} else {
			data.AllowSyncPTR = types.BoolNull()
		}
	}
	if !data.ZoneRefresh.IsNull() {
		if cfg.Idnszonerefresh != nil {
			data.ZoneRefresh = types.Int64Value(int64(*cfg.Idnszonerefresh))
		} else {
			data.ZoneRefresh = types.Int64Null()
		}
	}

	data.Id = types.StringValue("global")
	return cfg, nil
}

// suppress unused import warnings until Update/ImportState land in Task 3.
var (
	_ = strconv.Quote
	_ = path.Root
)
