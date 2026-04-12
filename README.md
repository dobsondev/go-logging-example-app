# Go Logging Example Application

This is an example backend Go API that allows me to test out logging via Loki and Grafana on my homelab. It is deployed via GitOps using ArgoCD.

## Manufacturing Errors

This application has the following endpoint that will generate errors logs:

```
/errors/{count}
```

You can ping this endpoint to generate a number of error log entries equal to `{count}`. For example, you could use the following to generate 6 errors:

```bash
curl localhost:8080/errors/6
```

The whole idea behind this is just to run this application and then use the errors endpoint to mess around with Granfana and Loki on my homelab.


## Homelab Setup

This repository is integrated with the [dobsondev/homelab](https://github.com/dobsondev/homelab) repository to enable a fully automated GitOps deployment pipeline via ArgoCD. When a release tag (e.g. `v0.0.1`) is pushed, the release workflow builds and pushes the Docker image to the GitHub Container Registry (GHCR), then automatically updates the Helm values file in the homelab repository with the new image tag. ArgoCD detects the change in the homelab repository and syncs the updated manifest to the cluster, deploying the new version of the application.

For day-to-day development, the CI workflow runs on every pull request targeting `main`, building the application, building the Docker image, and running integration tests via Newman against a live container to ensure the API is functioning correctly before any code is merged. A manual deployment workflow is also available via `workflow_dispatch`, allowing a specific image tag to be deployed at any time — it verifies the image exists in GHCR before updating the homelab repository, preventing a bad tag from being deployed.

See:
- [release.yml](./.github/workflows/release.yml)
- [manual-gitops.yml](./.github/workflows/manual-gitops.yml)
- https://github.com/dobsondev/homelab/blob/main/argocd/apps/go-logging-example-app.yml
- https://github.com/dobsondev/homelab/blob/main/argocd/helm/values/go-logging-example-app.yml