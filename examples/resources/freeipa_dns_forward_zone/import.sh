# The import id is the undotted zone FQDN.

import {
  to = freeipa_dns_forward_zone.internal
  id = "internal.example.lan"
}

resource "freeipa_dns_forward_zone" "internal" {
  zone_name  = "internal.example.lan"
  forwarders = ["10.0.0.10"]
}
