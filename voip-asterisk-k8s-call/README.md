# asterisk-k8s-call
Asterisk kubernetes project for call.

All of call request will reach to this asterisk farm.

# Components
```
asterisk-proxy
asterisk-docker
```

# Buckets
Media(Recording files): /mnt/media
Temp: /mnt/temp

# Test
```
$ docker build -t test:0.4 ./
$ docker run --name asterisk test:0.5
```
