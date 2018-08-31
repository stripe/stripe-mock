#!/bin/bash

# Pushes a built Docker image to Docker Hub.
#
# Cargo culted from the Travis documentation:
#
#     https://docs.travis-ci.com/user/docker/

docker login -u "$DOCKER_USERNAME" -p "$DOCKER_PASSWORD";

# $TRAVIS_TAG contains a tag if this is a build for a tag, otherwise it's
# empty.
#
# Create Docker Hub tags for Git tags and keep one in place for the master
# branch.
if [ ! -z "$TRAVIS_TAG" ]; then
    docker tag "$DOCKER_REPO:${TRAVIS_COMMIT:0:8}" "$DOCKER_REPO:$TRAVIS_TAG"
elif [ "$TRAVIS_BRANCH" == "master"]; then
    docker tag "$DOCKER_REPO:${TRAVIS_COMMIT:0:8}" "$DOCKER_REPO:master"
fi

docker push "$DOCKER_REPO"
