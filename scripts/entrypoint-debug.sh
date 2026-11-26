#!/bin/bash
set -e

# Setup tools if not already installed
if [ ! -f /usr/bin/docker-enter ]; then
    cp /home/tantellam/GolandProjects/beast/scripts/docker-enter /usr/bin/ 2>/dev/null || true
    cp /home/tantellam/GolandProjects/beast/scripts/docker_enter /usr/bin/ 2>/dev/null || true
    chmod u+s /usr/bin/docker_enter 2>/dev/null || true
fi

if [ ! -f /usr/bin/importenv ]; then
    gcc -o /usr/bin/importenv /home/tantellam/GolandProjects/beast/scripts/importenv.c 2>/dev/null || true
fi

# Start forwarder to bridge localhost db connections to host
/usr/local/bin/forwarder &

echo ">>> Building Beast with debug symbols (-gcflags 'all=-N -l')..."
cd /home/tantellam/GolandProjects/beast
go build -gcflags="all=-N -l" -o /tmp/beast-debug ./cmd/beast

echo ">>> Starting Delve headless debugger on :40000..."
echo ">>> Ready for GoLand Remote Debug connection at localhost:40000"
exec dlv exec /tmp/beast-debug --listen=:40000 --headless --api-version=2 --accept-multiclient --continue --only-same-user=false --check-go-version=false -- run -v
