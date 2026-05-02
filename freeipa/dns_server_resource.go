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

var _ resource.Resource = &dnsServer{}

type dnsServer struct {
	client *ipa.Client
}

func NewDNSServerResource() resource.Resource {
	return &dnsServer{}
}

func (r *dnsServer) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_server"
}

func (r *dnsServer) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "FreeIPA per-server DNS configuration resource (manage-in-place)",
		Attributes:          map[string]schema.Attribute{},
	}
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

func (r *dnsServer) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
}
func (r *dnsServer) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}
func (r *dnsServer) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}
func (r *dnsServer) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}
