FROM node:14.19.3-bullseye-slim

LABEL maintainer="Sungtae Kim <pchero21@gmail.com>"

WORKDIR /app

COPY . .

RUN apt update && apt install -y procps
RUN npm install
RUN npm install -g gulp-cli
