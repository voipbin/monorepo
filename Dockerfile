FROM node:14.19.3-bullseye-slim

LABEL maintainer="Sungtae Kim <pchero21@gmail.com>"

WORKDIR /app

COPY . .

RUN npm install
RUN npm install -g serve
RUN npm run build

