#!/bin/sh

echo "Building binary..."
cd cmd \
	&& go build -o $GOPATH/bin/lumerin \
	&& cd .. \
	&& echo "Executing..." \
	&& lumerin \
		--buyer="false" \
		--network="custom" \
		--ethnodeaddress="ws://127.0.0.1:7545" \
		--claimfunds="false" \
		--timethreshold="10" \
		--listenip="127.0.0.1" \
		--listenport="3333" \
		--configfile="./ropstenconfig.json" \
		--configdownload="" \
		--logfile="/tmp/lumerin1.log" \
		--loglevel="4" \
		--disablecontract="false" \
		--disableschedule="false" \
		--disableapi="false"