terraform {
  required_version = ">= 1.6.0"

  required_providers {
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.13"
    }
  }
}

variable "chart_path" {
  type        = string
  description = "Path to the Helm chart. Supports local paths or remote charts (e.g., 'stable/nginx')"
}

variable "chart_repository" {
  type        = string
  description = "Optional Helm repository URL if the chart is not local. Required for remote charts."
  default     = ""
}

variable "chart_values_path" {
  type        = string
  description = "Path to Helm charts values file"
  default     = ""
}

variable "chart_extra_values" {
  type        = map(string)
  description = "Additional Helm values to merge with the base values.yaml (in YAML format)"
  default     = {}
}

variable "release_name" {
  type        = string
  description = "Helm release name"
}

variable "namespace" {
  type        = string
  description = "Kubernetes namespace for the Helm release"
  default     = "default"
}

variable "wait" {
  type        = bool
  description = "Whether to wait for the Helm release to be deployed successfully"
  default     = true
}

resource "helm_release" "chart" {
  name             = var.release_name
  namespace        = var.namespace
  chart            = var.chart_path
  repository       = var.chart_repository != "" ? var.chart_repository : null
  create_namespace = true

  values = [
    var.chart_values_path != "" ? file(var.chart_values_path) : "",
    yamlencode(var.chart_extra_values),
  ]

  wait = var.wait
  wait_for_jobs = var.wait

  # Wait for 10 mins
  timeout = 600
}
