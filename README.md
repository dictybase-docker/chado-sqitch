# Chado-Sqitch
This is a source repository for [docker](http://docker.io) image to run
[Chado-sqitch](http://dictybase.github.io/Chado-Sqitch/) software.

## Usage
By default, running this container will try to install chado schema into
a postgres database. To work with the database, it needs another linked container
and bunch of environmental variables.
### Environmental variables
`$CHADO_DB`, `$CHADO_PASS` and `$CHADO_USER` for the database credentials. For
kubernetes, it can be mounted through `/secrets` through three files,
`chadodb`, `chadopass` and `chadouser`.

For connecting to a postgres database, it needs two variables,
`POSTGRES_SERVICE_HOST` and `POSTGRES_SERVICE_PORT`. It can be done in
kubernetes by creating a service named `postgres` or as a standalone container,
linked to a docker container named `postgres_srv_service`.

