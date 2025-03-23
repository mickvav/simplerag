## Simple RAG

This is very simple attempt to create RAG-aware CLI interface to locally-running LLM in golang.

Requirements:

* docker (to run postgresql + pgvector)
* golang
* llama-cpp
* downloaded model

Run-time requirements:

* environment variable PG_PASSWORD_FILE with the name of the file on local filesystem, that holds password for postgresql user.

* docker container running (as executed by install.sh).

* llama-cpp running locally with _some_ model.

