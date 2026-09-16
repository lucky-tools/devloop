#!/bin/bash

docker build -t $IMAGE .

image=$(echo $IMAGE)

if [ ! -z "$image" ]; then
  if $PUSH_IMAGE
  then
    docker push $image
  fi
fi
