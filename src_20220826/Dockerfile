FROM node:14.19.3-bullseye-slim

LABEL maintainer="Sungtae Kim <pchero21@gmail.com>"

WORKDIR /app

# COPY . .

RUN apt update && apt install -y git \
&& echo n | npm install -g --silent @angular/cli \
&& npm install

# RUN yarn
# RUN yarn install
# RUN npm install -g serve
# RUN npm run build

