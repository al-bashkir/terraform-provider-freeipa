resource "freeipa_dns_forward_zone" "internal" {
  zone_name          = "internal.example.lan"
  forwarders         = ["10.0.0.10", "10.0.0.11 port 5353"]
  forward_policy     = "only"
  disable_zone       = false
  skip_overlap_check = true
}
