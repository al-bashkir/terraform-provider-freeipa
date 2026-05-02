# freeipa_dns_global_config is a singleton — its import id is always the literal
# string "global".

import {
  to = freeipa_dns_global_config.global
  id = "global"
}

resource "freeipa_dns_global_config" "global" {
  forward_policy = "first"
}
