output "container_app_url" {
  description = "Container App FQDN"
  value       = azurerm_container_app.main.ingress[0].fqdn
}

output "container_registry_url" {
  description = "Container Registry login server URL"
  value       = azurerm_container_registry.main.login_server
}
