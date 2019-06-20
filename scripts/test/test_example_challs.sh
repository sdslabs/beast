#!/bin/bash

PWD=$(pwd)

ERROR="\e[31;1mERROR\e[0m"
SUCCESS="\e[32;1mSUCCESS\e[0m"
INFO="\e[34;1mINFO\e[0m"

checkPortAvailable() {
	local port=$1
	echo -e "$INFO : $CHALLENGE : Check port availability"
	(echo -e 1 > /dev/tcp/127.0.0.1/$port) 2> /dev/null
	if [[ $? -eq 0 ]]; then
		echo -e "$ERROR : $CHALLENGE : port $port is not free, cannot test challenge"
		exit 1
	fi
}

deployChallenge() {
	local challdir=$1
	echo -e "$INFO : $CHALLENGE : Start deploy"
	beast -v challenge deploy --local-directory $challdir
	if [[ $? -ne 0 ]]; then
		echo -e "$ERROR: $CHALLENGE : There was an error in deployment of challenge"
		exit 1
	fi
}

checkPortReachable() {
	(echo -e 1 > /dev/tcp/127.0.0.1/$PORT) 2> /dev/null
}

purge() {
	echo -e "$INFO : $CHALLENGE : Purge challenge"
	beast -v challenge purge $CHALLENGE -d
	if [[ $? -ne 0 ]]; then
		echo -e "$ERROR : $CHALLENGE : Error while purging"
		exit 1	
	fi
}

doHTTPProbe() {
	local url=$1
	curl --write-out %{http_code} --silent --output /dev/null $url
}

# Test challenge simple
CHALLENGE="simple"
PORT=10001
## Check if port is taken
checkPortAvailable $PORT
## Deploy challenge
deployChallenge $PWD/_examples/$CHALLENGE
## Test deployment
echo -e "$INFO : $CHALLENGE : Test deployment"
checkPortReachable
if [[ $? -eq 0 ]]; then
	echo -e "$SUCCESS: $CHALLENGE: Deployed successfully"
else
	echo -e "$ERROR: $CHALLENGE : There was an error in deployment of challenge"
	exit 1	
fi
## Purge
purge

#Test challenge static-chall
CHALLENGE="static-chall"
#Test beast-static container
echo -e "$INFO : $CHALLENGE : Test beast-static container"
docker ps | grep -q 'beast-static'
if [[ $? -ne 0 ]]; then
	echo -e "$ERROR: $CHALLENGE : beast-static container is not running"
	exit 1
fi
## Deploy challenge
deployChallenge $PWD/_examples/$CHALLENGE
## Test deployment
echo -e "$INFO : $CHALLENGE : Test deployment"
response_code=$(doHTTPProbe "http://static.beast.sdslabs.co/static/$CHALLENGE/index.html")
if [[ $response_code -eq 200 ]]; then
	echo -e "$SUCCESS : $CHALLENGE : Deployed successfully"
else
	echo -e "$ERROR : $CHALLENGE : There was an error in deployment of challenge"
	exit 1
fi
## Purge
purge

# Test challenge web-php
CHALLENGE="web-php"
PORT=10002
## Check if port is taken
checkPortAvailable $PORT
## Deploy challenge
deployChallenge $PWD/_examples/$CHALLENGE
## Test deployment
echo -e "$INFO : $CHALLENGE : Test deployment"
response_code=$(doHTTPProbe "http://localhost:$PORT/index.php")
if [[ $response_code -eq 200 ]]; then
	echo -e "$SUCCESS : $CHALLENGE : Deployed successfully"
else
	echo -e "$ERROR : $CHALLENGE : There was an error in deployment of challenge"
	exit 1	
fi
## Purge
purge

# Test challenge web-php-mysql
CHALLENGE="web-php-mysql"
PORT=10004
## Check if port is taken
checkPortAvailable $PORT
## Test beast-mysql container
echo -e "$INFO : $CHALLENGE : Test beast-mysql container"
docker ps | grep -q 'beast-mysql'
if [[ $? -ne 0 ]]; then
	echo -e "$ERROR: $CHALLENGE : beast-mysql container is not running"
	exit 1
fi
## Deploy challenge
deployChallenge $PWD/_examples/$CHALLENGE
## Test deployment
echo -e "$INFO : $CHALLENGE : Test deployment"
response_code=$(doHTTPProbe "http://localhost:$PORT/index.php")
if [[ $response_code -eq 200 ]]; then
	echo -e "$SUCCESS : $CHALLENGE : Deployed successfully"
else
	echo -e "$ERROR : $CHALLENGE : There was an error in deployment of challenge"
	exit 1	
fi
## Purge
purge

# Test challenge xinetd-service
CHALLENGE="xinetd-service"
PORT=10003
## Check if port is taken
checkPortAvailable $PORT
## Deploy challenge
deployChallenge $PWD/_examples/$CHALLENGE
## Test deployment
echo -e "$INFO : $CHALLENGE : Test deployment"
checkPortReachable
if [[ $? -eq 0 ]]; then
	echo -e "$SUCCESS : $CHALLENGE : Deployed successfully"
else
	echo -e "$ERROR : $CHALLENGE : There was an error in deployment of challenge"
	exit 1	
fi
## Purge
purge