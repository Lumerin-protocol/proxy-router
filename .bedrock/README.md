# Proxy Router Foundation

Terraform/Terragrunt infrastructure for deploying Lumerin Proxy Router to AWS ECS across multiple environments.

## Overview

This repository manages the AWS infrastructure for the Lumerin Proxy Router and Validator services using Terraform and Terragrunt. The actual application code lives in the [proxy-router GitHub repository](https://github.com/lumerin-protocol/proxy-router).

## Architecture

The deployment architecture consists of:

- **Source Code**: GitHub repository (`lumerin-protocol/proxy-router`)
- **Container Registry**: GitHub Container Registry (GHCR)
- **Infrastructure**: Terraform/Terragrunt (this repository)
- **Deployment**: GitHub Actions with AWS OIDC authentication
- **Secrets**: AWS Secrets Manager
- **Compute**: AWS ECS Fargate
- **Networking**: Network Load Balancers (NLB), Route53 DNS

See [.ai-docs/deployment-architecture.md](.ai-docs/deployment-architecture.md) for detailed architecture documentation.

## Environments

| Environment | Directory | AWS Account | Purpose |
|-------------|-----------|-------------|---------|
| Development | `02-dev/` | titanio-dev | Development testing |
| Staging | `03-stg/` | titanio-stg | Pre-production validation |
| Production | `04-lmn/` | titanio-lmn | Production deployment |

## Deployment Flow

```
Code Change → GitHub Push (dev/stg/main)
    ↓
GitHub Actions: Build & Test
    ↓
GitHub Actions: Build & Push Container → GHCR
    ↓
GitHub Actions: Deploy to AWS ECS (via OIDC)
    ↓
AWS ECS: Rolling Deployment with Circuit Breaker
```

## Quick Start

### Prerequisites

- Terraform >= 1.5
- Terragrunt >= 0.48
- AWS CLI configured with appropriate profiles
- Access to AWS accounts (dev/stg/lmn)

### Initial Setup

1. **Clone the repository**
   ```bash
   cd /path/to/proxy-router-foundation
   ```

2. **Configure AWS profiles**
   Ensure you have AWS profiles configured for:
   - `titanio-dev`
   - `titanio-stg` 
   - `titanio-lmn` (production)

3. **Initialize secrets**
   Create `secret.auto.tfvars` in each environment directory with sensitive values:
   ```hcl
   proxy_wallet_private_key      = "0x..."
   validator_wallet_private_key  = "0x..."
   proxy_eth_node_address        = "https://..."
   validator_eth_node_address    = "https://..."
   ```

4. **Deploy infrastructure**
   ```bash
   cd 02-dev
   terragrunt init
   terragrunt plan
   terragrunt apply
   ```

### Deploying Application Updates

Application deployments are **automated** via GitHub Actions:

1. **Development**: Push to `dev` branch in proxy-router repo
2. **Staging**: Push to `stg` branch in proxy-router repo
3. **Production**: Push to `main` branch in proxy-router repo

GitHub Actions will automatically:
- Build and test the application
- Create versioned Docker image
- Deploy to appropriate ECS services
- Validate deployment success

### Manual Infrastructure Updates

To update infrastructure (not application code):

```bash
cd 02-dev  # or 03-stg, 04-lmn
terragrunt plan
terragrunt apply
```

## Infrastructure Components

### ECS Services

Each environment runs two ECS services:

1. **proxy-router**: Main routing/seller service
   - Handles hashrate contract sales
   - Routes miners to buyers
   - Exposed on TCP port 7301 (stratum)

2. **proxy-validator**: Validation service
   - Validates contract execution
   - Performs dispute resolution
   - Separate from router for independence

### Secrets Management

Secrets are stored in AWS Secrets Manager:

```
/proxy-router/{env}/wallet-private-key
/proxy-router/{env}/eth-node-address
/proxy-validator/{env}/wallet-private-key
/proxy-validator/{env}/eth-node-address
```

Terraform creates these secrets from `secret.auto.tfvars` values. GitHub Actions reads them during deployment.

### IAM & Security

- **OIDC Provider**: Enables GitHub Actions to authenticate without long-lived credentials
- **Deployment Role**: `github-actions-proxy-router-deploy-{env}` assumed by GitHub Actions
- **Task Roles**: ECS tasks use `ecsTaskExecutionRole` with minimal required permissions

### Monitoring

- **CloudWatch Logs**: All container output
- **CloudWatch Metrics**: Custom metrics via Lambda functions
- **Dashboards**: Pre-built CloudWatch dashboards per environment
- **Alarms**: Metric filters and alarms for critical events

## Configuration

### Main Variables

Key variables in `terraform.tfvars`:

```hcl
# Environment
account_shortname = "titanio-dev"
account_lifecycle = "dev"
default_region    = "us-east-1"

# Proxy Router
proxy_router = {
  create                 = "true"
  image_tag              = "v1.7.5-dev"  # Terraform reference only
  task_cpu               = "256"
  task_ram               = "512"
  task_worker_qty        = "1"
  pool_address           = "//user:pass@pool.example.com:3333"
  validator_reg          = "0x..."
  clone_factory_address  = "0x..."
  # ... additional config
}

# Proxy Validator
proxy_validator = {
  create                 = "true"
  image_tag              = "v1.7.5-dev"  # Terraform reference only
  task_cpu               = "256"
  task_ram               = "512"
  task_worker_qty        = "1"
  # ... additional config
}
```

**Image Tag Modes**:
- **Auto mode** (recommended): Set `image_tag = "auto"` and Terraform will query GitHub for the latest tag
- **Pinned mode**: Set `image_tag = "v1.7.5-dev"` to pin to a specific version (useful for rollback or testing)
- GitHub Actions deployments update the container image, but Terraform can reference the current version for infrastructure updates

## GitHub Actions Setup

### Required Secrets

Configure these in the proxy-router GitHub repository settings:

**Development Environment:**
- `AWS_ACCOUNT_DEV` - AWS account number
- `AWS_ROLE_ARN_DEV` - IAM role ARN (output from Terraform)

**Staging Environment:**
- `AWS_ACCOUNT_STG` - AWS account number
- `AWS_ROLE_ARN_STG` - IAM role ARN (output from Terraform)

**Production Environment:**
- `AWS_ACCOUNT_LMN` - AWS account number
- `AWS_ROLE_ARN_LMN` - IAM role ARN (output from Terraform)

### Terraform Outputs

After applying Terraform, get the role ARN:

```bash
terragrunt output github_actions_role_arn
```

Add this ARN to GitHub secrets for the corresponding environment.

## Versioning

The project uses semantic versioning:

- **Production (main)**: `v1.8.0`
- **Staging (stg)**: `v1.7.5-stg`
- **Development (dev)**: `v1.7.5-dev`

Versions are automatically generated by GitHub Actions based on:
- Branch name
- Commit count since merge base
- Manual version bumps in workflow config

## Troubleshooting

### Deployment Fails

1. Check GitHub Actions logs in proxy-router repository
2. Verify ECS service events: `aws ecs describe-services --cluster <cluster> --services <service>`
3. Check CloudWatch Logs for container errors
4. Verify secrets are correctly set in Secrets Manager

### Terraform State Locked

```bash
terragrunt force-unlock <lock-id>
```

### Need to Rollback

Option 1: Use GitHub Actions to deploy previous tag
Option 2: Update `image_tag` in `terraform.tfvars` and run `terragrunt apply`

### Container Won't Start

1. Check task definition environment variables
2. Verify secrets are accessible
3. Check security group rules
4. Review ECS task execution role permissions

## Maintenance

### Updating Secrets

1. Update value in AWS Secrets Manager console, or
2. Update `secret.auto.tfvars` and run `terragrunt apply`

### Scaling Services

Update `task_worker_qty` in `terraform.tfvars`:

```hcl
proxy_router = {
  task_worker_qty = "2"  # Scale to 2 tasks
}
```

Then apply:
```bash
terragrunt apply
```

### Destroying Environment

**⚠️ CAUTION: This will destroy all resources!**

```bash
cd 02-dev  # Choose appropriate environment
terragrunt destroy
```

## Repository Structure

```
.
├── .ai-docs/                    # Architecture documentation
├── .terragrunt/                 # Terraform modules
│   ├── 00_*.tf                  # Variables, providers, data sources
│   ├── 01_*.tf                  # Secrets, IAM, OIDC
│   ├── 02_*.tf                  # ECS services, tasks, monitoring
│   ├── 03_*.tf                  # Lambda query functions
│   └── 04_*.tf                  # CloudWatch dashboards
├── 02-dev/                      # Development environment
│   ├── terraform.tfvars         # Environment config
│   ├── secret.auto.tfvars       # Sensitive values (gitignored)
│   └── terragrunt.hcl           # Terragrunt config
├── 03-stg/                      # Staging environment
├── 04-lmn/                      # Production environment
├── root.hcl                     # Terragrunt root config
└── README.md                    # This file
```

## Support

For issues related to:
- **Infrastructure**: Create issue in this repository
- **Application Code**: Create issue in [proxy-router repository](https://github.com/lumerin-protocol/proxy-router)
- **Deployment Issues**: Check GitHub Actions logs and ECS service events

## Contributing

1. Create feature branch
2. Make changes
3. Test in development environment
4. Submit merge request
5. Deploy to staging for validation
6. Deploy to production after approval

## License

See LICENSE file in the repository root.
