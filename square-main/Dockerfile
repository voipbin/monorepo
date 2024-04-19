FROM node:14.19.3-bullseye-slim

LABEL maintainer="Sungtae Kim <pchero21@gmail.com>"

WORKDIR /app

COPY . .

RUN cd square-main && npm install && npm install -g serve && npm run build

