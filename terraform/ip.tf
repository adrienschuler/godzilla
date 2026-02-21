resource "google_compute_global_address" "godzilla_ip" {
  name = "${var.cluster_name}-ip"

  depends_on = [google_project_service.compute]
}
