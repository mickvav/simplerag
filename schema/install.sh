#!/bin/bash

docker pull pgvector/pgvector:pg17

if [ "$PG_PASSWORD_FILE" = "" ]; then
	echo "set PG_PASSWORD_FILE envinronment variable to point to file with password"
	exit 1
fi
docker run --name v-postgres -e POSTGRES_PASSWORD=$(cat $PG_PASSWORD_FILE) -d postgres

