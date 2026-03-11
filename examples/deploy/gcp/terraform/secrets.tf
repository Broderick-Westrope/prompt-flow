resource "google_secret_manager_secret" "openai_api_key" {
  count     = var.openai_api_key != "" ? 1 : 0
  secret_id = "${var.service_name}-openai-api-key"

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "openai_api_key" {
  count       = var.openai_api_key != "" ? 1 : 0
  secret      = google_secret_manager_secret.openai_api_key[0].id
  secret_data = var.openai_api_key
}

resource "google_secret_manager_secret" "anthropic_api_key" {
  count     = var.anthropic_api_key != "" ? 1 : 0
  secret_id = "${var.service_name}-anthropic-api-key"

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "anthropic_api_key" {
  count       = var.anthropic_api_key != "" ? 1 : 0
  secret      = google_secret_manager_secret.anthropic_api_key[0].id
  secret_data = var.anthropic_api_key
}
