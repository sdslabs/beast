#!/bin/bash

PWD=$(pwd)

ERROR="\e[31;1mERROR\e[0m"
SUCCESS="\e[32;1mSUCCESS\e[0m"
INFO="\e[34;1mINFO\e[0m"

# Test challenge simple
CHALLENGE="simple"
## Check if port is taken
echo -e "$INFO : $CHALLENGE : Check port availability"
(echo -e 1 > /dev/tcp/127.0.0.1/10001) 2> /dev/null
if [[ $? -eq 0 ]]; then
	echo -e "$ERROR : $CHALLENGE : port 10001 is not free, cannot test challenge"
	exit 1
fi

## Deploy challenge
echo -e "$INFO : $CHALLENGE : Start deploy"
beast -v challenge deploy --local-directory $PWD/_examples/$CHALLENGE
if [[ $? -ne 0 ]]; then
	echo -e "$ERROR: $CHALLENGE : There was an error in deployment of challenge"
	exit 1
fi

## Test deployment
echo -e "$INFO : $CHALLENGE : Test deployment"
(echo -e 1 > /dev/tcp/127.0.0.1/10001) 2> /dev/null
if [[ $? -eq 0 ]]; then
	echo -e "$SUCCESS: $CHALLENGE: Deployed successfully"
else
	echo -e "$ERROR: $CHALLENGE : There was an error in deployment of challenge"
	exit 1	
fi

## Purge
echo -e "$INFO : $CHALLENGE : Purge challenge"
beast -v challenge purge $CHALLENGE -d
if [[ $? -ne 0 ]]; then
	echo -e "$ERROR : $CHALLENGE : Error while purging"
	exit 1	
fi


#Test challenge static-chall
CHALLENGE="static-chall"
## Deploy challenge
echo -e "$INFO : $CHALLENGE : Start deploy"
beast -v challenge deploy --local-directory $PWD/_examples/$CHALLENGE
if [[ $? -ne 0 ]]; then
	echo -e "$ERROR : $CHALLENGE : There was an error in deployment of challenge"
	exit 1
fi

## Test deployment
echo -e "$INFO : $CHALLENGE : Test deployment"
response_code=$(curl --write-out %{http_code} --silent --output /dev/null http://static.beast.sdslabs.co/static/$CHALLENGE/index.html)
if [[ $response_code -eq 200 ]]; then
	echo -e "$SUCCESS : $CHALLENGE : Deployed successfully"
else
	echo -e "$ERROR : $CHALLENGE : There was an error in deployment of challenge"
	exit 1
fi

## Purge
echo -e "$INFO : $CHALLENGE : Purge challenge"
beast -v challenge purge $CHALLENGE -d
if [[ $? -ne 0 ]]; then
	echo -e "$ERROR : $CHALLENGE : Error while purging"
	exit 1	
fi


# Test challenge web-php
CHALLENGE="web-php"
## Check if port is taken
echo -e "$INFO : $CHALLENGE : Check port availability"
(echo -e 1 > /dev/tcp/127.0.0.1/10002) 2> /dev/null
if [[ $? -eq 0 ]]; then
	echo -e "$ERROR : $CHALLENGE : port 10002 is not free, cannot test challenge"
	exit 1
fi

## Deploy challenge
echo -e "$INFO : $CHALLENGE : Start deploy"
beast -v challenge deploy --local-directory $PWD/_examples/$CHALLENGE
if [[ $? -ne 0 ]]; then
	echo -e "$ERROR : $CHALLENGE : There was an error in deployment of challenge"
	exit 1
fi

## Test deployment
echo -e "$INFO : $CHALLENGE : Test deployment"
response_code=$(curl --write-out %{http_code} --silent --output /dev/null http://localhost:10002/index.php)
if [[ $response_code -eq 200 ]]; then
	echo -e "$SUCCESS : $CHALLENGE : Deployed successfully"
else
	echo -e "$ERROR : $CHALLENGE : There was an error in deployment of challenge"
	exit 1	
fi

## Purge
echo -e "$INFO : $CHALLENGE : Purge challenge"
beast -v challenge purge $CHALLENGE -d
if [[ $? -ne 0 ]]; then
	echo -e "$ERROR : $CHALLENGE : Error while purging"
	exit 1	
fi




# Test challenge web-php-mysql
CHALLENGE="web-php-mysql"
## Check if port is taken
echo -e "$INFO : $CHALLENGE : Check port availability"
(echo -e 1 > /dev/tcp/127.0.0.1/10004) 2> /dev/null
if [[ $? -eq 0 ]]; then
	echo -e "$ERROR : $CHALLENGE : port 10004 is not free, cannot test challenge"
	exit 1
fi

## Deploy challenge
echo -e "$INFO : $CHALLENGE : Start deploy"
beast -v challenge deploy --local-directory $PWD/_examples/$CHALLENGE
if [[ $? -ne 0 ]]; then
	echo -e "$ERROR : $CHALLENGE : There was an error in deployment of challenge"
	exit 1
fi

## Test deployment
echo -e "$INFO : $CHALLENGE : Test deployment"
response_code=$(curl --write-out %{http_code} --silent --output /dev/null http://localhost:10004/index.php)
if [[ $response_code -eq 200 ]]; then
	echo -e "$SUCCESS : $CHALLENGE : Deployed successfully"
else
	echo -e "$ERROR : $CHALLENGE : There was an error in deployment of challenge"
	exit 1	
fi

## Purge
echo -e "$INFO : $CHALLENGE : Purge challenge"
beast -v challenge purge $CHALLENGE -d
if [[ $? -ne 0 ]]; then
	echo -e "$ERROR : $CHALLENGE : Error while purging"
	exit 1	
fi


# Test challenge xinetd-service
CHALLENGE="xinetd-service"
## Check if port is taken
echo -e "$INFO : $CHALLENGE : Check port availability"
(echo -e 1 > /dev/tcp/127.0.0.1/10003) 2> /dev/null
if [[ $? -eq 0 ]]; then
	echo -e "$ERROR : $CHALLENGE : port 10003 is not free, cannot test challenge"
	exit 1
fi

## Deploy challenge
echo -e "$INFO : $CHALLENGE : Start deploy"
beast -v challenge deploy --local-directory $PWD/_examples/$CHALLENGE
if [[ $? -ne 0 ]]; then
	echo -e "$ERROR : $CHALLENGE : There was an error in deployment of challenge"
	exit 1
fi

## Test deployment
echo -e "$INFO : $CHALLENGE : Test deployment"
(echo -e 1 > /dev/tcp/127.0.0.1/10003) 2> /dev/null
if [[ $? -eq 0 ]]; then
	echo -e "$SUCCESS : $CHALLENGE : Deployed successfully"
else
	echo -e "$ERROR : $CHALLENGE : There was an error in deployment of challenge"
	exit 1	
fi

## Purge
echo -e "$INFO : $CHALLENGE : Purge challenge"
beast -v challenge purge $CHALLENGE -d
if [[ $? -ne 0 ]]; then
	echo -e "$ERROR : $CHALLENGE : Error while purging"
	exit 1	
fi