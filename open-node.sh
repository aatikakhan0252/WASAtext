#!/bin/bash

# start a new (temporary) container using node:20 image
docker run --rm -it -v "$(pwd):/app" -w /app node:20 /bin/bash
