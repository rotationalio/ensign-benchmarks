# Ensign Benchmarks

**Benchmarking utilities for the Ensign eventing platform**

This package implements benchmarking utilities for the Ensign eventing platform allowing us to performance test live systems and ensure they are working correctly.


## Create Instance

The benchmarks are designed to run on a Google Cloud compute instance with 16 cores; ensuring that the same compute resources are used for all benchmarks. This project includes a [terraform](https://www.terraform.io/) resource file to setup the instance and tear it down. Before following these steps, please ensure you've installed the `terraform` CLI on your local machine. All other commands will be run on the instance that has been created by this step.

The following steps will allow you to create the new Ubuntu 22.04 instance, install dependencies, docker and docker compose, and clone this repository. Once the instance is running, the benchmarks can be executed.

Step 1: set some environment variables

```
# path to gcloud credentials file
export TF_VAR_credentials_path=

# project name
export TF_VAR_project=

# public keys to SSH into the instance with
export TF_VAR_pub_key_path=
```

See notes below about creating gcloud credentials and SSH keys.

Step 2: create the VM:

```
$ cd setup
$ terraform apply
```

This command will output the public IP address of the server. You can then SSH into the node using the `benchmarks` user in order to run the benchmarks.

After running all the benchmarks, destroy the instance because you will be charged real money by Google!

```
$ terraform destroy
```

### Note 1: Google Cloud

In order to run the benchmarks on a GCP instance, you need a [Google Cloud Platform account](https://cloud.google.com/). You'll have to setup the account and the user as well as a [project](https://cloud.google.com/resource-manager/docs/creating-managing-projects). You'll also likely need to install the [gcloud cli](https://cloud.google.com/sdk/gcloud) to ensure you can log into Google Cloud locally.

The terraform resource is set up to use a Google Service Account JSON configuration file as described [here](https://registry.terraform.io/providers/hashicorp/google/latest/docs/guides/provider_reference.html#running-terraform-outside-of-google-cloud). Create the service account and download the credentials, setting the `$TF_VAR_credentials_path` environment variable to the credentials file and the `$TF_VAR_project` environment variable to the project name.

NOTE: the service account will need the Compute Engine Admin permission.

### Note 2: SSH Keys and SSH

You will need SSH keys in order to login to the instance that is created by terraform.

To create SSH key pairs, use `ssh-keygen`.

This will ask you a few questions and generate two files, named by default `id_rsa` and `id_rsa.pub`. These files are usually stored in your `~/.ssh` directory, but you can place them wherever you'd like. The `$TF_VAR_pub_key_path` environment variable needs to point at the file with the `.pub` extension, e.g. `id_rsa.pub`.

The instance is created with a `benchmarks` user. To SSH into the instance:

```
$ ssh -i [path to id_rsa] benchmarks@[ipaddr]
```