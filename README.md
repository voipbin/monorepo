# monorepo
Welcome to voipbin project.

# Links
* http://voipbin.net/ : Voipbin project page
* https://api.voipbin.net/docs/ : Voipbin API documentation
* https://admin.voipbin.net/ : Console admin page

# Namespace
This voipbin monorepo has many sub repositories with multiple namespaces.

* bin-*: bin-manager projects.
* square-*: square-manager projects.
* voip-*: voip based projects.
* infra-* infrastructure projects.

# Merge existing projects
The monorepo concist of many sub projects. Most of projects were merged from the existing projects using the following command.
```
$ git subtree add -P <destination repository directory> <source repository> <source repository branch name>
```

## Exmaple
```
$ git subtree add -P bin-agent-manager ../../../gitlab/voipbin/bin-manager/agent-manager master
```