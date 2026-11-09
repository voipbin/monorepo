# bin-dbscheme-manager

Database schema management of the **Asterisk** database for **VoIPBin**.

-   **asterisk_config** -- Manages Asterisk-related database
    schema (Alembic).

The application tables of the services are not managed here: each service
owns its tables and applies its own `migrations/` directory at startup
(see `docs/workflows/service-owned-migrations.md`).

When this project runs via **GitLab CI/CD** or other CI/CD, the associated pod and job
remain until the next pipeline execution, at which point they are
automatically removed.

## Database Setup

Before using **Alembic**, you must create the required databases.

Run the following SQL commands:

```sql
CREATE DATABASE asterisk CHARACTER SET utf8 COLLATE utf8_general_ci;
```

## Initial Setup

The `asterisk_config` directory contains an **alembic.ini.sample** file.
Before running Alembic commands, you **must** copy it to `alembic.ini` and
update the database connection settings.

``` sh
cd asterisk_config
cp alembic.ini.sample alembic.ini
```

Then, edit `alembic.ini` to set the correct database connection
details.


## Adding a Database Change

To create a new migration for **asterisk_config**:

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
alembic -c alembic.ini revision -m "ps_endpoints add column dtls_auto_gen"
alembic -c alembic.ini revision -m "ps_example create table"
```


## Applying Database Migrations

Ensure you are connected to the **VPN** before running migrations.

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

Run these commands inside `asterisk_config`.


## Checking Migration Status

To check the current applied migration:

``` sh
alembic current --verbose
```

Example output:

``` text
INFO  [alembic.runtime.migration] Context impl MySQLImpl.
INFO  [alembic.runtime.migration] Will assume non-transactional DDL.
Current revision(s) for mysql://<db-user>:<db-password>@<db-host>/asterisk:
Rev: <revision-id>
Parent: <parent-revision-id>
Path: <repo-root>/bin-dbscheme-manager/asterisk_config/config/versions/<revision-id>_<title>.py

    <title>

    Revision ID: <revision-id>
    Revises: <parent-revision-id>
    Create Date: <date>
```

Run this inside `asterisk_config` to check the status.

## Viewing Migration History

To list all migrations:

``` sh
alembic history --verbose
```

Example output:

``` text
Rev: <revision-id> (head)
Parent: <parent-revision-id>
Path: <repo-root>/bin-dbscheme-manager/asterisk_config/config/versions/<revision-id>_<title>.py

    <title>

    Revision ID: <revision-id>
    Revises: <parent-revision-id>
    Create Date: <date>
...
```

Run this inside `asterisk_config` to check the migration history.

## Database Details

**Asterisk Configuration (asterisk_config)** 
- Manages **Asterisk-related** database schema. 
- Uses a **fixed Alembic table name**: `alembic_version_config → alembic_version`

These tables are managed by `asterisk_config/` and owned by the Asterisk PBX configuration subsystem. They are written by Kamailio/Asterisk at runtime, not by Go services.

| table_name | owning_service | notes |
|---|---|---|
| ps_endpoints | asterisk-config | PJSIP endpoint definitions |
| ps_auths | asterisk-config | PJSIP authentication records |
| ps_aors | asterisk-config | PJSIP address-of-record bindings |
| ps_contacts | asterisk-config | PJSIP contact registrations |
| ps_domain_aliases | asterisk-config | PJSIP domain alias mappings |
| ps_globals | asterisk-config | PJSIP global settings |
| ps_identify | asterisk-config | PJSIP endpoint identification rules |
| ps_transports | asterisk-config | PJSIP transport configurations |

> The Asterisk schema is sourced from `asterisk/contrib/ast-db-manage`. When upgrading Asterisk, copy the new version files from the Asterisk repo into `asterisk_config/config/versions/`.

## Updating Asterisk Database Schema

To fetch the latest Asterisk DB schema:

``` sh
cd <asterisk-source>
git pull
cp <asterisk-source>/contrib/ast-db-manage/config/versions/* <repo-root>/bin-dbscheme-manager/asterisk_config/config/versions
```
<!-- Updated dependencies: 2026-02-20 -->
