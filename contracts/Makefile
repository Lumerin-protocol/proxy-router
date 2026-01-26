version := $(shell cat ./VERSION 2>/dev/null || echo "0.0.0")

.PHONY: clean compile deploy-validator-registry update-validator-registry build-go release-go release-git

# Install dependencies
install:
	yarn install

# Clean build artifacts
clean:
	rm -rf abi artifacts cache build-go build-js typechain typechain-types

# Compile contracts
compile:
	yarn hardhat compile

# Deploy ValidatorRegistry contract
deploy-validator-registry:
	yarn hardhat run --network default ./scripts/deploy-validator-registry.ts

# Update ValidatorRegistry contract
update-validator-registry:
	yarn hardhat run --network default ./scripts/update-validator-registry.ts

# Build Go bindings from ABI
build-go:
	./build-go.sh

# Release Go bindings to contracts-go repo
release-go:
	make release-git path="build-go" remote="git@github.com:Lumerin-protocol/contracts-go.git"

# Helper target for releasing to git
release-git:
	cd $(path) \
		&& rm -rf .git \
		&& git init \
		&& git checkout -b main \
		&& git config --local user.email "lumerin@titan.io" \
		&& git config --local user.name "Lumerin Bot" \
		&& git remote add origin $(remote) \
		&& git add . \
		&& git commit -m "release: $(version)" \
		&& git fetch origin main \
		&& git merge origin/main --strategy=ours --allow-unrelated-histories -m "release: $(version)" \
		&& git tag -a v$(version) -m "release $(version)" \
		&& git push -u --tags --set-upstream origin main

# Format Solidity files
format:
	forge fmt

# Lint TypeScript/JavaScript files
lint:
	yarn biome check ./scripts ./lib
