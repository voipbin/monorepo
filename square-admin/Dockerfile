FROM node:20-slim

LABEL maintainer="Sungtae Kim <pchero21@gmail.com>"

WORKDIR /app

COPY . .

RUN apt update && apt install -y curl

RUN export NODE_OPTIONS=--max_old_space_size=2048
RUN cd square-admin && npm update -g && npm install
