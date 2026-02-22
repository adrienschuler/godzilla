output "cluster_name" {
  value = google_container_cluster.primary.name
}

output "cluster_endpoint" {
  value     = google_container_cluster.primary.endpoint
  sensitive = true
}

output "registry_url" {
  value = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.godzilla.repository_id}"
}

output "static_ip" {
  value = google_compute_global_address.godzilla_ip.address
}
