#!/bin/bash

docker pull pgvector/pgvector:pg17

if [ "$PG_PASSWORD_FILE" = "" ]; then
	echo "set PG_PASSWORD_FILE envinronment variable to point to file with password"
	exit 1
fi
docker run --name v-postgres -d -p 5432:5432 -e POSTGRES_PASSWORD=$(cat $PG_PASSWORD_FILE) pgvector/pgvector:pg17

llama-server -m qwen2.5-coder-1.5b-instruct-q4_k_m.gguf --port 8012 -ngl 80 -fa -dt 0.1 --ubatch-size 2048 --batch-size 2048 --cache-reuse 256 -v