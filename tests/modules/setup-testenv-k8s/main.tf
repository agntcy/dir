terraform {
  required_version = ">= 1.6.0"

  required_providers {
    null = {
      source  = "hashicorp/null"
      version = "~> 3.2"
    }

    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.13"
    }
  }
}

// Create kind cluster for testing
variable "kind_cluster_name" {
  type        = string
  description = "Name of the Kind cluster to set up"
  default     = "e2e-test"
}

variable "kind_config_path" {
  type        = string
  description = "Path to the Kind cluster configuration file"
}

variable "kind_extra_images" {
  type        = list(string)
  description = "List of additional container images to load into the Kind cluster"
  default     = []
}

variable "kind_bin" {
  type        = string
  description = "Path to the Kind binary"
  default     = "kind"
}

module "setup_kind" {
  source = "../setup-kind"
  count = var.kind_cluster_name != "" ? 1 : 0

  kind_cluster_name = var.kind_cluster_name
  kind_config_path  = var.kind_config_path
  kind_extra_images = var.kind_extra_images
  kind_bin          = var.kind_bin
}

// Deploy a set of Helm charts to the Kind cluster
variable "kubeconfig_context" {
    type        = string
    description = "Kubernetes context to use for Helm deployments."
    default     = ""
}

provider "helm" {
  kubernetes {
    config_path = "~/.kube/config"
    config_context = var.kind_cluster_name != "" ? "kind-${var.kind_cluster_name}" : var.kubeconfig_context
  }
}

variable "chart_deployments" {
  type = list(object({
    chart_path = string
    release_name = string
    namespace = string
    chart_repository = optional(string)
    chart_values_path = optional(string)
    chart_extra_values = optional(map(string))
  }))
  description = "List of Helm charts to deploy to the Kind cluster"
  default = []
}

module "deploy_charts" {
  source = "../deploy-chart"
  depends_on = [ module.setup_kind ]

  for_each = { for idx, chart in var.chart_deployments : idx => chart }

  chart_path = each.value.chart_path
  release_name = each.value.release_name
  namespace = each.value.namespace
  chart_repository = each.value.chart_repository != null ? each.value.chart_repository : ""
  chart_values_path = each.value.chart_values_path != null ? each.value.chart_values_path : ""
  chart_extra_values = each.value.chart_extra_values != null ? each.value.chart_extra_values : {}
  wait = false
}

resource "null_resource" "wait_for_charts" {
  depends_on = [ module.deploy_charts ]

  for_each = { for idx, chart in var.chart_deployments : idx => chart }

  provisioner "local-exec" {
    # Wait for 10 mins
    command = "sleep 10 && kubectl wait --for=condition=ready --timeout=600s pods -n ${each.value.namespace} -l app.kubernetes.io/instance=${each.value.release_name}"
  }
}
