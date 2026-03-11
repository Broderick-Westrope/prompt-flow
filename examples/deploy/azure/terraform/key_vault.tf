data "azurerm_client_config" "current" {}

resource "azurerm_key_vault" "main" {
  name                = var.service_name
  resource_group_name = local.resource_group.name
  location            = local.resource_group.location
  tenant_id           = data.azurerm_client_config.current.tenant_id
  sku_name            = "standard"

  # Enable for production. Purge protection prevents Key Vault name reuse for
  # the soft-delete retention period after destroy, which creates friction when
  # iterating on reference infrastructure.
  purge_protection_enabled   = false
  soft_delete_retention_days = 7

  # Grant the Terraform caller full secret management permissions.
  access_policy {
    tenant_id = data.azurerm_client_config.current.tenant_id
    object_id = data.azurerm_client_config.current.object_id

    secret_permissions = [
      "Get",
      "List",
      "Set",
      "Delete",
      "Purge",
      "Recover",
    ]
  }

  # Grant the Container App managed identity read-only secret access.
  access_policy {
    tenant_id = data.azurerm_client_config.current.tenant_id
    object_id = azurerm_user_assigned_identity.container_app.principal_id

    secret_permissions = [
      "Get",
    ]
  }

  # Production hardening: restrict network access via VNet rules and private endpoints.
}

# --- API key secrets (conditional on non-empty values) ---

resource "azurerm_key_vault_secret" "openai_api_key" {
  count        = var.openai_api_key != "" ? 1 : 0
  name         = "${var.service_name}-openai-api-key"
  value        = var.openai_api_key
  key_vault_id = azurerm_key_vault.main.id
}

resource "azurerm_key_vault_secret" "anthropic_api_key" {
  count        = var.anthropic_api_key != "" ? 1 : 0
  name         = "${var.service_name}-anthropic-api-key"
  value        = var.anthropic_api_key
  key_vault_id = azurerm_key_vault.main.id
}

resource "azurerm_key_vault_secret" "azure_openai_api_key" {
  count        = var.azure_openai_api_key != "" ? 1 : 0
  name         = "${var.service_name}-azure-openai-api-key"
  value        = var.azure_openai_api_key
  key_vault_id = azurerm_key_vault.main.id
}
