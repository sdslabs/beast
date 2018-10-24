#!/bin/bash

/etc/init.d/nginx start
exec tail -f /var/log/nginx/*
