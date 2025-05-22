#!/bin/bash
set -e

DB_USER="costmetrics"
DB_NAME="costmetrics"
DB_HOST="localhost"
DB_PORT="5432"

# Drop database
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -c "DROP DATABASE IF EXISTS $DB_NAME;"

# Create database
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -c "CREATE DATABASE $DB_NAME;"

# Apply migrations
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f internal/db/migrations/0001_init.up.sql
