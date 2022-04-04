#!/bin/bash

# Detect the platform (similar to $OSTYPE)
OS="`uname`"
case $OS in
  'Linux')
    OS='Linux'
    sudo apt install shellcheck
    ;;

  'Arch')
    OS='Arch'
    pacman -S shellcheck
    ;;

  'WindowsNT')
    OS='Windows'
    C:\> choco install shellcheck
    ;;

  'Darwin') 
    OS='Mac'
    brew install shellcheck
    ;;

   *)
    echo "OS not supported, please install shellcheck manually"
    ;; 
esac




