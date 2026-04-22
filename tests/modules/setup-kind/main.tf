terraform {
  required_version = ">= 1.6.0"

  required_providers {
    null = {
      source  = "hashicorp/null"
      version = "~> 3.2"
    }
  }
}

variable "kind_cluster_name" {
  type        = string
  description = "KIND cluster name"
  default     = "test-cluster"
}

variable "kind_config_path" {
  type        = string
  description = "Path to KIND cluster config"
  default     = ""
}

variable "kind_extra_images" {
  type        = list(string)
  description = "Additional container images to load into the KIND cluster"
  default     = []
}

variable "kind_bin" {
  type        = string
  description = "Path to the KIND binary"
  default     = "kind"
}

locals {
  kind_config_abs = abspath(var.kind_config_path)
  config_arg = var.kind_config_path != "" ? "--config ${local.kind_config_abs}" : ""
}

resource "null_resource" "kind_cluster" {
  triggers = {
    cluster_name = var.kind_cluster_name
    kind_bin = var.kind_bin
  }

  provisioner "local-exec" {
    command = <<-EOT
      ${self.triggers.kind_bin} create cluster --name ${self.triggers.cluster_name} ${local.config_arg}
      for image in ${join(" ", var.kind_extra_images)}; do
        ${self.triggers.kind_bin} load docker-image $image --name ${self.triggers.cluster_name}
      done
    EOT
  }

  provisioner "local-exec" {
    when    = destroy
    command = "${self.triggers.kind_bin} delete cluster --name ${self.triggers.cluster_name}"
  }
}
