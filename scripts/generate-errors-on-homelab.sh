#!/bin/bash

for i in 5 10 3 8 15 2 12; do
  curl -sk https://go-logging-example-app.homelab.dobson.dev/errors/$i > /dev/null
  echo "Generated $i errors"
  sleep 30
done