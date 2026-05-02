// Authors:
//   Pavel Aksenov <aksenov.pavel.v@gmail.com>
//
// SPDX-License-Identifier: GPL-3.0-only

package freeipa

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	ipa "github.com/infra-monkey/go-freeipa/freeipa"
)

var _ resource.Resource = &dnsForwardZone{}

type dnsForwardZone struct {
	client *ipa.Client
}

func NewDNSForwardZoneResource() resource.Resource {
	return &dnsForwardZone{}
}

func (r *dnsForwardZone) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_forward_zone"
}

func (r *dnsForwardZone) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "FreeIPA DNS forward zone resource",
		Attributes:          map[string]schema.Attribute{},
	}
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

func (r *dnsForwardZone) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
}
func (r *dnsForwardZone) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}
func (r *dnsForwardZone) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}
func (r *dnsForwardZone) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}
