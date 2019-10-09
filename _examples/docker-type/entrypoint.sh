#!/bin/bash

socat tcp-l:10005,fork,reuseaddr exec:./script.sh
