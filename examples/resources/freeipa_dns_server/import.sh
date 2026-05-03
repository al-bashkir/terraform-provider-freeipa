# The import id is the FQDN of the IPA replica (matches `idnsserverid` in
# FreeIPA). The dnsserver row must already exist (created by ipa-dns-install).

import {
  to = freeipa_dns_server.ipa1
  id = "ipa1.example.lan"
}

resource "freeipa_dns_server" "ipa1" {
  server_name    = "ipa1.example.lan"
  forward_policy = "first"
}
