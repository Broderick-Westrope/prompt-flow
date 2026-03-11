# --- Logging ---

resource "azurerm_log_analytics_workspace" "main" {
  name                = "${var.service_name}-logs"
  resource_group_name = local.resource_group.name
  location            = local.resource_group.location
  sku                 = "PerGB2018"
  retention_in_days   = 30
}

# --- Container App Environment ---

resource "azurerm_container_app_environment" "main" {
  name                       = "${var.service_name}-env"
  resource_group_name        = local.resource_group.name
  location                   = local.resource_group.location
  log_analytics_workspace_id = azurerm_log_analytics_workspace.main.id
}

# --- Secrets configuration ---
#
# Container Apps uses two-layer indirection for Key Vault integration:
#   1. Container App "secret" blocks reference Key Vault secrets via managed identity
#   2. Container "env" blocks reference the Container App secrets by name
#
# We build these dynamically so secrets are only created when the corresponding
# API key variable is non-empty (matching the GCP dynamic env pattern).

locals {
  # Each entry maps a Container App secret name to its Key Vault secret and
  # the environment variable it should be exposed as.
  secret_mappings = {
    openai-api-key = {
      key_vault_secret_id = var.openai_api_key != "" ? azurerm_key_vault_secret.openai_api_key[0].versionless_id : ""
      env_name            = "OPENAI_API_KEY"
      enabled             = var.openai_api_key != ""
    }
    anthropic-api-key = {
      key_vault_secret_id = var.anthropic_api_key != "" ? azurerm_key_vault_secret.anthropic_api_key[0].versionless_id : ""
      env_name            = "ANTHROPIC_API_KEY"
      enabled             = var.anthropic_api_key != ""
    }
    azure-openai-api-key = {
      key_vault_secret_id = var.azure_openai_api_key != "" ? azurerm_key_vault_secret.azure_openai_api_key[0].versionless_id : ""
      env_name            = "AZURE_OPENAI_API_KEY"
      enabled             = var.azure_openai_api_key != ""
    }
  }

  # Filter to only enabled secrets.
  enabled_secrets = { for k, v in local.secret_mappings : k => v if v.enabled }
}

# --- Container App ---

resource "azurerm_container_app" "main" {
  name                         = var.service_name
  resource_group_name          = local.resource_group.name
  container_app_environment_id = azurerm_container_app_environment.main.id
  revision_mode                = "Single"

  identity {
    type         = "UserAssigned"
    identity_ids = [azurerm_user_assigned_identity.container_app.id]
  }

  registry {
    server   = azurerm_container_registry.main.login_server
    identity = azurerm_user_assigned_identity.container_app.id
  }

  # Key Vault secret references — Container App secret blocks.
  dynamic "secret" {
    for_each = local.enabled_secrets
    content {
      name                = secret.key
      key_vault_secret_id = secret.value.key_vault_secret_id
      identity            = azurerm_user_assigned_identity.container_app.id
    }
  }

  ingress {
    external_enabled = var.allow_external_ingress
    target_port      = 8080

    traffic_weight {
      latest_revision = true
      percentage      = 100
    }
  }

  template {
    min_replicas = var.min_replicas
    max_replicas = var.max_replicas

    container {
      name   = var.service_name
      image  = "${azurerm_container_registry.main.login_server}/${var.service_name}:${var.image_tag}"
      cpu    = var.cpu
      memory = var.memory

      args = ["serve", "-p", "8080", "--timeout", var.execution_timeout, var.flow_path]

      # Environment variables mapped from Container App secrets.
      dynamic "env" {
        for_each = local.enabled_secrets
        content {
          name        = env.value.env_name
          secret_name = env.key
        }
      }

      # Azure OpenAI endpoint — plain env var (not sensitive).
      dynamic "env" {
        for_each = var.azure_openai_endpoint != "" ? [1] : []
        content {
          name  = "AZURE_OPENAI_ENDPOINT"
          value = var.azure_openai_endpoint
        }
      }

      liveness_probe {
        transport = "HTTP"
        path      = "/healthz"
        port      = 8080

        interval_seconds = 30
      }

      startup_probe {
        transport = "HTTP"
        path      = "/readyz"
        port      = 8080

        initial_delay = 5
      }
    }

    http_scale_rule {
      name                = "concurrent-requests"
      concurrent_requests = tostring(var.concurrency)
    }
  }
}
