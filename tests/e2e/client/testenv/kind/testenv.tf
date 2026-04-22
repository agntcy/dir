variable "image_tag" {
    type        = string
    description = "Directory image tags to use in testenv"
    default     = "latest"
}

module "testenv" {
    source = "../../../../modules/setup-testenv-k8s"
    
    kind_cluster_name = "dir-e2e-client"
    kind_config_path = "./kind-cluster.yaml"
    kind_extra_images = [
        "ghcr.io/agntcy/dir-reconciler:${var.image_tag}",
        "ghcr.io/agntcy/dir-apiserver:${var.image_tag}"
    ]

    chart_deployments = [
        {
            release_name = "dir"
            namespace = "dir"
            chart_path = "../../../../../install/charts/dir"
            chart_values_path = "./dir-chart-values.yaml"
            chart_extra_values = {
                "apiserver.image.tag": "${var.image_tag}",
                "apiserver.reconciler.image.tag": "${var.image_tag}"
            }
        }
    ]
}
