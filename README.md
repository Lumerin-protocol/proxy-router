# Hashrouter

Controlling a miner through a smart contract

## How to release

Current working repository is in Gitlab (https://gitlab.com/TitanInd/proxy/proxy-router). Its main branch is being mirrored to Github (https://github.com/Lumerin-protocol/hash-router). The releasing job is ran by Gitlab. The releases are stored in Github.

1. Make sure GITHUB_TOKEN is set as Gitlab env variable (https://goreleaser.com/scm/github/)
1. Create a new tag `git tag 0.0.x`
1. Push it to remote `git push --tags`
1. The CI job that runs on tag push should release the artifacts

## How to run integration test

1. Run `make build`. Make sure hashrouter executable is located in root
1. Run `go test -tags wireinject -run ^TestHashrateDelivery$ gitlab.com/TitanInd/hashrouter/test/fakeminerpool -v -count=1 -timeout=120m`
