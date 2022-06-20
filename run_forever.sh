#!/bin/bash
while [ 1 = 1 ]
do
echo "Building binary..."
cd cmd && go build -o $GOPATH/bin/lumerin && cd ..
echo "Executing..."
lumerin --configfile="./base_alex_config.test"
done | tee /tmp/lumerin1.log
