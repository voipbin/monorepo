# dbscheme-bin-manager

Database scheme manage project for bin-manager

After run this project using the gitlab-ci/cd, the used pod and job will be remained. But it will be destroyed once this poject runs in pipelie again.

# Status check
```
$ alembic current --verbose                                                                   7s
INFO  [alembic.runtime.migration] Context impl MySQLImpl.
INFO  [alembic.runtime.migration] Will assume non-transactional DDL.
Current revision(s) for mysql://bin-manager:XXXXX@10.126.80.5/bin_manager:
Rev: c939ba877f8f
Parent: 6a7807e971e1
Path: /home/pchero/gitlab/voipbin/bin-manager/dbscheme-bin-manager/bin-manager/main/versions/c939ba877f8f_add_table_extensions.py

    add table extensions

    Revision ID: c939ba877f8f
    Revises: 6a7807e971e1
    Create Date: 2021-02-23 14:17:17.716292
```

# history check
```
alembic history --verbose                                                                   7s
Rev: 5d2aab77fd9d (head)
Parent: c939ba877f8f
Path: /home/pchero/gitlab/voipbin/bin-manager/dbscheme-bin-manager/bin-manager/main/versions/5d2aab77fd9d_add_provider_info.py

    add provider info

    Revision ID: 5d2aab77fd9d
    Revises: c939ba877f8f
    Create Date: 2021-03-01 01:53:00.395262
...
```

# Run
Need a connection to the VPN.

```
$ cd bin-manager
$ alembic -c alembic.ini upgrade head
```

# Rollback

```
$ cd bin-manager
```

Run one of the below.
```
$ alembic -c alembic.ini downgrade -1
$ alembic -c alembic.ini downgrade ae1027a6acf
```


# Make alembic change

```
$ cd bin-manager
$ alembic -c alembic.ini revision -m "<your change title>"
```
