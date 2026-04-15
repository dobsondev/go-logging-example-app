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

The same type of script exists for local as well:

```bash
./scripts/generate-errors-on-local.sh
```

The local version just uses `localhost:8080` and has less time inbetween making the errors because it is more for checking in Jaegar for traces.

## OpenTelemetry Traces

This application is instrumented with [OpenTelemetry](https://opentelemetry.io/) for distributed tracing. OpenTelemetry is a vendor-neutral observability framework that provides a standard way to collect and export telemetry data — traces, metrics, and logs — from your applications. A **trace** represents the entire journey of a single request through your system and is identified by a unique `traceID`. Within a trace, each unit of work is represented as a **span**, which has its own `spanID`, a start and end time, and optionally a set of attributes, events, and a status. Spans are linked together in a parent/child hierarchy — for example, an incoming HTTP request span might have child spans for a database query and an external API call — forming the waterfall view you see in Jaeger or Grafana Tempo. When requests cross service boundaries, the `traceID` is propagated via HTTP headers (the `traceparent` header) so the entire distributed trace can be reconstructed end-to-end across multiple services.

For further reading:

- [Observability Primer](https://opentelemetry.io/docs/concepts/observability-primer/) — explains the "why" behind traces, spans, and metrics before any code
- [OpenTelemetry Traces Concepts](https://opentelemetry.io/docs/concepts/signals/traces/) — deep dive on traces and spans, covering parent/child relationships, attributes, events, and status
- [OpenTelemetry Go — Getting Started](https://opentelemetry.io/docs/languages/go/getting-started/) — hands-on introduction to OpenTelemetry in Go
- [OpenTelemetry Go — Instrumentation](https://opentelemetry.io/docs/languages/go/instrumentation/) — covers manual instrumentation patterns in depth, including custom spans, attributes, and error recording

### Local Tracing with Jaeger

For local development, [Jaeger](https://www.jaegertracing.io/) can be used to collect and visualize OpenTelemetry traces. This lets you inspect traces locally without needing to connect to the homelab Tempo instance.

A `docker-compose.yml` is provided that runs both the application and a Jaeger instance together:

```bash
docker compose up
```

Once running, you can generate some traces by hitting the application endpoints:

```bash
curl localhost:8080/health
curl localhost:8080/errors/5
```

Then open the Jaeger UI at **http://localhost:16686**, select `go-logging-example-app` from the service dropdown, and click **Find Traces**. You should see a trace for each request you made, showing the full span breakdown including HTTP method, path, and response status.

### What You're Seeing

Each trace represents a single HTTP request flowing through the application. Within a trace you'll see one or more **spans** — each span represents a unit of work. With the current setup you'll see a single span per request since the app is not yet manually instrumented with child spans. As you add custom spans to handlers and business logic, the trace waterfall view will show the full breakdown of where time is being spent within each request.

### Homelab

When deployed to the homelab cluster, traces are sent to Tempo instead of Jaeger. The `OTEL_EXPORTER_OTLP_ENDPOINT` environment variable controls where traces are sent — locally it points to Jaeger, and in the cluster it points to the Tempo service:

```
tempo.monitoring.svc.cluster.local:4318
```

Traces can be explored in Grafana under **Explore → Tempo**.

### OpenTelemetry Context

Context is how OpenTelemetry knows which trace and span a new span belongs to. When a request comes in, the `otelhttp` middleware creates a root span and stores it in the request's `context.Context`. As that context is passed down through your call stack, any new spans you create are automatically linked to the parent span stored in the context — forming the parent/child hierarchy you see in the trace waterfall view.

This is why every function that creates a span needs to both accept a `context.Context` as its first argument and pass it down to any functions it calls:

```go
// The context carries the parent span from the HTTP middleware
func LogErrors(w http.ResponseWriter, r *http.Request) {
  ctx := r.Context() // contains the root span created by otelhttp

  extraWork(ctx) // pass it down so extraWork can link to the parent
}

func extraWork(ctx context.Context) {
  tracer := otel.Tracer("go-logging-example-app")
  ctx, span := tracer.Start(ctx, "extraWork") // creates a child span linked to the parent
  defer span.End()

  // ...
}
```

If you call `tracer.Start()` without passing the context — or pass a fresh `context.Background()` instead — the new span has no parent and will appear as an entirely separate unconnected trace in Jaeger or Tempo, which is almost never what you want.

The simplest rule to remember is: **always pass `ctx` through, never create a new context from scratch inside a function that is part of a request lifecycle**. This is also just good Go practice regardless of tracing — the standard Go convention is that `context.Context` is always the first argument of any function that does I/O or meaningful work.

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

### Fabricate a Deployment

We can annotate the app on ArgoCD in order to force a refresh:

```bash
kubectl annotate application go-logging-example-app -n argocd argocd.argoproj.io/refresh=hard --overwrite
```

This triggers the same notification pipeline as a real deployment, which is useful for testing the Grafana annotation webhook without having to push a new release.

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