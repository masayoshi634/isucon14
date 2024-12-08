#!/usr/bin/env bash

set -eux
cd $(dirname $0)

if [ "${ENV:-}" == "local-dev" ]; then
  exit 0
fi

if test -f /home/isucon/env.sh; then
	. /home/isucon/env.sh
fi

ISUCON_DB_HOST=192.168.0.13
ISUCON_DB_PORT=5432
ISUCON_DB_USER=isucon
ISUCON_DB_PASSWORD=isucon
ISUCON_DB_NAME=isuride

psql -U isucon -h 192.168.0.13 -d isuride -f /home/isucon/webapp/sql/pg/1-schema.sql
psql -U isucon -h 192.168.0.13 -d isuride -f /home/isucon/webapp/sql/pg/2-master-data.sql
psql -U isucon -h 192.168.0.13 -d isuride -f /home/isucon/webapp/sql/pg/3-initial-data.sql
psql -U isucon -h 192.168.0.13 -d isuride -f /home/isucon/webapp/sql/pg/4-migrate.sql

psql -U isucon -h 192.168.0.11 -d isuride -f /home/isucon/webapp/sql/pg/1-schema.sql
psql -U isucon -h 192.168.0.11 -d isuride -f /home/isucon/webapp/sql/pg/2-master-data.sql
psql -U isucon -h 192.168.0.11 -d isuride -f /home/isucon/webapp/sql/pg/3-initial-data.sql
psql -U isucon -h 192.168.0.11 -d isuride -f /home/isucon/webapp/sql/pg/4-migrate.sql

