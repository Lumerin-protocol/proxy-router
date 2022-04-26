#!/bin/sh

solc --abi --overwrite CloneFactory.sol -o ../build
solc --bin --overwrite CloneFactory.sol -o ../build