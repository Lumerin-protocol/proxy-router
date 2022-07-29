#!/bin/bash

ulimit -n 400

while [ 1 = 1 ]
do
echo "Building binary..."
cd cmd && go build -o lumerin && cd ..
echo "Executing..."
cmd/lumerin --configfile="base_alex_config.test" 2>&1 | tee /tmp/lumerin1-`date +"%y:%m:%d-%H:%M"`
done 
