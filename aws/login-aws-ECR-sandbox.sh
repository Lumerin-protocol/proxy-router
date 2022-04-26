#!/bin/bash

PASS=`aws ecr get-login --region us-east-1 --no-include-email --profile sandbox `
echo $PASS
$PASS


