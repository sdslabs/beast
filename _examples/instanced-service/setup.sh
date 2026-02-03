#!/bin/bash
set -e

echo "[*] Setting up instanced-service challenge..."

# Compile the vulnerable binary
gcc -o pwn pwn_me.c -fno-stack-protector -no-pie -z execstack

# Make it executable
chmod +x pwn

# Create flag file
echo "FLAG{1nst4nc3d_ch4ll3ng3_w0rks!}" > flag.txt
chmod 444 flag.txt

echo "[*] Setup complete!"
