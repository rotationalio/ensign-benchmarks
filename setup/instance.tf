provider "google" {
    credentials = "${file(var.credentials_path)}"
    project     = "${var.project}"
    region      = "us-central1"
}

resource "google_compute_instance" "default" {
    name         = "benchmarks-${formatdate("YYYYMMDDhhmmss", timestamp())}"
    machine_type = "n1-highcpu-16"
    zone         = "us-central1-c"

    boot_disk {
      initialize_params {
        size  = 128
        type  = "pd-ssd"
        image = "ubuntu-2204-jammy-v20230727"
      }
    }

    network_interface {
      network = "default"
      access_config {
      }
    }

    metadata = {
      ssh-keys = "benchmarks:${file(var.pub_key_path)}"
    }

    metadata_startup_script = "${file("install.sh")}"
}

output "public_ip" {
    value = "${google_compute_instance.default.network_interface.0.access_config.0.nat_ip}"
}

variable "credentials_path" {}
variable "project" {}
variable "pub_key_path" {}
