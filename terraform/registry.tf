resource "google_artifact_registry_repository" "godzilla" {
  location      = var.region
  repository_id = "godzilla"
  format        = "DOCKER"

  depends_on = [google_project_service.artifactregistry]
}
