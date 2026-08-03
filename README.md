# slo-guardian

A Kubernetes operator that enforces Service Level Objectives (SLOs) on workloads.

Given an `SLOPolicy` custom resource pointing at a target Deployment/URL, the operator:

- Continuously health-checks the target over HTTP
- Tracks a rolling (sliding-window) availability percentage
- Computes remaining error budget against a configured SLO target
- Auto-remediates (restarts or scales the target) when the error budget is exhausted, with a configurable cooldown
- Exposes Prometheus metrics for availability, error budget, checks, and remediations
- Records Kubernetes Events and updates CR status for every action taken

## Why

This project exists to demonstrate practical Kubernetes operator development and SRE
concepts (SLOs, error budgets, auto-remediation) end to end — not just the application code,
but the surrounding tooling: tests, CI, containerization, and infra-as-code.

## Tech Stack

- Go
- Kubernetes
- controller-runtime / Kubebuilder
- Docker
- Prometheus
- GitHub Actions
- Terraform (GCP)

## Example

```yaml
apiVersion: sre.sre.dev/v1alpha1
kind: SLOPolicy
metadata:
  name: sample-slopolicy
spec:
  targetDeployment: nginx-demo
  targetURL: http://nginx-demo.default.svc.cluster.local
  sloTargetPercent: 99.9
  checkIntervalSeconds: 15
  remediationAction: RestartDeployment
  remediationCooldownSeconds: 30
```

```bash
kubectl get slopolicy
NAME               AVAILABILITY   BUDGET REMAINING   TARGET
sample-slopolicy   99.5           50                 99.9
```

## Running locally

Requires: Go 1.22+, Docker, [kind](https://kind.sigs.k8s.io/), kubectl, [kubebuilder](https://book.kubebuilder.io/) (for regenerating manifests only).

```bash
# 1. Create a local cluster
kind create cluster --name slo-guardian

# 2. Install the CRD
make install

# 3. Build and load the controller image
docker build -t slo-guardian:v0.1.0 .
kind load docker-image slo-guardian:v0.1.0 --name slo-guardian

# 4. Deploy the controller
make deploy IMG=slo-guardian:v0.1.0

# 5. Deploy something to monitor
kubectl create deployment nginx-demo --image=nginx:alpine --port=80
kubectl expose deployment nginx-demo --port=80 --target-port=80

# 6. Apply the sample SLOPolicy
kubectl apply -f config/samples/sre_v1alpha1_slopolicy.yaml

# 7. Watch it work
kubectl get slopolicy -w
```

To see remediation trigger, force an outage:

```bash
kubectl scale deployment nginx-demo --replicas=0
# wait for error budget to go negative, then:
kubectl scale deployment nginx-demo --replicas=1
kubectl describe slopolicy sample-slopolicy   # see the Remediated event
```

## Testing

```bash
make test       # unit + envtest integration tests
make test-e2e    # full e2e suite against a throwaway kind cluster
make lint
```

## CI/CD

- `test.yml` — unit/integration tests + lint on every push and PR
- `test-e2e.yml` — full e2e suite (spins up its own kind cluster) on every push and PR
- `publish.yml` — builds and publishes the controller image to
  [GitHub Container Registry](https://github.com/Shihasz/slo-guardian/pkgs/container/slo-guardian)
  on every push to `main`

## Infrastructure as Code

`terraform/` provisions a GKE Standard cluster and Artifact Registry repo for running this in GCP.
See [terraform/README.md](terraform/README.md) for details — this is written to be applied for a real
deployment, but is not currently applied to a live GCP project.

## Project layout

```text
.
├── api/
│   └── v1alpha1/          # CRD type definitions (SLOPolicy)
├── config/                # Kustomize manifests (CRDs, RBAC, manager, Prometheus)
├── internal/
│   └── controller/        # Reconcile loop, health checks, remediation, metrics, tracker
├── terraform/             # GKE cluster and Artifact Registry provisioning
├── test/
│   └── e2e/               # End-to-end test suite
├── .github/
│   └── workflows/         # CI/CD pipelines
├── Dockerfile
├── Makefile
├── go.mod
├── go.sum
└── README.md
```

## Status

Feature-complete for its current scope. Possible extensions: multi-endpoint targets,
Slack/PagerDuty notifications on breach, a Grafana dashboard, webhook-based validation
of SLOPolicy specs.

## License

MIT — see [LICENSE](LICENSE).
