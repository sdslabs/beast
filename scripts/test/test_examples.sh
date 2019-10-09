#!/bin/bash

PWD=$(pwd)

ERROR="\e[31;1mERROR\e[0m"
SUCCESS="\e[32;1mSUCCESS\e[0m"
INFO="\e[34;1mINFO\e[0m"

checkPortAvailable() {
	local challenge=$1
	local port=$2
	echo -e "$INFO : $challenge : Check port availability"
	(echo -e 1 > /dev/tcp/127.0.0.1/$port) 2> /dev/null
	if [[ $? -eq 0 ]]; then
		echo -e "$ERROR : $challenge : port $port is not free, cannot test challenge"
		exit 1
	fi
}

deployChallenge() {
	local challenge=$1
	local challdir=$2
	echo -e "$INFO : $challenge : Start deploy"
	beast -v challenge deploy --local-directory $challdir
	if [[ $? -ne 0 ]]; then
		echo -e "$ERROR: $challenge : There was an error in deployment of challenge"
		exit 1
	fi
}

checkPortReachable() {
	local port=$1
	(echo -e 1 > /dev/tcp/127.0.0.1/$port) 2> /dev/null
}

purge() {
	local challenge=$1
	echo -e "$INFO : $challenge : Purge challenge"
	beast -v challenge purge $challenge -d
	if [[ $? -ne 0 ]]; then
		echo -e "$ERROR : $challenge : Error while purging"
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
checkPortAvailable $CHALLENGE $PORT
## Deploy challenge
deployChallenge $CHALLENGE $PWD/_examples/$CHALLENGE
## Test deployment
echo -e "$INFO : $CHALLENGE : Test deployment"
checkPortReachable $PORT
if [[ $? -eq 0 ]]; then
	echo -e "$SUCCESS: $CHALLENGE : Deployed successfully"
else
	echo -e "$ERROR: $CHALLENGE : There was an error in deployment of challenge"
	exit 1	
fi

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
deployChallenge $CHALLENGE $PWD/_examples/$CHALLENGE
## Test deployment
echo -e "$INFO : $CHALLENGE : Test deployment"
response_code=$(doHTTPProbe "http://localhost/static/$CHALLENGE/index.html")
if [[ $response_code -eq 200 ]]; then
	echo -e "$SUCCESS : $CHALLENGE : Deployed successfully"
else
	echo -e "$ERROR : $CHALLENGE : There was an error in deployment of challenge"
	exit 1
fi

# Test challenge web-php
CHALLENGE="web-php"
PORT=10002
## Check if port is taken
checkPortAvailable $CHALLENGE $PORT
## Deploy challenge
deployChallenge $CHALLENGE $PWD/_examples/$CHALLENGE
## Test deployment
echo -e "$INFO : $CHALLENGE : Test deployment"
response_code=$(doHTTPProbe "http://localhost:$PORT/index.php")
if [[ $response_code -eq 200 ]]; then
	echo -e "$SUCCESS : $CHALLENGE : Deployed successfully"
else
	echo -e "$ERROR : $CHALLENGE : There was an error in deployment of challenge"
	exit 1	
fi

# Test challenge web-php-mysql
CHALLENGE="web-php-mysql"
PORT=10004
## Check if port is taken
checkPortAvailable $CHALLENGE $PORT
## Test beast-mysql container
echo -e "$INFO : $CHALLENGE : Test beast-mysql container"
docker ps | grep -q 'beast-mysql'
if [[ $? -ne 0 ]]; then
	echo -e "$ERROR: $CHALLENGE : beast-mysql container is not running"
	exit 1
fi
## Deploy challenge
deployChallenge $CHALLENGE $PWD/_examples/$CHALLENGE
## Test deployment
echo -e "$INFO : $CHALLENGE : Test deployment"
response_code=$(doHTTPProbe "http://localhost:$PORT/index.php")
if [[ $response_code -eq 200 ]]; then
	echo -e "$SUCCESS : $CHALLENGE : Deployed successfully"
else
	echo -e "$ERROR : $CHALLENGE : There was an error in deployment of challenge"
	exit 1	
fi

# Test challenge xinetd-service
CHALLENGE="xinetd-service"
PORT=10003
## Check if port is taken
checkPortAvailable $CHALLENGE $PORT
## Deploy challenge
deployChallenge $CHALLENGE $PWD/_examples/$CHALLENGE
## Test deployment
echo -e "$INFO : $CHALLENGE : Test deployment"
checkPortReachable $PORT
if [[ $? -eq 0 ]]; then
	echo -e "$SUCCESS : $CHALLENGE : Deployed successfully"
else
	echo -e "$ERROR : $CHALLENGE : There was an error in deployment of challenge"
	exit 1	
fi

# Test challenge xinetd-service
CHALLENGE="docker-type"
PORT=10005
## Check if port is taken
checkPortAvailable $CHALLENGE $PORT
## Deploy challenge
deployChallenge $CHALLENGE $PWD/_examples/$CHALLENGE
## Test deployment
echo -e "$INFO : $CHALLENGE : Test deployment"
checkPortReachable $PORT
if [[ $? -eq 0 ]]; then
	echo -e "$SUCCESS : $CHALLENGE : Deployed successfully"
else
	echo -e "$ERROR : $CHALLENGE : There was an error in deployment of challenge"
	exit 1	
fi

## Purge all challenges
# simple
purge simple
# static-chall
purge static-chall
# web-php
purge web-php
# web-php-mysql
purge web-php-mysql
# xinetd-service
purge xinetd-service
