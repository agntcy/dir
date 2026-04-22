terraform {
  required_version = ">= 1.6.0"

  required_providers {
    null = {
      source  = "hashicorp/null"
      version = "~> 3.2"
    }
  }
}

// Run a CLI command for a local environment
variable "local_deployment" {
  type = object({
    setup_cmd = string
    destroy_cmd = string
  })
  description = "Local deployment commands to run for setup and teardown"
}

resource "null_resource" "manage_deployment" {
  triggers = {
    setup_cmd = var.local_deployment.setup_cmd
    destroy_cmd = var.local_deployment.destroy_cmd
  }

  provisioner "local-exec" {
    command = self.triggers.setup_cmd
  }

  provisioner "local-exec" {
    when = destroy
    command = self.triggers.destroy_cmd
  }
}
