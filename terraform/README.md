# Terraform (GCP)

Provisions everything needed to run SLO Guardian on Google Kubernetes Engine:

- A regional **GKE Standard** cluster with a separately managed, autoscaling node pool
- A **GCP Artifact Registry** repository to host the controller image

## Status

This configuration is validated (`terraform validate` passes) but has not been applied against a real
GCP project as part of this project — no cluster is currently running in the cloud. It's written to be
usable as-is by anyone who wants to actually deploy this project.

## Using this for a real deployment

1. Have a GCP project with billing enabled, and the following APIs turned on:
   `container.googleapis.com`, `artifactregistry.googleapis.com`

2. Authenticate locally:

```bash
   gcloud auth application-default login
```

3. Configure your variables:

```bash
   cd terraform
   cp terraform.tfvars.example terraform.tfvars
   # edit terraform.tfvars and set your real project_id
```

4. Provision:

```bash
   terraform init
   terraform validate
   terraform plan
   terraform apply
```

5. Point `kubectl` at the new cluster:

```bash
   gcloud container clusters get-credentials $(terraform output -raw cluster_name) --region <your-region>
```

6. Push the controller image to the Artifact Registry repo this creates, then deploy with:

```bash
   make deploy IMG=$(terraform output -raw artifact_registry_url)/slo-guardian:v0.1.0
```

7. Apply the CRDs and a sample `SLOPolicy` as described in the main [README](../README.md).

## Tearing down

```bash
terraform destroy
```

## Notes on cost

A GKE Standard cluster with the default settings in this config (`e2-standard-2` nodes, autoscaling
1-3) is **not** free-tier eligible and will incur real GCP charges while running. Review `variables.tf`
and adjust `node_machine_type`/`node_count` before applying against a real project if cost matters to you.
