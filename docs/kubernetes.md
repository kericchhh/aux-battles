# Kubernetes

The app runs on a local [k3d](https://k3d.io) cluster. Argo CD deploys it from
`main`, together with the monitoring stack, so after the one-time setup below a
merged pull request is a deploy.

| Path | Contents |
| --- | --- |
| [`deploy/k8s/k3d.yaml`](../deploy/k8s/k3d.yaml) | Cluster config: one node, ingress on `localhost:8088` |
| [`deploy/k8s/manifests/`](../deploy/k8s/manifests) | Namespace and the CloudNativePG `Cluster`, applied by hand |
| [`deploy/k8s/chart/aux-battles/`](../deploy/k8s/chart/aux-battles) | Helm chart for the API, worker, web client and migrations |
| [`deploy/k8s/argocd/root.yaml`](../deploy/k8s/argocd/root.yaml) | Root Application that deploys everything in `apps/` |
| [`deploy/k8s/argocd/apps/`](../deploy/k8s/argocd/apps) | Applications for the app, kube-prometheus-stack and the Grafana dashboard |

PostgreSQL stays outside the chart because the migrate hook runs before any
chart resource exists, so the database must already be there.

## Set up

You need Docker, `kubectl`, Helm and k3d. The commands use bash.

1. Create the cluster:

   ```sh
   k3d cluster create --config deploy/k8s/k3d.yaml --wait
   ```

2. Install the CloudNativePG operator, then create the database:

   ```sh
   helm repo add cnpg https://cloudnative-pg.github.io/charts
   helm install cnpg cnpg/cloudnative-pg --version 0.29.1 -n cnpg-system --create-namespace --wait
   kubectl apply -f deploy/k8s/manifests/namespace.yaml -f deploy/k8s/manifests/postgres.yaml
   kubectl wait cluster/db -n aux-battles --for=condition=Ready --timeout=5m
   ```

   The operator stores the app's connection string in the `db-app` Secret.

3. Install Argo CD:

   ```sh
   helm repo add argo https://argoproj.github.io/argo-helm
   helm install argocd argo/argo-cd --version 10.11.7 -n argocd --create-namespace --wait
   ```

4. Create the Secrets that are not in Git. Grafana's admin password is
   random; the Telegram values are the same as `TELEGRAM_BOT_TOKEN` and
   `TELEGRAM_CHAT_ID` in `.env` (see [Telegram setup](observability.md#telegram-notifications)):

   ```sh
   kubectl create namespace monitoring
   kubectl create secret generic grafana-admin -n monitoring \
     --from-literal=admin-user=admin --from-literal=admin-password="$(openssl rand -base64 24)"
   read -rsp 'Bot token: ' TG_TOKEN; echo; read -rsp 'Chat ID: ' TG_CHAT; echo
   kubectl create secret generic alertmanager-telegram -n monitoring \
     --from-file=bot-token=<(printf %s "$TG_TOKEN") --from-file=chat-id=<(printf %s "$TG_CHAT")
   unset TG_TOKEN TG_CHAT
   ```

5. Hand over to Argo CD. This is the last manual apply:

   ```sh
   kubectl apply -f deploy/k8s/argocd/root.yaml
   kubectl get applications -n argocd
   ```

   After a few minutes `root`, `aux-battles`, `monitoring` and `dashboards` are
   all `Synced` and `Healthy`, and the game is at <http://localhost:8088>.

To upload songs, an account needs the admin role:

```sh
kubectl exec -n aux-battles db-1 -c postgres -- psql -d aux -c "UPDATE users SET role = 'ADMIN' WHERE lower(username) = lower('USERNAME')"
```

## Open the dashboards

| Tool | Command | Address | Login |
| --- | --- | --- | --- |
| Argo CD | `kubectl port-forward -n argocd svc/argocd-server 8443:443` | <https://localhost:8443> | `admin`, password in Secret `argocd-initial-admin-secret` |
| Grafana | `kubectl port-forward -n monitoring svc/monitoring-grafana 3000:80` | <http://localhost:3000> | `admin`, password in Secret `grafana-admin` |
| Prometheus | `kubectl port-forward -n monitoring svc/monitoring-kube-prometheus-prometheus 9090:9090` | <http://localhost:9090> | none |
| Alertmanager | `kubectl port-forward -n monitoring svc/monitoring-kube-prometheus-alertmanager 9093:9093` | <http://localhost:9093> | none |

Read a password without it landing in your shell history:

```sh
kubectl get secret grafana-admin -n monitoring -o jsonpath='{.data.admin-password}' | base64 -d; echo
```

## Deploy

Merge to `main`. Argo CD checks Git every three minutes; **Refresh** in its UI
checks at once.

To release a new app version, first tag it as in the
[deploy runbook](runbooks/deploy.md#release) and wait for CI to publish the
images. Then change `appVersion` and bump `version` in
[`Chart.yaml`](../deploy/k8s/chart/aux-battles/Chart.yaml). The same rule
applies as there: migrations only add, so going back to an older
`appVersion` still works.

Do not use `helm upgrade` or `kubectl edit` on the app: self-heal reverts any
change that is not in Git. Only `root.yaml`, the `manifests/` files and the
operator and Argo CD installs are applied by hand.

## Alerts

Prometheus evaluates the stack's community rules and two app rules from the
chart, and Alertmanager sends firing and resolved alerts to Telegram:

| Alert | Fires when |
| --- | --- |
| `WorkerDown` | No worker has been scrapeable for 2 minutes, including when none exists |
| `SongProcessingFailing` | More than 3 processing attempts failed in 30 minutes; one bad upload makes 3 |

The always-firing `Watchdog` alert is routed to a `null` receiver.

## Maintenance

To change the cluster by hand, for example to test `WorkerDown` by scaling the
worker to 0, pause sync on `root` first. Otherwise it restores the
`aux-battles` Application, whose self-heal undoes the change:

```sh
kubectl patch app root -n argocd --type merge -p '{"spec":{"syncPolicy":{"automated":null}}}'
kubectl patch app aux-battles -n argocd --type merge -p '{"spec":{"syncPolicy":{"automated":{"selfHeal":false}}}}'
kubectl scale deploy/worker -n aux-battles --replicas=0
```

Resume with `kubectl apply -f deploy/k8s/argocd/root.yaml`. Root restores the
Applications from Git and self-heal brings the worker back.

## Stop and start

```sh
k3d cluster stop aux-battles
k3d cluster start aux-battles
```

Data survives a stop. After a start, the API and worker restart a few times
until PostgreSQL is ready. Prometheus and Alertmanager keep no history across
restarts.

Deleting the cluster deletes the database, songs and the Secrets from step 4.
