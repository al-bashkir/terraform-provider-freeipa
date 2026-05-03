// Authors:
//   Pavel Aksenov <41126916+al-bashkir@users.noreply.github.com>
//
// SPDX-License-Identifier: GPL-3.0-only

package freeipa

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccFreeIPADNSForwardZone_basic(t *testing.T) {
	cfg := map[string]string{
		"index":              "0",
		"zone_name":          `"fwd.example.lan"`,
		"forwarders":         `["1.1.1.1", "8.8.8.8 port 5353"]`,
		"forward_policy":     `"first"`,
		"skip_overlap_check": "true",
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFreeIPAProvider() + testAccFreeIPADNSForwardZone_resource(cfg),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("freeipa_dns_forward_zone.fwd-zone-0", "computed_zone_name", "fwd.example.lan."),
					resource.TestCheckResourceAttr("freeipa_dns_forward_zone.fwd-zone-0", "forward_policy", "first"),
					resource.TestCheckResourceAttr("freeipa_dns_forward_zone.fwd-zone-0", "forwarders.#", "2"),
					resource.TestCheckResourceAttr("freeipa_dns_forward_zone.fwd-zone-0", "disable_zone", "false"),
				),
			},
			{
				Config: testAccFreeIPAProvider() + testAccFreeIPADNSForwardZone_resource(cfg),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestAccFreeIPADNSForwardZone_update(t *testing.T) {
	a := map[string]string{
		"index":              "0",
		"zone_name":          `"fwdupd.example.lan"`,
		"forwarders":         `["1.1.1.1"]`,
		"forward_policy":     `"first"`,
		"skip_overlap_check": "true",
	}
	b := map[string]string{
		"index":              "0",
		"zone_name":          `"fwdupd.example.lan"`,
		"forwarders":         `["9.9.9.9", "149.112.112.112"]`,
		"forward_policy":     `"only"`,
		"skip_overlap_check": "true",
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: testAccFreeIPAProvider() + testAccFreeIPADNSForwardZone_resource(a)},
			{
				Config: testAccFreeIPAProvider() + testAccFreeIPADNSForwardZone_resource(b),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("freeipa_dns_forward_zone.fwd-zone-0", "forward_policy", "only"),
					resource.TestCheckResourceAttr("freeipa_dns_forward_zone.fwd-zone-0", "forwarders.#", "2"),
				),
			},
			{
				Config: testAccFreeIPAProvider() + testAccFreeIPADNSForwardZone_resource(b),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestAccFreeIPADNSForwardZone_disable(t *testing.T) {
	enabled := map[string]string{
		"index":              "0",
		"zone_name":          `"fwddis.example.lan"`,
		"forwarders":         `["1.1.1.1"]`,
		"disable_zone":       "false",
		"skip_overlap_check": "true",
	}
	disabled := map[string]string{
		"index":              "0",
		"zone_name":          `"fwddis.example.lan"`,
		"forwarders":         `["1.1.1.1"]`,
		"disable_zone":       "true",
		"skip_overlap_check": "true",
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: testAccFreeIPAProvider() + testAccFreeIPADNSForwardZone_resource(enabled)},
			{
				Config: testAccFreeIPAProvider() + testAccFreeIPADNSForwardZone_resource(disabled),
				Check:  resource.TestCheckResourceAttr("freeipa_dns_forward_zone.fwd-zone-0", "disable_zone", "true"),
			},
			{
				Config: testAccFreeIPAProvider() + testAccFreeIPADNSForwardZone_resource(enabled),
				Check:  resource.TestCheckResourceAttr("freeipa_dns_forward_zone.fwd-zone-0", "disable_zone", "false"),
			},
		},
	})
}

func TestAccFreeIPADNSForwardZone_skipOverlap(t *testing.T) {
	cfg := map[string]string{
		"index":              "0",
		"zone_name":          `"fwdoverlap.example.lan"`,
		"forwarders":         `["1.1.1.1"]`,
		"skip_overlap_check": "true",
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFreeIPAProvider() + testAccFreeIPADNSForwardZone_resource(cfg),
				Check:  resource.TestCheckResourceAttr("freeipa_dns_forward_zone.fwd-zone-0", "skip_overlap_check", "true"),
			},
		},
	})
}

func TestAccFreeIPADNSForwardZone_caseInsensitive(t *testing.T) {
	a := map[string]string{
		"index":              "0",
		"zone_name":          `"fwdcase.example.lan"`,
		"forwarders":         `["1.1.1.1"]`,
		"skip_overlap_check": "true",
	}
	b := map[string]string{
		"index":              "0",
		"zone_name":          `"FwdCase.Example.LAN"`,
		"forwarders":         `["1.1.1.1"]`,
		"skip_overlap_check": "true",
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: testAccFreeIPAProvider() + testAccFreeIPADNSForwardZone_resource(a)},
			{
				Config: testAccFreeIPAProvider() + testAccFreeIPADNSForwardZone_resource(b),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestAccFreeIPADNSForwardZone_import(t *testing.T) {
	cfg := map[string]string{
		"index":              "0",
		"zone_name":          `"fwdimport.example.lan"`,
		"forwarders":         `["1.1.1.1"]`,
		"forward_policy":     `"first"`,
		"skip_overlap_check": "true",
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: testAccFreeIPAProvider() + testAccFreeIPADNSForwardZone_resource(cfg)},
			{
				ResourceName:            "freeipa_dns_forward_zone.fwd-zone-0",
				ImportState:             true,
				ImportStateId:           "fwdimport.example.lan",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"zone_name", "skip_overlap_check", "forwarders", "forward_policy"},
			},
		},
	})
}
