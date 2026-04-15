#!/bin/bash

for i in 5 10 3 8 15 2 12; do
  curl -s localhost:8080/errors/$i > /dev/null
  echo "Generated $i errors"
  sleep 3
done