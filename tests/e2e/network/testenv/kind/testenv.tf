variable "image_tag" {
    type        = string
    description = "Directory image tags to use in testenv"
    default     = "latest"
}

module "testenv" {
  source = "../../../../modules/setup-testenv-k8s"

  kind_bin          = "kind"
  kind_cluster_name = "dir-e2e-network"
  kind_config_path  = "./kind-cluster.yaml"
  kind_extra_images = [
    "ghcr.io/agntcy/dir-reconciler:${var.image_tag}",
    "ghcr.io/agntcy/dir-apiserver:${var.image_tag}"
  ]

  chart_deployments = [
    {
      release_name      = "dir"
      namespace         = "bootstrap"
      chart_path        = "../../../../../install/charts/dir"
      chart_values_path = "./dir-chart-values-bootstrap.yaml"
      chart_extra_values = {
        "apiserver.image.tag" : "${var.image_tag}",
        "apiserver.reconciler.image.tag" : "${var.image_tag}"
      }
    },
    {
      release_name      = "dir"
      namespace         = "peer1"
      chart_path        = "../../../../../install/charts/dir"
      chart_values_path = "./dir-chart-values-peer1.yaml"
      chart_extra_values = {
        "apiserver.image.tag" : "${var.image_tag}",
        "apiserver.reconciler.image.tag" : "${var.image_tag}"
      }
    },
    {
      release_name      = "dir"
      namespace         = "peer2"
      chart_path        = "../../../../../install/charts/dir"
      chart_values_path = "./dir-chart-values-peer2.yaml"
      chart_extra_values = {
        "apiserver.image.tag" : "${var.image_tag}",
        "apiserver.reconciler.image.tag" : "${var.image_tag}"
      }
    },
    {
      release_name      = "dir"
      namespace         = "peer3"
      chart_path        = "../../../../../install/charts/dir"
      chart_values_path = "./dir-chart-values-peer3.yaml"
      chart_extra_values = {
        "apiserver.image.tag" : "${var.image_tag}",
        "apiserver.reconciler.image.tag" : "${var.image_tag}"
      }
    }
  ]
}
