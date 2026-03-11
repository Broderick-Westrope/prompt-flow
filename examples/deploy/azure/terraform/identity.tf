resource "azurerm_user_assigned_identity" "container_app" {
  name                = "${var.service_name}-identity"
  resource_group_name = local.resource_group.name
  location            = local.resource_group.location
}

# Allow the Container App to pull images from the container registry.
resource "azurerm_role_assignment" "acr_pull" {
  scope                = azurerm_container_registry.main.id
  role_definition_name = "AcrPull"
  principal_id         = azurerm_user_assigned_identity.container_app.principal_id
}

# Key Vault access is handled via access policy in key_vault.tf.
