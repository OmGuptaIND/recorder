#!/usr/bin/env bash
set -euxo pipefail

rm -rf /var/run/pulse /var/lib/pulse /root/.config/pulse

pulseaudio -D --verbose --exit-idle-time=-1 --system --disallow-exit

pactl load-module module-null-sink sink_name="grab" sink_properties=device.description="monitorOUT"

exec tail -f /dev/null