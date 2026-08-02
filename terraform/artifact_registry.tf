resource "google_artifact_registry_repository" "slo_guardian" {
  location      = var.region
  repository_id = "slo-guardian"
  description   = "Container images for the SLO Guardian operator"
  format        = "DOCKER"
}
