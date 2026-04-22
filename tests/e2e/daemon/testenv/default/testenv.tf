variable "daemon_config_path" {
    type        = string
    description = "Path to the daemon configuration file"
    default     = "./dir-daemon-config.yaml"
}

variable "daemon_data_dir_path" {
    type        = string
    description = "Path to the daemon data directory"
    default     = "/tmp/testenv-e2e-daemon-default"
}

variable "dirctl_path" {
    type        = string
    description = "Path to the dirctl binary"
    default     = "../../../../../.bin/dirctl"
}

module "testenv" {
    source = "../../../../modules/setup-testenv-local"
    
    local_deployment = {
        setup_cmd = <<-EOT
            ${var.dirctl_path} daemon start \
                --config ${var.daemon_config_path} \
                --data-dir ${var.daemon_data_dir_path} \
                    > logs.daemon.log 2>&1 &
        EOT
        destroy_cmd = <<-EOT
            ${var.dirctl_path} daemon stop \
                --config ${var.daemon_config_path} \
                --data-dir ${var.daemon_data_dir_path}

            cat logs.daemon.log
        EOT
    }
}
