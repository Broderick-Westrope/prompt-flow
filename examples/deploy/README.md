# Deployment Examples

These are **reference implementations** — copy them into your own repository and adapt to your needs. They are not meant to be used directly from this repo.

## Deployment Model

Each flow runs as its own serverless container service:

- **One image per flow** — your Dockerfile bundles `pfctl` with your flow definition(s)
- **One service per flow** — each flow gets its own auto-scalable serverless instance
- **Scale to zero** — no traffic means no cost (supported on both GCP and Azure)

## Building a Flow Image

Use `Dockerfile.example` as a starting point. The key pattern is:

1. Use the pfctl base image
2. Copy your flow YAML file into the image
3. Set CMD to `pfctl serve` with your flow file

```dockerfile
FROM ghcr.io/broderick/prompt-flow:latest
COPY my-flow.flow.yaml /flow.yaml
CMD ["serve", "-p", "8080", "/flow.yaml"]
```

Pin to a specific version for reproducible builds (e.g., `ghcr.io/broderick/prompt-flow:1.0.0`).

See `Dockerfile.example` for the full template.

## Cloud Examples

### GCP (Cloud Run)

The `gcp/` directory contains:

- **`terraform/`** — Terraform module for Cloud Run, Artifact Registry, Secret Manager, and IAM
- **`deploy-gcp.yml`** — GitHub Actions workflow for building, pushing, and deploying to Cloud Run

Prerequisites: GCP project, `gcloud` CLI, Terraform, Workload Identity Federation for CI/CD.

### Azure (Coming Soon)

Azure Container Apps support is planned. The `azure/` directory will contain Terraform modules and a GitHub Actions workflow following the same pattern as the GCP example.
