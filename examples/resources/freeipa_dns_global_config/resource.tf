resource "freeipa_dns_global_config" "global" {
  forwarders     = ["1.1.1.1", "8.8.8.8 port 5353"]
  forward_policy = "first"
  allow_sync_ptr = true
}
