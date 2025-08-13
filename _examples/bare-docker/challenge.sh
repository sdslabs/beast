#!/bin/bash

echo -ne "Hello there, I am Bashy, the shell\nWhat can I do for you\n\nType 'e' to exit\n"
while true; do
	read command
	if [[ $command == 'e' ]]; then
		break
	fi
	if [[ $command =~ ^[/?b]*$ ]]; then 
		eval $command > /dev/null
	else
		echo "Command Error!!!!"
	fi
done
