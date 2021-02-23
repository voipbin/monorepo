# dbscheme-bin-manager

Database scheme manage project for bin-manager

After run this project using the gitlab-ci/cd, the used pod and job will be remained. But it will be destroyed once this poject runs in pipelie again.

# Run
Need a connection to the VPN.

```
$ cd bin-manager
$ alembic -c alembic.ini upgrade head
```
