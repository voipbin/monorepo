FROM node:20-slim

LABEL maintainer="Sungtae Kim <pchero21@gmail.com>"

WORKDIR /app

COPY . .

RUN export NODE_OPTIONS=--max_old_space_size=2048

RUN npm update -g
RUN npm install

RUN apt update && apt install -y curl
