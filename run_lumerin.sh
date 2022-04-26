#!/bin/sh

echo "Building binary..."
cd cmd && go build -o $GOPATH/bin/lumerin && cd ..

echo "Executing..."
lumerin \
	--buyer="false" \
	--network="custom" \
	--mnemonic="" \
	--ethnodeaddress="ws://127.0.0.1:7545" \
	--claimfunds="false" \
	--accountindex="0" \
	--timethreshold="10" \
	--listenip="127.0.0.1" \
	--listenport="3334" \
	--configfile="./ropstenconfig.json" \
	--configdownload="" \
	--logfile="/tmp/lumerin1.log" \
	--loglevel="4" \
	--schedulepassthrough="true" \
	--defaultpooladdr="stratum+tcp://127.0.0.1:33334/" \
	--disableconnection="false" \
	--disablecontract="false" \
	--disableschedule="false" \
	--disableapi="false"