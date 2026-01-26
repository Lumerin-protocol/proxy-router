# Proxy Router Deployment Architecture

## Overview
This document describes the streamlined CI/CD architecture for deploying the Lumerin Proxy Router from GitHub directly to AWS ECS environments.

## Architecture Components

### 1. Source Repository (GitHub)
- **Repository**: `lumerin-protocol/proxy-router`
- **Application**: Go-based proxy router service
- **Branches**: 
  - `dev` → deploys to DEV environment
  - `stg` → deploys to STG environment  
  - `main` → deploys to LMN (production) environment

### 2. Infrastructure Repository (GitLab)
- **Repository**: `TitanInd/bedrock/foundation-afs/proxy-router-foundation`
- **Purpose**: Terraform/Terragrunt infrastructure definitions
- **Environments**: `02-dev`, `03-stg`, `04-lmn`

### 3. CI/CD Pipeline (GitHub Actions)

#### Build Phase
1. Generate semantic version tag based on branch
2. Build Docker image for multiple platforms (linux/amd64, linux/arm64)
3. Run health check tests
4. Build OS-specific binaries (Linux, Darwin, Windows)
5. Create GitHub release with artifacts

#### Deploy Phase
1. Authenticate to AWS using OIDC (no long-lived credentials)
2. Retrieve deployment secrets from AWS Secrets Manager
3. Update ECS task definition with new container image
4. Deploy updated task to ECS service
5. Verify deployment success

### 4. AWS Resources

#### ECS Services
- **proxy-router**: Main seller/router service
- **proxy-validator**: Validator service

#### Secrets Management
All sensitive configuration stored in AWS Secrets Manager as JSON objects:
- `/proxy-router/{env}/config` - Contains `wallet_private_key`, `eth_node_address`
- `/proxy-validator/{env}/config` - Contains `wallet_private_key`, `eth_node_address`

**Security Note**: Secrets are pulled at **runtime by ECS tasks** using the task execution role, NOT by GitHub Actions. This follows AWS best practices and minimizes secret exposure.

#### IAM Configuration
- **OIDC Provider**: Shared across projects (reused if exists from proxy-ui-foundation)
- **Deployment Role**: Assumed by GitHub Actions (per environment), permissions:
  - Describe/Register ECS task definitions
  - Update ECS services
  - Pass IAM roles to ECS tasks
- **Task Execution Role**: `bedrock-foundation-role` permissions:
  - Pull container images from GHCR
  - Read secrets from Secrets Manager (added by this module)
  - Write logs to CloudWatch

## Deployment Flow

```
GitHub Push → Branch (dev/stg/main)
    ↓
GitHub Actions: Build & Test
    ↓
GitHub Actions: Build & Push Docker Image → GHCR
    ↓
GitHub Actions: Deploy to AWS
    ↓ (Assume Role via OIDC - environment-specific)
AWS: Get Current Task Definition
    ↓
AWS: Update Image Tag Only (secrets unchanged)
    ↓
AWS: Register New Task Definition
    ↓
AWS: Update ECS Service
    ↓
ECS: Pull New Container
    ↓
ECS: Pull Secrets from Secrets Manager (at runtime)
    ↓
ECS: Rolling Update (circuit breaker enabled)
```

## Version Management

### Semantic Versioning
- **main branch**: `v{major}.{minor}.{patch}` (e.g., `v1.8.0`)
- **stg branch**: `v{major}.{minor}.{patch}-stg` (e.g., `v1.7.5-stg`)
- **dev branch**: `v{major}.{minor}.{patch}-dev` (e.g., `v1.7.5-dev`)

### Container Tags
- Images stored in GHCR: `ghcr.io/lumerin-protocol/proxy-router:{tag}`
- Terraform queries latest tag per environment via GitHub API
- Task definitions support two modes:
  - **Auto mode**: Set `image_tag = "auto"` in tfvars to use latest from GitHub
  - **Pinned mode**: Set `image_tag = "v1.7.5-dev"` to pin to specific version (for rollback/testing)
- Most deployments use "auto" to automatically track GitHub releases

## Security

### Authentication
- **GitHub → AWS**: OIDC federation (no access keys)
  - Each environment trusts only its specific branch
  - dev account trusts `dev` branch
  - stg account trusts `stg` branch  
  - lmn account trusts `main` branch
- **ECS Tasks**: IAM roles with least-privilege policies

### Secrets
- Secrets stored as JSON objects in AWS Secrets Manager
- **NOT read by GitHub Actions** - only by ECS tasks at runtime
- ECS tasks pull secrets using task execution role
- Secrets never appear in task definition or environment variables
- Supports automatic rotation via Secrets Manager

## Terraform Management

### Container Image Updates
Terraform tracks desired container version but uses `lifecycle.ignore_changes` on:
- `task_definition` - allows GitHub Actions to update without Terraform conflicts
- `desired_count` - preserves manual scaling adjustments

### Manual Rollback
If needed, can update `image_tag` in `terraform.tfvars` and apply to roll back to specific version.

## Monitoring
- CloudWatch Logs: All container output
- CloudWatch Metrics: Custom metrics via Lambda queries
- ECS Circuit Breaker: Auto-rollback on failed deployments

