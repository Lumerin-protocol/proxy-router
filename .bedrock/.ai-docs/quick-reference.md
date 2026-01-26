# Quick Reference Guide

Fast reference for common operations with the Lumerin Proxy Router infrastructure.

## Deployment Commands

### Deploy New Application Version

**Automatic (Recommended)**:
```bash
# In proxy-router repository
git checkout dev  # or stg, or main
git pull
git commit --allow-empty -m "Deploy latest version"
git push origin dev
```

GitHub Actions handles the rest automatically.

### Check Deployment Status

```bash
# GitHub Actions
open https://github.com/lumerin-protocol/proxy-router/actions

# AWS ECS
aws ecs describe-services \
  --cluster ecs-proxy-router-dev-use1 \
  --services svc-proxy-router-dev-use1 \
  --region us-east-1 \
  --query 'services[0].deployments'
```

## Infrastructure Commands

### Apply Terraform Changes

```bash
cd 02-dev  # or 03-stg, 04-lmn
terragrunt plan
terragrunt apply
```

### View Outputs

```bash
terragrunt output
terragrunt output github_actions_role_arn
terragrunt output proxy_router_miner_target
```

### Update State

```bash
terragrunt refresh
```

## ECS Operations

### List Running Tasks

```bash
aws ecs list-tasks \
  --cluster ecs-proxy-router-dev-use1 \
  --service-name svc-proxy-router-dev-use1 \
  --region us-east-1
```

### Describe Service

```bash
aws ecs describe-services \
  --cluster ecs-proxy-router-dev-use1 \
  --services svc-proxy-router-dev-use1 svc-proxy-validator-dev-use1 \
  --region us-east-1
```

### Force New Deployment

```bash
aws ecs update-service \
  --cluster ecs-proxy-router-dev-use1 \
  --service svc-proxy-router-dev-use1 \
  --force-new-deployment \
  --region us-east-1
```

### Scale Service

```bash
aws ecs update-service \
  --cluster ecs-proxy-router-dev-use1 \
  --service svc-proxy-router-dev-use1 \
  --desired-count 2 \
  --region us-east-1
```

### Stop Task (Force Restart)

```bash
# List tasks first
aws ecs list-tasks \
  --cluster ecs-proxy-router-dev-use1 \
  --service-name svc-proxy-router-dev-use1 \
  --region us-east-1

# Stop specific task (ECS will start a new one)
aws ecs stop-task \
  --cluster ecs-proxy-router-dev-use1 \
  --task <task-id> \
  --region us-east-1
```

## Secrets Management

### View Secret Names

```bash
aws secretsmanager list-secrets \
  --region us-east-1 \
  --query 'SecretList[?contains(Name, `proxy-router`) || contains(Name, `proxy-validator`)].Name'
```

### Get Secret Value (JSON)

```bash
# Get entire secret
aws secretsmanager get-secret-value \
  --secret-id "/proxy-router/dev/config" \
  --region us-east-1 \
  --query SecretString --output text | jq .

# Get specific field
aws secretsmanager get-secret-value \
  --secret-id "/proxy-router/dev/config" \
  --region us-east-1 \
  --query SecretString --output text | jq -r '.wallet_private_key'
```

### Update Secret (JSON)

```bash
# Update wallet key only
CURRENT=$(aws secretsmanager get-secret-value \
  --secret-id "/proxy-router/dev/config" \
  --query SecretString --output text)

UPDATED=$(echo $CURRENT | jq '.wallet_private_key = "0xNEW_KEY"')

aws secretsmanager put-secret-value \
  --secret-id "/proxy-router/dev/config" \
  --secret-string "$UPDATED" \
  --region us-east-1

# Note: After updating secrets, restart ECS tasks to pick up new values
aws ecs update-service \
  --cluster ecs-proxy-router-dev-use1 \
  --service svc-proxy-router-dev-use1 \
  --force-new-deployment \
  --region us-east-1
```

### Update via Terraform

```bash
# Edit secret.auto.tfvars
vim 02-dev/secret.auto.tfvars

# Apply changes
cd 02-dev
terragrunt apply
```

## Logging & Monitoring

### View Recent Logs

```bash
# Proxy Router
aws logs tail /aws/ecs/proxy-router --follow --region us-east-1

# Proxy Validator
aws logs tail /aws/ecs/proxy-validator --follow --region us-east-1
```

### View Specific Time Range

```bash
aws logs filter-log-events \
  --log-group-name /aws/ecs/proxy-router \
  --start-time $(date -u -d '1 hour ago' +%s)000 \
  --region us-east-1
```

### Search Logs

```bash
aws logs filter-log-events \
  --log-group-name /aws/ecs/proxy-router \
  --filter-pattern "ERROR" \
  --region us-east-1
```

### View CloudWatch Dashboard

```bash
# Open in browser
open https://console.aws.amazon.com/cloudwatch/home?region=us-east-1#dashboards:name=proxy-router-dev
```

## Application Health Checks

### Proxy Router Health

```bash
# Development
curl http://proxyapi.dev.lumerin.io:8080/healthcheck

# Staging  
curl http://proxyapi.stg.lumerin.io:8080/healthcheck

# Production
curl http://proxyapi.lumerin.io:8080/healthcheck
```

### Validator Health

```bash
# Development
curl http://validatorapi.dev.lumerin.io:8080/healthcheck

# Staging
curl http://validatorapi.stg.lumerin.io:8080/healthcheck

# Production
curl http://validatorapi.lumerin.io:8080/healthcheck
```

### Detailed Stats

```bash
curl http://proxyapi.dev.lumerin.io:8080/stats
curl http://proxyapi.dev.lumerin.io:8080/contracts
```

## Docker Image Information

### List Available Tags

```bash
# Via GitHub API
curl -H "Accept: application/vnd.github+json" \
  https://api.github.com/orgs/lumerin-protocol/packages/container/proxy-router/versions \
  | jq -r '.[].metadata.container.tags[]' | head -20
```

### Pull Specific Version

```bash
docker pull ghcr.io/lumerin-protocol/proxy-router:v1.7.5-dev
```

### Inspect Image

```bash
docker inspect ghcr.io/lumerin-protocol/proxy-router:v1.7.5-dev
```

## Network & DNS

### Check DNS Resolution

```bash
# Proxy Router endpoints
dig proxy.dev.lumerin.io
dig proxyapi.dev.lumerin.io

# Validator endpoints
dig validator.dev.lumerin.io
dig validatorapi.dev.lumerin.io
```

### Test Port Connectivity

```bash
# Stratum mining port
nc -zv proxy.dev.lumerin.io 7301

# API port (via VPN)
nc -zv proxyapi.dev.lumerin.io 8080
```

### List Load Balancers

```bash
aws elbv2 describe-load-balancers \
  --region us-east-1 \
  --query 'LoadBalancers[?contains(LoadBalancerName, `proxy-router`)].LoadBalancerArn'
```

## IAM & OIDC

### View OIDC Provider

```bash
aws iam list-open-id-connect-providers
```

### View GitHub Actions Role

```bash
aws iam get-role \
  --role-name github-actions-proxy-router-deploy-dev \
  | jq .Role.AssumeRolePolicyDocument
```

### Test OIDC Authentication (GitHub Actions)

Check in GitHub Actions workflow logs under "Configure AWS Credentials" step.

## Common Environment Variables

### Development (02-dev)

```bash
export AWS_PROFILE=titanio-dev
export AWS_REGION=us-east-1
export TF_ENV=dev
export CLUSTER=ecs-proxy-router-dev-use1
```

### Staging (03-stg)

```bash
export AWS_PROFILE=titanio-stg
export AWS_REGION=us-east-1
export TF_ENV=stg
export CLUSTER=ecs-proxy-router-stg-use1
```

### Production (04-lmn)

```bash
export AWS_PROFILE=titanio-lmn
export AWS_REGION=us-east-1
export TF_ENV=lmn
export CLUSTER=ecs-proxy-router-lmn-use1
```

## Troubleshooting Quick Checks

### 1. Is the service running?

```bash
aws ecs describe-services \
  --cluster $CLUSTER \
  --services svc-proxy-router-$TF_ENV-use1 \
  --query 'services[0].{Status:status,Running:runningCount,Desired:desiredCount}'
```

### 2. Are tasks healthy?

```bash
aws ecs describe-tasks \
  --cluster $CLUSTER \
  --tasks $(aws ecs list-tasks --cluster $CLUSTER --service svc-proxy-router-$TF_ENV-use1 --query 'taskArns[0]' --output text) \
  --query 'tasks[0].{Health:healthStatus,Status:lastStatus}'
```

### 3. What's in the logs?

```bash
aws logs tail /aws/ecs/proxy-router --follow
```

### 4. What's the current image?

```bash
aws ecs describe-task-definition \
  --task-definition tsk-proxy-router \
  --query 'taskDefinition.containerDefinitions[0].image'
```

### 5. When was it last deployed?

```bash
aws ecs describe-services \
  --cluster $CLUSTER \
  --services svc-proxy-router-$TF_ENV-use1 \
  --query 'services[0].deployments[0].{Created:createdAt,Status:status,Image:taskDefinition}'
```

## Emergency Procedures

### Rollback to Previous Version

**Option 1: Redeploy previous tag via GitHub**
```bash
cd /path/to/proxy-router
git checkout dev
git revert HEAD  # Revert the problematic commit
git push origin dev
```

**Option 2: Manual ECS task definition rollback**
```bash
# List recent task definitions
aws ecs list-task-definitions \
  --family-prefix tsk-proxy-router \
  --sort DESC \
  --max-items 5

# Update service to use previous task definition
aws ecs update-service \
  --cluster $CLUSTER \
  --service svc-proxy-router-$TF_ENV-use1 \
  --task-definition tsk-proxy-router:PREVIOUS_REVISION
```

### Scale to Zero (Emergency Stop)

```bash
aws ecs update-service \
  --cluster $CLUSTER \
  --service svc-proxy-router-$TF_ENV-use1 \
  --desired-count 0
```

### Scale Back Up

```bash
aws ecs update-service \
  --cluster $CLUSTER \
  --service svc-proxy-router-$TF_ENV-use1 \
  --desired-count 1
```

## Useful Aliases

Add to your `~/.zshrc` or `~/.bashrc`:

```bash
alias tf='terragrunt'
alias tfp='terragrunt plan'
alias tfa='terragrunt apply'
alias tfo='terragrunt output'

alias ecs-dev='aws ecs --region us-east-1 --profile titanio-dev'
alias ecs-stg='aws ecs --region us-east-1 --profile titanio-stg'
alias ecs-lmn='aws ecs --region us-east-1 --profile titanio-lmn'

alias logs-proxy='aws logs tail /aws/ecs/proxy-router --follow'
alias logs-validator='aws logs tail /aws/ecs/proxy-validator --follow'
```

## Key URLs

- **GitHub Repo**: https://github.com/lumerin-protocol/proxy-router
- **GitHub Actions**: https://github.com/lumerin-protocol/proxy-router/actions
- **GHCR**: https://github.com/orgs/lumerin-protocol/packages/container/package/proxy-router
- **AWS Console**: https://console.aws.amazon.com/ecs/v2/clusters
- **CloudWatch**: https://console.aws.amazon.com/cloudwatch/home?region=us-east-1

## Version Reference

| Environment | Current Version | Branch | Container Tag | TFVars Setting |
|-------------|----------------|--------|---------------|----------------|
| Development | v1.7.5-dev | dev | v1.7.5-dev | `image_tag = "auto"` |
| Staging | v1.7.5-stg | stg | v1.7.5-stg | `image_tag = "auto"` |
| Production | v1.8.0 | main | v1.8.0 | `image_tag = "auto"` |

**Note**: Set `image_tag = "auto"` (recommended) to automatically use the latest GitHub tag, or set a specific version like `image_tag = "v1.7.5-dev"` to pin.

## Support Contacts

- Infrastructure Issues: Create issue in proxy-router-foundation repo
- Application Issues: Create issue in proxy-router repo
- Emergency: Contact DevOps team via standard channels

