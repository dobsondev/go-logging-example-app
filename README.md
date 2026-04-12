# Go Logging Example Application

This is an example backend Go API that allows me to test out logging via Loki and Grafana on my homelab. It is deployed via GitOps using ArgoCD.

## Manufacturing Errors

This application has the following endpoint that will generate error logs:

```
/errors/{count}
```

You can ping this endpoint to generate a number of error log entries equal to `{count}`. For example, you could use the following to generate 6 errors:

```bash
curl localhost:8080/errors/6
```

The whole idea behind this is just to run this application and then use the errors endpoint to mess around with Grafana and Loki on my homelab.

## Generating a Spread of Errors

A script is provided to automatically generate a spread of errors over time, which is useful for creating a more realistic error pattern on the Grafana dashboard:

```bash
./scripts/generate-errors-on-homelab.sh
```

This hits the `/errors/{count}` endpoint with varying counts (`5, 10, 3, 8, 15, 2, 12`) with a 30 second sleep between each request, spreading errors over roughly 3.5 minutes. The variation in counts produces a jagged line on the Grafana dashboard that is much more useful for testing than a flat uniform spike.

## Fabricate a Deployment

We can annotate the app on ArgoCD in order to force a refresh:

```bash
kubectl annotate application go-logging-example-app -n argocd argocd.argoproj.io/refresh=hard --overwrite
```

This triggers the same notification pipeline as a real deployment, which is useful for testing the Grafana annotation webhook without having to push a new release.

## Homelab Setup

This repository is integrated with the [dobsondev/homelab](https://github.com/dobsondev/homelab) repository to enable a fully automated GitOps deployment pipeline via ArgoCD. When a release tag (e.g. `v0.0.1`) is pushed, the release workflow builds and pushes the Docker image to the GitHub Container Registry (GHCR), then automatically updates the Helm values file in the homelab repository with the new image tag. ArgoCD detects the change in the homelab repository and syncs the updated manifest to the cluster, deploying the new version of the application.

For day-to-day development, the CI workflow runs on every pull request targeting `main`, building the application, building the Docker image, and running integration tests via Newman against a live container to ensure the API is functioning correctly before any code is merged. A manual deployment workflow is also available via `workflow_dispatch`, allowing a specific image tag to be deployed at any time — it verifies the image exists in GHCR before updating the homelab repository, preventing a bad tag from being deployed.

See:
- [release.yml](./.github/workflows/release.yml)
- [manual-gitops.yml](./.github/workflows/manual-gitops.yml)
- https://github.com/dobsondev/homelab/blob/main/argocd/apps/go-logging-example-app.yml
- https://github.com/dobsondev/homelab/blob/main/argocd/helm/values/go-logging-example-app.yml

## Grafana Dashboard

A dedicated Grafana dashboard — **go-logging-example-app Monitoring** — is set up to visualize error logs and correlate them with deployments. It consists of two panels:

### Error Rate Panel (Time Series)

Displays the number of error logs over time using the following LogQL query against Loki:

```logql
sum(
  count_over_time(
    {app="go-logging-example-app-pod"} | json | level="ERROR"
    [$__interval]
  )
) or vector(0)
```

The `or vector(0)` ensures the line drops to zero during periods with no errors rather than connecting across gaps. The panel is styled as a red step-line graph so spikes are immediately visible.

### Error Logs Panel (Logs)

Displays the raw error log lines below the time series graph using:

```logql
{app="go-logging-example-app-pod"} | json | level="ERROR"
```

Clicking on any spike in the time series panel above automatically narrows the logs panel to that time window, so you can read the exact error messages without leaving the dashboard.

## Deployment Annotations

Every time ArgoCD successfully syncs this application, it automatically posts a deployment annotation to Grafana via the ArgoCD Notifications webhook. This appears as a vertical red line on the Error Rate panel, making it easy to correlate error spikes with specific deployments.

The annotation text includes the exact Docker image that was deployed:

```
Deployed go-logging-example-app → ghcr.io/dobsondev/go-logging-example-app:v0.0.2
```

### Setting This Up for Another Service

To replicate this setup for a different service:

**1. Add the notification subscription annotation to the ArgoCD Application manifest:**

```yaml
metadata:
  annotations:
    notifications.argoproj.io/subscribe.on-sync-succeeded.grafana: ""
```

**2. Create the Grafana dashboard panels** with the LogQL queries above, substituting your app's Loki label for `go-logging-example-app-pod`. You can find the correct label value by browsing to **Explore → Loki → Label browser** in Grafana and looking for the `app` label on your service's pods.

**3. Add the deployment annotation query to the dashboard** under **Settings → Annotations → Add annotation query**:

| Field | Value |
|---|---|
| Data source | `-- Grafana --` |
| Filter by | Tags |
| Tags | `deployment`, `<your-app-name>` |
| Match any | Off |
| Color | Red |

The ArgoCD Notifications ConfigMap and Grafana webhook service are already configured in the homelab repository and apply to any ArgoCD Application that has the subscription annotation — no further infrastructure changes are needed.

See:
- https://github.com/dobsondev/homelab/blob/main/argocd/argocd-notifications.yml