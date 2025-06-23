
# Installation mit Helm

## Helm Chart Repository hinzufügen

Komplette Liste alle Values: [Chart Repo](https://ruedigerp.github.io/helm-charts/docs/comments/)

```bash
helm repo add ruedigerp https://ruedigerp.github.io/helm-charts/
helm repo update ruedigerp
```

## Installation 

```bash
helm upgrade --install comments ruedigerp/comments --namespace comments --create-namespace --wait -f values.yaml
```

## Example values.yaml 

```yaml

deployment:
  auth_enabled: "true"
  PUBLIC_API_URL: "https://comments.kuepper.nrw"
  DOMAIN: "https://comments.kuepper.nrw"
  admin_token: "" # openssl rand -hex 24

ingress:
  enabled: true
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
  ingressClassName: traefik
  tls: true
  domains:
    - name: comments.example.com
      tls: true

storage:
  size: 1Gi
```

## Update with Helm Chart


```bash
helm upgrade --install comments ruedigerp/comments --namespace comments --create-namespace --wait -f values.yaml
```

