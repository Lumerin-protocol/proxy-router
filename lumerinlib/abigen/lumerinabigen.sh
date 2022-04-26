#!/bin/sh

abigen --bin=./build/CloneFactory.bin --abi=./build/CloneFactory.abi --pkg=clonefactory --out=../clonefactory/clonefactory.go
abigen --bin=./build/Implementation.bin --abi=./build/Implementation.abi --pkg=implementation --out=../implementation/implementation.go
abigen --bin=./build/Lumerin.bin --abi=./build/Lumerin.abi --pkg=lumerintoken --out=../lumerintoken/lumerintoken.go