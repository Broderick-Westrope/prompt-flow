resource "azurerm_container_registry" "main" {
  name                = replace(var.service_name, "-", "")
  resource_group_name = local.resource_group.name
  location            = local.resource_group.location
  sku                 = "Basic"
  admin_enabled       = false # managed identity handles authentication
}
