# For full specification on the configuration of this file visit:
# https://github.com/hashicorp/integration-template#metadata-configuration
integration {
  name        = "Numspot"
  description = "Use Packer to create Numspot Images."
  identifier  = "packer/numspot/numspot"

  component {
    type = "builder"
    name = "Numspot BSU"
    slug = "bsu"
  }
}
