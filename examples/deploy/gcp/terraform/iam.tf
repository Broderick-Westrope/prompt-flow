resource "google_service_account" "cloud_run" {
  account_id   = "${var.service_name}-run"
  display_name = "Cloud Run service account for ${var.service_name}"
}

resource "google_project_iam_member" "vertex_ai_user" {
  project = var.project_id
  role    = "roles/aiplatform.user"
  member  = "serviceAccount:${google_service_account.cloud_run.email}"
}

resource "google_secret_manager_secret_iam_member" "openai_accessor" {
  count     = var.openai_api_key != "" ? 1 : 0
  secret_id = google_secret_manager_secret.openai_api_key[0].secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.cloud_run.email}"
}

resource "google_secret_manager_secret_iam_member" "anthropic_accessor" {
  count     = var.anthropic_api_key != "" ? 1 : 0
  secret_id = google_secret_manager_secret.anthropic_api_key[0].secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.cloud_run.email}"
}
