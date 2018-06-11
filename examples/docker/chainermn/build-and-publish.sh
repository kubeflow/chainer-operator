#! /usr/bin/env bash

IMAGE_REPO=$1 # your_org/your_image
IMAGE_TAG=${2:-latest}

docker build -t $IMAGE_REPO:$IMAGE_TAG .
docker push $IMAGE_REPO:$IMAGE_TAG
