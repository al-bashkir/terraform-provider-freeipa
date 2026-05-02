// Authors:
//   Pavel Aksenov <41126916+al-bashkir@users.noreply.github.com>
//
// SPDX-License-Identifier: GPL-3.0-only

package freeipa

import (
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func dnsServerName(t *testing.T) string {
	v := os.Getenv("FREEIPA_HOST")
	if v == "" {
		t.Skip("FREEIPA_HOST not set; cannot determine dns server name")
	}
	return v
}

func TestAccFreeIPADNSServer_basic(t *testing.T) {
	cfg := map[string]string{
		"index":          "0",
		"server_name":    `"` + dnsServerName(t) + `"`,
		"forwarders":     `["1.1.1.1", "8.8.8.8"]`,
		"forward_policy": `"first"`,
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFreeIPAProvider() + testAccFreeIPADNSServer_resource(cfg),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("freeipa_dns_server.dns-server-0", "id", dnsServerName(t)),
					resource.TestCheckResourceAttr("freeipa_dns_server.dns-server-0", "forward_policy", "first"),
					resource.TestCheckResourceAttr("freeipa_dns_server.dns-server-0", "forwarders.#", "2"),
				),
			},
			{
				Config: testAccFreeIPAProvider() + testAccFreeIPADNSServer_resource(cfg),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestAccFreeIPADNSServer_update(t *testing.T) {
	a := map[string]string{
		"index":          "0",
		"server_name":    `"` + dnsServerName(t) + `"`,
		"forwarders":     `["1.1.1.1"]`,
		"forward_policy": `"first"`,
	}
	b := map[string]string{
		"index":          "0",
		"server_name":    `"` + dnsServerName(t) + `"`,
		"forwarders":     `["9.9.9.9", "149.112.112.112"]`,
		"forward_policy": `"only"`,
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: testAccFreeIPAProvider() + testAccFreeIPADNSServer_resource(a)},
			{
				Config: testAccFreeIPAProvider() + testAccFreeIPADNSServer_resource(b),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("freeipa_dns_server.dns-server-0", "forward_policy", "only"),
					resource.TestCheckResourceAttr("freeipa_dns_server.dns-server-0", "forwarders.#", "2"),
				),
			},
			{
				Config: testAccFreeIPAProvider() + testAccFreeIPADNSServer_resource(b),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestAccFreeIPADNSServer_notFound(t *testing.T) {
	cfg := map[string]string{
		"index":          "0",
		"server_name":    `"nonexistent.replica.invalid"`,
		"forward_policy": `"first"`,
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccFreeIPAProvider() + testAccFreeIPADNSServer_resource(cfg),
				ExpectError: regexp.MustCompile(`DNS server not found`),
			},
		},
	})
}

func TestAccFreeIPADNSServer_import(t *testing.T) {
	cfg := map[string]string{
		"index":          "0",
		"server_name":    `"` + dnsServerName(t) + `"`,
		"forward_policy": `"first"`,
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: testAccFreeIPAProvider() + testAccFreeIPADNSServer_resource(cfg)},
			{
				ResourceName:            "freeipa_dns_server.dns-server-0",
				ImportState:             true,
				ImportStateId:           dnsServerName(t),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"forwarders", "soa_mname_override"},
			},
		},
	})
}
