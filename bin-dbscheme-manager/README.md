# dbscheme-bin-manager

Database schema management for **VoIPBin**.

This project is divided into two directories, each managing its own
database resources:

-   **asterisk_config** -- Manages Asterisk-related database
    schema.
-   **bin-manager** -- Manages most of **VoIPBin's** database
    resources.

When this project runs via **GitLab CI/CD** or other CI/CD, the associated pod and job
remain until the next pipeline execution, at which point they are
automatically removed.

## Database Setup

Before using **Alembic**, you must create the required databases.

Run the following SQL commands:

```sql
CREATE DATABASE asterisk CHARACTER SET utf8 COLLATE utf8_general_ci;
CREATE DATABASE voipbin CHARACTER SET utf8 COLLATE utf8_general_ci;
```

## Initial Setup

Each directory contains an **alembic.ini.sample** file. Before
running Alembic commands, you **must** copy it to `alembic.ini` and
update the database connection settings.

For **bin-manager**:

``` sh
cd bin-manager
cp alembic.ini.sample alembic.ini
```

For **asterisk_config**:

``` sh
cd asterisk_config
cp alembic.ini.sample alembic.ini
```

Then, edit `alembic.ini` in each directory to set the correct database
connection details.


## Adding a Database Change

To create a new database migration, navigate to the appropriate
directory.

For **bin-manager**:

``` sh
cd bin-manager
alembic -c alembic.ini revision -m "<your change title>"
```

For **asterisk_config**:

``` sh
cd asterisk_config
alembic -c alembic.ini revision -m "<your change title>"
```

## Naming Convention for Migration Titles

Follow this format for migration messages:

``` text
<table_name>_<action: create/remove/add/update>_<type: column/table>_<items>
```

### Examples:

``` sh
alembic -c alembic.ini revision -m "customers add column email phone_number address"
alembic -c alembic.ini revision -m "registrar_trunks create table"
```


## Applying Database Migrations

Ensure you are connected to the **VPN** before running migrations.

For **bin-manager**:

``` sh
cd bin-manager
alembic -c alembic.ini upgrade head
```

For **asterisk_config**:

``` sh
cd asterisk_config
alembic -c alembic.ini upgrade head
```

## Rolling Back Migrations

To revert the last applied migration:

``` sh
alembic -c alembic.ini downgrade -1
```

To revert to a specific migration (replace `<revision_id>` with the
actual ID):

``` sh
alembic -c alembic.ini downgrade <revision_id>
```

Example:

``` sh
alembic -c alembic.ini downgrade ae1027a6acf
```

Run these commands inside either `bin-manager` or `asterisk_config`,
depending on which schema you need to roll back.


## Checking Migration Status

To check the current applied migration:

``` sh
alembic current --verbose
```

Example output:

``` text
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

Run this inside `bin-manager` or `asterisk_config` to check the status
for each database.

## Viewing Migration History

To list all migrations:

``` sh
alembic history --verbose
```

Example output:

``` text
Rev: 5d2aab77fd9d (head)
Parent: c939ba877f8f
Path: /home/pchero/gitlab/voipbin/bin-manager/dbscheme-bin-manager/bin-manager/main/versions/5d2aab77fd9d_add_provider_info.py

    add provider info

    Revision ID: 5d2aab77fd9d
    Revises: c939ba877f8f
    Create Date: 2021-03-01 01:53:00.395262
...
```

Run this inside `bin-manager` or `asterisk_config` to check migration
history for each database.

## Database Details

**Asterisk Configuration (asterisk_config)** 
- Manages **Asterisk-related** database schema. 
- Uses a **fixed Alembic table name**: `alembic_version_config → alembic_version`

**VoIPBin Database (bin-manager)**
- Manages most of **VoIPBin's** database resources, excluding Asterisk-related data.

## Updating Asterisk Database Schema

To fetch the latest Asterisk DB schema:

``` sh
cd /home/pchero/gittmp/asterisk
git pull
cp /home/pchero/gittmp/asterisk/contrib/ast-db-manage/config/versions/* /home/pchero/gitlab/voipbin/bin-manager/dbscheme-bin-manager/asterisk_config/config/versions
```
<!-- Updated dependencies: 2026-02-20 -->
