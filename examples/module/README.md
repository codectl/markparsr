# Virtual Network

This terraform module simplifies the process of creating and managing virtual network resources on azure with configurable options for network topology, subnets, security groups, and more to ensure a secure and efficient environment for resource communication in the cloud.

<!-- BEGIN_TF_DOCS -->
## Requirements

The following requirements are needed by this module:

- <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) (>= 1.9.0)

- <a name="requirement_azurerm"></a> [azurerm](#requirement\_azurerm) (~> 4.0)

## Providers

The following providers are used by this module:

- <a name="provider_azurerm"></a> [azurerm](#provider\_azurerm) (~> 4.0)

## Resources

The following resources are used by this module:

- [azurerm_network_security_group.this](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/network_security_group) (resource)
- [azurerm_network_security_rule.this](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/network_security_rule) (resource)
- [azurerm_route.this](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/route) (resource)
- [azurerm_route_table.this](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/route_table) (resource)
- [azurerm_subnet.this](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/subnet) (resource)
- [azurerm_subnet_network_security_group_association.this](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/subnet_network_security_group_association) (resource)
- [azurerm_subnet_route_table_association.this](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/subnet_route_table_association) (resource)
- [azurerm_virtual_network.this](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/virtual_network) (resource)
- [azurerm_virtual_network_dns_servers.this](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/virtual_network_dns_servers) (resource)

## Required Inputs

The following input variables are required:

### <a name="input_config"></a> [config](#input\_config)

Description: Contains virtual network configuration

Type:

```hcl
object({
    name                    = string
    resource_group_name     = optional(string)
    location                = optional(string)
    address_space           = list(string)
    tags                    = optional(map(string))
    edge_zone               = optional(string)
    bgp_community           = optional(string)
    flow_timeout_in_minutes = optional(number)
    dns_servers             = optional(list(string), [])
    encryption = optional(object({
      enforcement = optional(string, "AllowUnencrypted")
    }))
    subnets = optional(map(object({
      name                                          = optional(string)
      address_prefixes                              = list(string)
      service_endpoints                             = optional(list(string), [])
      private_link_service_network_policies_enabled = optional(bool, false)
      private_endpoint_network_policies             = optional(string, "Disabled")
      default_outbound_access_enabled               = optional(bool)
      service_endpoint_policy_ids                   = optional(list(string))
      delegations = optional(map(object({
        name    = string
        actions = optional(list(string), [])
      })))
      network_security_group = optional(object({
        name = optional(string)
        tags = optional(map(string))
        rules = optional(map(object({
          name                         = optional(string)
          priority                     = number
          direction                    = string
          access                       = string
          protocol                     = string
          description                  = optional(string, null)
          source_port_range            = optional(string, null)
          source_port_ranges           = optional(list(string), null)
          destination_port_range       = optional(string, null)
          destination_port_ranges      = optional(list(string), null)
          source_address_prefix        = optional(string, null)
          source_address_prefixes      = optional(list(string), null)
          destination_address_prefix   = optional(string, null)
          destination_address_prefixes = optional(list(string), null)
        })))
      }))
      route_table = optional(object({
        name                          = optional(string)
        bgp_route_propagation_enabled = optional(bool, true)
        tags                          = optional(map(string))
        routes = optional(map(object({
          name                   = optional(string)
          address_prefix         = string
          next_hop_type          = string
          next_hop_in_ip_address = optional(string, null)
        })))
      }))
      shared = optional(object({
        route_table            = optional(string)
        network_security_group = optional(string)
      }), {})
    })), {})
    network_security_groups = optional(map(object({
      name = optional(string)
      tags = optional(map(string))
      rules = optional(map(object({
        name                         = optional(string)
        priority                     = number
        direction                    = string
        access                       = string
        protocol                     = string
        description                  = optional(string, null)
        source_port_range            = optional(string, null)
        source_port_ranges           = optional(list(string), null)
        destination_port_range       = optional(string, null)
        destination_port_ranges      = optional(list(string), null)
        source_address_prefix        = optional(string, null)
        source_address_prefixes      = optional(list(string), null)
        destination_address_prefix   = optional(string, null)
        destination_address_prefixes = optional(list(string), null)
      })))
    })), {})
    route_tables = optional(map(object({
      name                          = optional(string)
      bgp_route_propagation_enabled = optional(bool, true)
      tags                          = optional(map(string))
      routes = optional(map(object({
        name                   = optional(string)
        address_prefix         = string
        next_hop_type          = string
        next_hop_in_ip_address = optional(string, null)
      })))
    })), {})
  })
```

## Optional Inputs

The following input variables are optional (have default values):

### <a name="input_location"></a> [location](#input\_location)

Description: default azure region to be used.

Type: `string`

Default: `null`

### <a name="input_naming"></a> [naming](#input\_naming)

Description: contains naming convention

Type: `map(string)`

Default: `{}`

### <a name="input_resource_group_name"></a> [resource\_group\_name](#input\_resource\_group\_name)

Description: default resource group to be used.

Type: `string`

Default: `null`

### <a name="input_tags"></a> [tags](#input\_tags)

Description: tags to be added to the resources

Type: `map(string)`

Default: `{}`

## Outputs

The following outputs are exported:

### <a name="output_config"></a> [config](#output\_config)

Description: contains virtual network configuration

### <a name="output_subnets"></a> [subnets](#output\_subnets)

Description: contains subnets configuration
<!-- END_TF_DOCS -->

## Goals

For more information, please see our [goals and non-goals](./GOALS.md).

## Testing

For more information, please see our testing [guidelines](./TESTING.md)

## Notes

This is an experimental module for private use.

A naming convention is developed using regular expressions to ensure correct abbreviations, with flexibility for multiple prefixes and suffixes.

Full usage examples and integrations with dependency modules are in the examples directory.

To update the module's documentation run `make doc`
