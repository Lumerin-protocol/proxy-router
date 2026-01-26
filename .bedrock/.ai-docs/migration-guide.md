# Migration Guide: GitLab CI/CD to GitHub Actions

This guide walks through migrating from GitLab-based deployments to GitHub Actions with direct AWS deployment.

## Pre-Migration Checklist

- [ ] Backup current Terraform state
- [ ] Document current secrets/variables from GitLab CI/CD
- [ ] Verify AWS account access (dev, stg, lmn)
- [ ] Test in dev environment first
- [ ] Notify team of deployment changes

## Step 1: Prepare Secret Values

Create `secret.auto.tfvars` in each environment directory (`02-dev/`, `03-stg/`, `04-lmn/`):

```hcl
# Proxy Router Secrets
proxy_wallet_private_key = "0x..."
proxy_eth_node_address   = "https://..."

# Proxy Validator Secrets  
validator_wallet_private_key = "0x..."
validator_eth_node_address   = "https://..."

# Monitoring Secrets (if enabled)
eth_api_key              = "..."
foreman_api_key          = "..."
ghissues_query_authtoken = "..."
```

**Important**: These files are git-ignored and contain sensitive data. Store securely!

## Step 2: Deploy Terraform Changes to DEV

```bash
cd 02-dev

# Initialize Terragrunt
terragrunt init

# Review planned changes
terragrunt plan

# Look for:
# - New aws_secretsmanager_secret resources (4 total)
# - New aws_iam_openid_connect_provider resource
# - New aws_iam_role resource (github-actions-proxy-router-deploy-dev)
# - New aws_iam_role_policy resource
```

Expected resources to be created:
- 4x Secrets Manager secrets (proxy-router & proxy-validator wallet + eth node)
- 1x IAM OIDC provider for GitHub
- 1x IAM role for GitHub Actions
- 1x IAM role policy

```bash
# Apply changes
terragrunt apply
```

## Step 3: Capture Terraform Outputs

After applying, get the GitHub Actions role ARN:

```bash
terragrunt output github_actions_role_arn
```

Save this ARN - you'll need it for GitHub secrets.

Example output:
```
arn:aws:iam::434960487817:role/github-actions-proxy-router-deploy-dev
```

## Step 4: Configure GitHub Secrets

In the **proxy-router GitHub repository** (not GitLab), add repository secrets:

### For DEV Environment:
- Name: `AWS_ACCOUNT_DEV`
- Value: `434960487817` (or your dev account number)

- Name: `AWS_ROLE_ARN_DEV`  
- Value: `arn:aws:iam::434960487817:role/github-actions-proxy-router-deploy-dev`

### For STG Environment:
Repeat Step 2-4 in `03-stg/` directory, then add:
- `AWS_ACCOUNT_STG`
- `AWS_ROLE_ARN_STG`

### For LMN (Production) Environment:
Repeat Step 2-4 in `04-lmn/` directory, then add:
- `AWS_ACCOUNT_LMN`
- `AWS_ROLE_ARN_LMN`

**Important Security Note**: These are the ONLY GitHub secrets needed. GitHub Actions does NOT have access to wallet keys or ETH node addresses. Those secrets are pulled at runtime by ECS tasks using the task execution role.

## Step 5: Test GitHub Actions Workflow

### Option A: Test with Existing Branch

If there's a recent commit on `dev` branch:

```bash
cd /path/to/proxy-router
git checkout dev

# Trigger workflow by pushing (even empty commit)
git commit --allow-empty -m "Test GitHub Actions deployment"
git push origin dev
```

### Option B: Test with PR

1. Create a test branch from `dev`
2. Make a trivial change
3. Push and create PR
4. Verify build phase completes
5. Merge to `dev` to trigger deployment

## Step 6: Verify Deployment

### Check GitHub Actions

1. Go to https://github.com/lumerin-protocol/proxy-router/actions
2. Find the latest workflow run
3. Verify all jobs pass:
   - Generate-Tag ✅
   - Build-Test ✅
   - OS-Build ✅
   - Release ✅
   - GHCR-Build-and-Push ✅
   - **AWS-Deploy** ✅ (new)

### Check AWS ECS

```bash
# Check ECS service status
aws ecs describe-services \
  --cluster ecs-proxy-router-dev-use1 \
  --services svc-proxy-router-dev-use1 \
  --region us-east-1

# Check task health
aws ecs list-tasks \
  --cluster ecs-proxy-router-dev-use1 \
  --service-name svc-proxy-router-dev-use1 \
  --region us-east-1
```

### Check CloudWatch Logs

```bash
# View recent logs
aws logs tail /aws/ecs/proxy-router --follow --region us-east-1
```

### Test Application Endpoints

```bash
# Health check
curl http://proxyapi.dev.lumerin.io:8080/healthcheck

# Expected response:
# {"status":"ok","version":"v1.7.5-dev"}
```

## Step 7: Migrate Staging Environment

Once dev is stable, repeat for staging:

```bash
cd ../03-stg
terragrunt init
terragrunt plan
terragrunt apply
terragrunt output github_actions_role_arn
```

Add GitHub secrets:
- `AWS_ACCOUNT_STG`
- `AWS_ROLE_ARN_STG`

Test deployment by pushing to `stg` branch.

## Step 8: Migrate Production Environment

⚠️ **CRITICAL**: Only proceed after successful dev and stg deployments!

```bash
cd ../04-lmn
terragrunt init
terragrunt plan
terragrunt apply
terragrunt output github_actions_role_arn
```

Add GitHub secrets:
- `AWS_ACCOUNT_LMN`
- `AWS_ROLE_ARN_LMN`

Test deployment by pushing to `main` branch.

## Step 9: Cleanup GitLab CI/CD (Already Done)

The `.gitlab-ci.yml` file has been removed from this repository. 

### Optional: Disable GitLab Runners

If you have dedicated GitLab runners for this project:

1. Go to GitLab project → Settings → CI/CD → Runners
2. Disable or remove project-specific runners
3. Document their configuration if you may need to restore

### Remove GitLab Secrets (Optional)

You may want to keep these temporarily in case of rollback:

- `GITLAB_TRIGGER_URL`
- `GITLAB_TRIGGER_TOKEN`
- `SELLER_PRIVATEKEY`
- `VALIDATOR_PRIVATEKEY`
- `PROXY_ROUTER_ETH_NODE_ADDRESS`
- `VALIDATOR_ETH_NODE_ADDRESS`

## Step 10: Update Team Documentation

- [ ] Update internal runbooks
- [ ] Train team on new deployment process
- [ ] Update incident response procedures
- [ ] Document rollback procedures

## Rollback Procedure

If you need to rollback to GitLab CI/CD:

### 1. Restore .gitlab-ci.yml

```bash
git checkout <previous-commit> -- .gitlab-ci.yml
git add .gitlab-ci.yml
git commit -m "Restore GitLab CI/CD temporarily"
git push
```

### 2. Temporarily Disable GitHub Actions

In proxy-router repo, edit `.github/workflows/build.yml`:

```yaml
on:
  push:
    branches: [] # Empty array disables workflow
```

### 3. Trigger GitLab Manually

Use GitLab's pipeline trigger or push to GitLab-connected branch.

## Common Issues

### Issue: GitHub Actions can't assume AWS role

**Symptoms**: 
```
Error: Unable to assume role arn:aws:iam::...:role/github-actions-proxy-router-deploy-dev
```

**Solutions**:
1. Verify OIDC provider thumbprints are correct
2. Check IAM role trust policy includes correct repository
3. Ensure GitHub Actions has `id-token: write` permission
4. Verify repository and branch names match trust conditions

### Issue: Cannot read secrets

**Symptoms**:
```
Error: Secret not found or access denied
```

**Solutions**:
1. Verify secrets were created by Terraform: `/proxy-router/{env}/config` and `/proxy-validator/{env}/config`
2. Check ECS task execution role (`bedrock-foundation-role`) has `secretsmanager:GetSecretValue` permission
3. Verify the IAM policy in `01_ecs_secrets_policy.tf` was applied
4. Check CloudWatch logs for specific error messages from ECS tasks

### Issue: ECS task won't start

**Symptoms**: Tasks fail health checks or won't start

**Solutions**:
1. Check CloudWatch Logs for container errors
2. Verify environment variables are set correctly
3. Check security group rules
4. Verify secrets contain valid values

### Issue: Deployment timeout

**Symptoms**: ECS service update takes too long

**Solutions**:
1. Check ECS circuit breaker events
2. Review task health check configuration
3. Verify container image is accessible
4. Check task has enough CPU/memory

## Verification Checklist

After migration is complete, verify:

- [ ] Dev environment deploys successfully from `dev` branch
- [ ] Stg environment deploys successfully from `stg` branch
- [ ] Lmn environment deploys successfully from `main` branch
- [ ] Health checks pass in all environments
- [ ] CloudWatch logs show container output
- [ ] Metrics are being collected
- [ ] DNS entries resolve correctly
- [ ] Application functionality works (mining, contracts, etc.)
- [ ] Monitoring and alerting still function
- [ ] Team can access and understand new deployment process

## Benefits Realized

After migration:

✅ No long-lived AWS credentials in CI/CD
✅ Secrets stored securely in AWS Secrets Manager
✅ Faster deployments (no GitLab trigger delay)
✅ Single source of truth (GitHub)
✅ Better audit trail via CloudTrail
✅ Reduced infrastructure complexity
✅ Consistent with other projects (Morpheus pattern)

## Support

If you encounter issues during migration:

1. Check GitHub Actions logs first
2. Review CloudWatch Logs for container issues
3. Check AWS ECS service events
4. Consult `.ai-docs/deployment-architecture.md`
5. If stuck, you can temporarily rollback using the procedure above

