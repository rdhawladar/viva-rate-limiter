# Kubernetes Manifests Structure

## Directory Layout

```
k8s/
├── base/                   # Base configurations (shared across environments)
│   ├── deployment.yaml     # API deployment & service
│   ├── postgres.yaml       # PostgreSQL deployment (for dev/stage)
│   └── redis.yaml         # Redis deployment (for dev/stage)
└── environments/
    ├── dev/
    │   ├── namespace.yaml     # Dev namespace
    │   ├── config.yaml        # Dev ConfigMap & Secrets
    │   └── kustomization.yaml # Kustomize config for dev
    ├── stage/
    │   ├── namespace.yaml     # Stage namespace
    │   ├── config.yaml        # Stage ConfigMap & Secrets
    │   └── kustomization.yaml # Kustomize config for stage
    └── prod/
        ├── namespace.yaml     # Prod namespace
        ├── config.yaml        # Prod ConfigMap & Secrets
        ├── kustomization.yaml # Kustomize config for prod
        └── postgres-pvc.yaml  # Optional: PVC for postgres if not using RDS

```

## Environment Configurations

### Development (dev)
- Uses in-cluster PostgreSQL and Redis
- 1 API replica
- No persistent storage (uses emptyDir)
- SSL disabled for databases
- Debug logging enabled

### Staging (stage)
- Uses in-cluster PostgreSQL and Redis
- 2 API replicas
- No persistent storage (uses emptyDir)
- SSL enabled for databases
- Info logging level

### Production (prod)
- **Recommended**: Use managed services (AWS RDS, ElastiCache)
- 3 API replicas
- Persistent storage with PVC (if using in-cluster postgres)
- SSL required for databases
- Health checks enabled
- Warning logging level

## Deployment Commands

### Using Kustomize (Recommended)

```bash
# Deploy to dev
kubectl apply -k k8s/environments/dev/

# Deploy to stage
kubectl apply -k k8s/environments/stage/

# Deploy to prod
kubectl apply -k k8s/environments/prod/
```

### Manual Deployment

```bash
# Dev environment
kubectl apply -f k8s/environments/dev/namespace.yaml
kubectl apply -f k8s/base/ -n dev
kubectl apply -f k8s/environments/dev/config.yaml

# Stage environment
kubectl apply -f k8s/environments/stage/namespace.yaml
kubectl apply -f k8s/base/ -n stage
kubectl apply -f k8s/environments/stage/config.yaml

# Prod environment (API only, assuming managed DB/Cache)
kubectl apply -f k8s/environments/prod/namespace.yaml
kubectl apply -f k8s/base/deployment.yaml -n prod
kubectl apply -f k8s/environments/prod/config.yaml
```

## Production Considerations

### Using Managed Services (Recommended for Production)

For production, it's recommended to use:
- **AWS RDS** for PostgreSQL instead of in-cluster postgres
- **AWS ElastiCache** for Redis instead of in-cluster redis

To configure:

1. Create RDS and ElastiCache instances via Terraform
2. Update `k8s/environments/prod/config.yaml` with the connection strings:

```yaml
stringData:
  DATABASE_URL: "postgresql://user:pass@your-rds-endpoint.amazonaws.com:5432/ratelimiter?sslmode=require"
  REDIS_URL: "redis://your-elasticache-endpoint.amazonaws.com:6379"
```

3. Comment out postgres and redis from `k8s/environments/prod/kustomization.yaml`

### Using In-Cluster Databases (Testing Only)

If you want to test with in-cluster databases in production:

1. Uncomment these lines in `k8s/environments/prod/kustomization.yaml`:
```yaml
  - ../../base/postgres.yaml
  - ../../base/redis.yaml
```

2. Apply the PVC for persistent storage:
```bash
kubectl apply -f k8s/environments/prod/postgres-pvc.yaml
```

## Secrets Management

### Current Setup (Basic)
Secrets are defined in plaintext in `config.yaml` files. This is fine for dev/stage but NOT for production.

### Production Setup (Secure)
For production, use one of these approaches:

1. **AWS Secrets Manager with External Secrets Operator**:
```bash
# Install External Secrets Operator
helm repo add external-secrets https://charts.external-secrets.io
helm install external-secrets external-secrets/external-secrets -n external-secrets-system --create-namespace
```

2. **Sealed Secrets**:
```bash
# Install Sealed Secrets controller
kubectl apply -f https://github.com/bitnami-labs/sealed-secrets/releases/download/v0.18.0/controller.yaml
```

3. **Manual Secret Creation** (minimum):
```bash
# Create secrets manually without storing in Git
kubectl create secret generic viva-secrets \
  --from-literal=DATABASE_URL='your-real-connection-string' \
  --from-literal=REDIS_URL='your-real-redis-url' \
  -n prod
```

## Monitoring

### View Deployments
```bash
kubectl get deployments -n dev
kubectl get deployments -n stage
kubectl get deployments -n prod
```

### View Pods
```bash
kubectl get pods -n dev
kubectl get pods -n stage
kubectl get pods -n prod
```

### View Services
```bash
kubectl get services -n dev
kubectl get services -n stage
kubectl get services -n prod
```

### View Logs
```bash
# Dev
kubectl logs -f deployment/viva-api -n dev

# Stage
kubectl logs -f deployment/viva-api -n stage

# Prod
kubectl logs -f deployment/viva-api -n prod
```

## Troubleshooting

### Pod not starting
```bash
kubectl describe pod <pod-name> -n <environment>
kubectl logs <pod-name> -n <environment>
```

### Database connection issues
```bash
# Check if database pod is running
kubectl get pod -l app=postgres -n <environment>

# Check database service
kubectl get svc postgres-service -n <environment>

# Test connection from API pod
kubectl exec -it deployment/viva-api -n <environment> -- /bin/sh
```

### Redis connection issues
```bash
# Check if redis pod is running
kubectl get pod -l app=redis -n <environment>

# Check redis service
kubectl get svc redis-service -n <environment>
```

## Clean Up

### Remove specific environment
```bash
kubectl delete namespace dev
kubectl delete namespace stage
kubectl delete namespace prod
```

### Remove everything
```bash
kubectl delete -k k8s/environments/dev/
kubectl delete -k k8s/environments/stage/
kubectl delete -k k8s/environments/prod/
```