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

func TestAccFreeIPADNSGlobalConfig_basic(t *testing.T) {
	cfg := map[string]string{
		"forwarders":     `["1.1.1.1", "8.8.8.8 port 5353"]`,
		"forward_policy": `"first"`,
		"allow_sync_ptr": "true",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFreeIPAProvider() + testAccFreeIPADNSGlobalConfig_resource(cfg),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("freeipa_dns_global_config.global", "id", "global"),
					resource.TestCheckResourceAttr("freeipa_dns_global_config.global", "forward_policy", "first"),
					resource.TestCheckResourceAttr("freeipa_dns_global_config.global", "allow_sync_ptr", "true"),
					resource.TestCheckResourceAttr("freeipa_dns_global_config.global", "forwarders.#", "2"),
				),
			},
			{
				Config: testAccFreeIPAProvider() + testAccFreeIPADNSGlobalConfig_resource(cfg),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestAccFreeIPADNSGlobalConfig_update(t *testing.T) {
	a := map[string]string{
		"forwarders":     `["1.1.1.1"]`,
		"forward_policy": `"first"`,
		"allow_sync_ptr": "false",
	}
	b := map[string]string{
		"forwarders":     `["9.9.9.9", "149.112.112.112"]`,
		"forward_policy": `"only"`,
		"allow_sync_ptr": "true",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: testAccFreeIPAProvider() + testAccFreeIPADNSGlobalConfig_resource(a)},
			{
				Config: testAccFreeIPAProvider() + testAccFreeIPADNSGlobalConfig_resource(b),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("freeipa_dns_global_config.global", "forward_policy", "only"),
					resource.TestCheckResourceAttr("freeipa_dns_global_config.global", "allow_sync_ptr", "true"),
					resource.TestCheckResourceAttr("freeipa_dns_global_config.global", "forwarders.#", "2"),
				),
			},
			{
				Config: testAccFreeIPAProvider() + testAccFreeIPADNSGlobalConfig_resource(b),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestAccFreeIPADNSGlobalConfig_clearForwarders(t *testing.T) {
	withFwd := map[string]string{
		"forwarders":     `["1.1.1.1"]`,
		"forward_policy": `"first"`,
	}
	noFwd := map[string]string{
		"forwarders":     `[]`,
		"forward_policy": `"first"`,
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: testAccFreeIPAProvider() + testAccFreeIPADNSGlobalConfig_resource(withFwd)},
			{
				Config: testAccFreeIPAProvider() + testAccFreeIPADNSGlobalConfig_resource(noFwd),
				Check:  resource.TestCheckResourceAttr("freeipa_dns_global_config.global", "forwarders.#", "0"),
			},
		},
	})
}

func TestAccFreeIPADNSGlobalConfig_import(t *testing.T) {
	cfg := map[string]string{
		"forward_policy": `"first"`,
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: testAccFreeIPAProvider() + testAccFreeIPADNSGlobalConfig_resource(cfg)},
			{
				ResourceName:            "freeipa_dns_global_config.global",
				ImportState:             true,
				ImportStateId:           "global",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"forwarders", "forward_policy", "allow_sync_ptr"},
			},
		},
	})
}
