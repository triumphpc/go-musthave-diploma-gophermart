#!/bin/bash
cd ../migrations

DBSTRING="host=$DBHOST user=$DBUSER password=$DBPASSWORD dbname=$DBNAME sslmode=$DBSSL"
goose postgres "$DBSTRING" "$@"
