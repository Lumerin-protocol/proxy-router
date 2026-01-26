# Proxy Router Contracts

This directory contains the smart contracts for the proxy-router, specifically the `ValidatorRegistry` contract.

## Directory Structure

```
contracts/
├── validator-registry/     # Solidity contracts
│   ├── ValidatorRegistry.sol
│   └── EC.sol
├── util/                   # Shared Solidity utilities
│   └── Versionable.sol
├── scripts/                # Deployment & management scripts
│   ├── deploy-validator-registry.ts
│   ├── update-validator-registry.ts
│   └── lib/               # Script utilities
├── lib/                    # TypeScript utilities
│   └── utils.ts
├── hardhat.config.ts       # Hardhat configuration
├── foundry.toml           # Foundry configuration (for formatting)
├── package.json           # Node.js dependencies
├── tsconfig.json          # TypeScript configuration
├── biome.json             # Biome linter configuration
├── Makefile               # Build targets
└── VERSION                # Contract version
```

## Prerequisites

- Node.js 20.x
- Yarn package manager
- Go (for building Go bindings)
- [Foundry](https://book.getfoundry.sh/) (optional, for Solidity formatting)
- [abigen](https://geth.ethereum.org/docs/developers/dapp-developer/native-bindings) (for Go binding generation)

## Setup

```bash
cd contracts
yarn install
```

## Commands

### Using Yarn

```bash
# Compile contracts
yarn compile

# Clean build artifacts
yarn clean

# Format Solidity files (requires forge)
yarn format:sol

# Lint TypeScript files
yarn lint

# Deploy ValidatorRegistry
yarn deploy:validator-registry

# Update ValidatorRegistry
yarn update:validator-registry

# Build Go bindings
yarn build:go
```

### Using Make

```bash
# Install dependencies
make install

# Compile contracts
make compile

# Clean build artifacts
make clean

# Deploy ValidatorRegistry
make deploy-validator-registry

# Update ValidatorRegistry
make update-validator-registry

# Build Go bindings
make build-go

# Format Solidity files
make format

# Lint TypeScript files
make lint
```

## Environment Variables

For deployment scripts, set the following environment variables (or use a `.env` file):

### Deploy ValidatorRegistry

- `OWNER_PRIVATEKEY` - Private key of the deployer
- `LUMERIN_TOKEN_ADDRESS` - Address of the LMR token contract
- `VALIDATOR_STAKE_MINIMUM` - Minimum stake to be considered active
- `VALIDATOR_STAKE_REGISTER` - Stake required to register as validator
- `VALIDATOR_PUNISH_AMOUNT` - Amount to slash on punishment
- `VALIDATOR_PUNISH_THRESHOLD` - Number of complaints before punishment
- `SAFE_OWNER_ADDRESS` (optional) - Address to transfer ownership to

### Update ValidatorRegistry

- `VALIDATOR_REGISTRY_ADDRESS` - Address of the deployed proxy

## Building Go Bindings

The `build-go.sh` script generates Go bindings from the contract ABIs:

```bash
# First compile contracts to generate ABIs
yarn compile

# Then generate Go bindings
yarn build:go
```

This creates a `build-go/` directory with Go packages that can be used in the proxy-router application.

## CI/CD - Automated Go Binding Releases

A GitHub Actions workflow automatically builds and releases Go bindings to the [contracts-go](https://github.com/Lumerin-protocol/contracts-go) repository.

### Automatic Triggers

The workflow runs automatically when changes are pushed to `main` that affect:
- `contracts/validator-registry/**`
- `contracts/util/**`

### Manual Triggers

You can also trigger a release manually from the GitHub Actions UI:

1. Go to **Actions** → **Contracts Go Release**
2. Click **Run workflow**
3. Select version bump type:
   - `patch` (default): `1.0.0` → `1.0.1`
   - `minor`: `1.0.0` → `1.1.0`
   - `major`: `1.0.0` → `2.0.0`
4. Optionally enable **Dry run** to test without pushing

### Required Secrets

The workflow requires a **Personal Access Token (PAT)** to push to the contracts-go repository:

1. Create a PAT with `repo` scope at [GitHub Settings → Developer settings → Personal access tokens](https://github.com/settings/tokens)
2. Add it as a repository secret named `CONTRACTS_GO_PAT`

### What the Workflow Does

1. Compiles Solidity contracts using Hardhat
2. Generates Go bindings using `abigen`
3. Fetches the current version from contracts-go repo
4. Increments the semver automatically
5. Pushes to contracts-go with the new version tag

## Self-Contained Design

This contracts directory is designed to be self-contained within the proxy-router monorepo:

- Has its own `package.json` with dedicated dependencies
- Has its own `Makefile` (doesn't conflict with root Makefile)
- All paths in scripts are relative to this directory
- Can be developed and built independently
