# common-handler
common-handler for bin namespace.

# Update all project
```
$ ls -d */ | xargs -I {} bash -c "cd '{}' && go get -u ./... && go mod vendor && go generate ./... && go test ./..."
```
