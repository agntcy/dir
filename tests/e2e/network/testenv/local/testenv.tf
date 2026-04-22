terraform {
  required_version = ">= 1.6.0"
}

variable "daemon_data_dir_path" {
    type        = string
    description = "Path to the daemon data directory"
    default     = "/tmp"
}

variable "daemon_extra_env" {
    type        = string
    description = "Extra environment variables to set for the daemon process"
    default     = ""
}

variable "dirctl_path" {
    type        = string
    description = "Path to the dirctl binary"
    default     = "../../../../../.bin/dirctl"
}

variable "bootstrap_key_path" {
    type        = string
    description = "Path to the bootstrap key file"
    default     = "./bootstrap.key"
}

variable "bootstrap_daemon_config_path" {
    type        = string
    description = "Path to the bootstrap daemon config file"
    default     = "./daemon-bootstrap-config.yaml"
}

variable "nodes_daemon_config_paths" {
    type        = list(string)
    description = "Path to the nodes daemon config files"
    default     = [
        "./daemon-peer1-config.yaml",
        "./daemon-peer2-config.yaml",
        "./daemon-peer3-config.yaml"
    ]
}

locals {
    bootstrap_data_dir = abspath("${var.daemon_data_dir_path}/bootstrap")
    logs_dir = abspath("./.logs")
}

module "bootstrap" {
    source = "../../../../modules/setup-testenv-local"

    local_deployment = {
        setup_cmd = <<-EOT
            # Copy bootstrap key to the data dir
            mkdir -p ${local.bootstrap_data_dir}
            cp ${var.bootstrap_key_path} ${local.bootstrap_data_dir}/bootstrap.key

            # Create log dir
            mkdir -p ${local.logs_dir}

            # Start daemon
            ${var.daemon_extra_env} \
            ${var.dirctl_path} daemon start \
                --config ${var.bootstrap_daemon_config_path} \
                --data-dir ${local.bootstrap_data_dir} \
                    > ${local.logs_dir}/bootstrap.daemon.log 2>&1 &

            # Give some time for daemon to start since peers need it
            sleep 10
        EOT
        destroy_cmd = <<-EOT
            ${var.daemon_extra_env} \
            ${var.dirctl_path} daemon stop \
                --config ${var.bootstrap_daemon_config_path} \
                --data-dir ${local.bootstrap_data_dir}

            cat ${local.logs_dir}/bootstrap.daemon.log
        EOT
    }
}

module "testenv" {
    source = "../../../../modules/setup-testenv-local"
    
    depends_on = [ module.bootstrap ]
    for_each = toset(var.nodes_daemon_config_paths)

    local_deployment = {
        setup_cmd = <<-EOT
            ${var.daemon_extra_env} \
            ${var.dirctl_path} daemon start \
                --config ${each.value} \
                --data-dir ${var.daemon_data_dir_path}/${basename(each.value)} \
                   > ${local.logs_dir}/${basename(each.value)}.log 2>&1 &
        EOT
        destroy_cmd = <<-EOT
            ${var.daemon_extra_env} \
            ${var.dirctl_path} daemon stop \
                --config ${each.value} \
                --data-dir ${var.daemon_data_dir_path}/${basename(each.value)}

            cat ${local.logs_dir}/${basename(each.value)}.log
        EOT
    }
}
