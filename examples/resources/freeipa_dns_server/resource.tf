resource "freeipa_dns_server" "ipa1" {
  server_name        = "ipa1.example.lan"
  forwarders         = ["1.1.1.1", "8.8.8.8"]
  forward_policy     = "first"
  soa_mname_override = "ns1.example.lan."
}
