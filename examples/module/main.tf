# virtual network
resource "azurerm_virtual_network" "this" {
  location = try(
    var.config.location, var.location
  )

  resource_group_name = try(
    var.config.resource_group_name, var.resource_group_name
  )

  name                    = var.config.name
  address_space           = var.config.address_space
  edge_zone               = var.config.edge_zone
  bgp_community           = var.config.bgp_community
  flow_timeout_in_minutes = var.config.flow_timeout_in_minutes

  dynamic "encryption" {
    for_each = var.config.encryption != null ? [var.config.encryption] : []
    content {
      enforcement = encryption.value.enforcement
    }
  }

  tags = try(
    var.config.tags, var.tags, {}
  )

  lifecycle {
    ignore_changes = [subnet, dns_servers]
  }
}

resource "azurerm_virtual_network_dns_servers" "this" {
  for_each = length(lookup(var.config, "dns_servers", [])) > 0 ? { "default" = var.config.dns_servers } : {}

  virtual_network_id = azurerm_virtual_network.this.id
  dns_servers        = each.value
}

# subnets
resource "azurerm_subnet" "this" {
  for_each = lookup(
    var.config, "subnets", {}
  )

  name = coalesce(
    lookup(each.value, "name", null),
    join("-", [var.naming.subnet, each.key])
  )

  resource_group_name = try(
    var.config.resource_group_name, var.resource_group_name
  )

  virtual_network_name                          = azurerm_virtual_network.this.name
  address_prefixes                              = each.value.address_prefixes
  service_endpoints                             = each.value.service_endpoints
  private_link_service_network_policies_enabled = each.value.private_link_service_network_policies_enabled
  private_endpoint_network_policies             = each.value.private_endpoint_network_policies
  service_endpoint_policy_ids                   = each.value.service_endpoint_policy_ids
  default_outbound_access_enabled               = each.value.default_outbound_access_enabled

  dynamic "delegation" {
    for_each = each.value.delegations != null ? each.value.delegations : {}

    content {
      name = delegation.key

      service_delegation {
        name    = delegation.value.name
        actions = delegation.value.actions
      }
    }
  }
}

# network security groups individual and shared
resource "azurerm_network_security_group" "this" {
  for_each = merge(
    var.config.network_security_groups, {
      for subnet_key, subnet in var.config.subnets : subnet_key => subnet.network_security_group
      if subnet.network_security_group != null
    }
  )

  name = coalesce(
    lookup(each.value, "name", null),
    "${var.naming.network_security_group}-${each.key}"
  )

  resource_group_name = try(
    var.config.resource_group_name, var.resource_group_name
  )
  location = try(
    var.config.location, var.location
  )

  tags = try(
    var.config.tags, var.tags, {}
  )

  lifecycle {
    ignore_changes = [security_rule]
  }
}

# security rules individual and shared
resource "azurerm_network_security_rule" "this" {
  for_each = merge({
    for pair in flatten([
      for nsg_key, nsg in lookup(var.config, "network_security_groups", {}) :
      can(nsg.rules) ? (
        nsg.rules != null ? [
          for rule_key, rule in nsg.rules : {
            key = "${nsg_key}_${rule_key}"
            value = {
              nsg_name = azurerm_network_security_group.this[nsg_key].name
              rule     = rule
              rule_name = coalesce(lookup(rule, "name", null),
                join("-", [var.naming.network_security_group_rule, rule_key])
              )
            }
          }
        ] : []
      ) : []
    ]) : pair.key => pair.value
    }, {
    for pair in flatten([
      for subnet_key, subnet in var.config.subnets :
      can(subnet.network_security_group.rules) ? (
        subnet.network_security_group.rules != null ? [
          for rule_key, rule in subnet.network_security_group.rules : {
            key = "${subnet_key}_${rule_key}"
            value = {
              nsg_name = azurerm_network_security_group.this[subnet_key].name
              rule     = rule
              rule_name = coalesce(lookup(rule, "name", null),
                join("-", [var.naming.network_security_group_rule, rule_key])
              )
            }
          }
        ] : []
      ) : []
    ]) : pair.key => pair.value
    }
  )

  name                         = each.value.rule_name
  priority                     = each.value.rule.priority
  direction                    = each.value.rule.direction
  access                       = each.value.rule.access
  protocol                     = each.value.rule.protocol
  source_port_range            = each.value.rule.source_port_range
  source_port_ranges           = each.value.rule.source_port_ranges
  destination_port_range       = each.value.rule.destination_port_range
  destination_port_ranges      = each.value.rule.destination_port_ranges
  source_address_prefix        = each.value.rule.source_address_prefix
  source_address_prefixes      = each.value.rule.source_address_prefixes
  destination_address_prefix   = each.value.rule.destination_address_prefix
  destination_address_prefixes = each.value.rule.destination_address_prefixes
  description                  = each.value.rule.description
  resource_group_name          = var.config.resource_group_name
  network_security_group_name  = each.value.nsg_name
}

# nsg associations
resource "azurerm_subnet_network_security_group_association" "this" {
  for_each = {
    for subnet_key, subnet in var.config.subnets : subnet_key => subnet
    if subnet.shared.network_security_group != null || subnet.network_security_group != null
  }

  subnet_id                 = azurerm_subnet.this[each.key].id
  network_security_group_id = each.value.shared.network_security_group != null ? azurerm_network_security_group.this[each.value.shared.network_security_group].id : azurerm_network_security_group.this[each.key].id

  depends_on = [azurerm_network_security_rule.this]
}

# route tables individual and shared
resource "azurerm_route_table" "this" {
  for_each = merge(
    var.config.route_tables, {
      for subnet_key, subnet in var.config.subnets : subnet_key => subnet.route_table
      if subnet.route_table != null
    }
  )

  name = coalesce(
    lookup(each.value, "name", null),
    "${var.naming.route_table}-${each.key}"
  )

  resource_group_name = try(
    var.config.resource_group_name, var.resource_group_name
  )
  location = try(
    var.config.location, var.location
  )

  bgp_route_propagation_enabled = each.value.bgp_route_propagation_enabled

  tags = try(
    var.config.tags, var.tags, {}
  )

  lifecycle {
    ignore_changes = [route]
  }
}

# routes individual and shared
resource "azurerm_route" "this" {
  for_each = merge({
    for pair in flatten([
      for rt_key, rt in lookup(var.config, "route_tables", {}) :
      can(rt.routes) ? (
        rt.routes != null ? [
          for route_key, route in rt.routes : {
            key = "${rt_key}_${route_key}"
            value = {
              route_table_name = azurerm_route_table.this[rt_key].name
              route            = route
              route_name = coalesce(
                lookup(route, "name", null),
                join("-", [var.naming.route, route_key])
              )
            }
          }
        ] : []
      ) : []
    ]) : pair.key => pair.value
    }, {
    for pair in flatten([
      for subnet_key, subnet in var.config.subnets :
      can(subnet.route_table.routes) ? (
        subnet.route_table.routes != null ? [
          for route_key, route in subnet.route_table.routes : {
            key = "${subnet_key}_${route_key}"
            value = {
              route_table_name = azurerm_route_table.this[subnet_key].name
              route            = route
              route_name = coalesce(
                lookup(route, "name", null),
                join("-", [var.naming.route, route_key])
              )
            }
          }
        ] : []
      ) : []
    ]) : pair.key => pair.value
  })

  name                   = each.value.route_name
  resource_group_name    = var.config.resource_group_name
  route_table_name       = each.value.route_table_name
  address_prefix         = each.value.route.address_prefix
  next_hop_type          = each.value.route.next_hop_type
  next_hop_in_ip_address = each.value.route.next_hop_in_ip_address
}

# route table associations
resource "azurerm_subnet_route_table_association" "this" {
  for_each = {
    for subnet_key, subnet in var.config.subnets : subnet_key => subnet
    if subnet.shared.route_table != null || subnet.route_table != null
  }

  subnet_id      = azurerm_subnet.this[each.key].id
  route_table_id = each.value.shared.route_table != null ? azurerm_route_table.this[each.value.shared.route_table].id : azurerm_route_table.this[each.key].id
}
