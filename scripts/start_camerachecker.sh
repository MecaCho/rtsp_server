#!/bin/bash
/usr/bin/ief-camera-checker \
                --alsologtostderr \
                --mqtt-retry-time=3 \
                --check-camera-interval=60 \
                --check-remote-camera=true \
                --node_id=${NODE_ID}