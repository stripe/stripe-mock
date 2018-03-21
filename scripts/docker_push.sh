#!/bin/bash

# Pushes a built Docker image to Docker Hub.
#
# Cargo culted from the Travis documentation:
#
#     https://docs.travis-ci.com/user/docker/

docker login -u "$DOCKER_USERNAME" -p "$DOCKER_PASSWORD";
docker push "$DOCKER_REPO"
