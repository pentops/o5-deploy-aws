#!/bin/bash -ue -o pipefail
cluster=$1
dbgroup=$2

echo $cluster | grep -q '^[a-z]\+$' || (echo "Invalid identifier" && exit 1)
echo $dbgroup | grep -q '^[a-z]\+$' || (echo "Invalid identifier" && exit 1)
identifier="${cluster}-${dbgroup}"
adminRole="${cluster}_o5admin"
echo "Identifier: $identifier"
echo "Admin Role: $adminRole"
dbdata=$(aws rds describe-db-clusters --db-cluster-identifier="${identifier}" | jq -r '.DBClusters[0]')

endpoint=$(echo $dbdata | jq -r '.Endpoint')
secret=$(echo $dbdata | jq -r '.MasterUserSecret.SecretArn')
username=$(echo $dbdata | jq -r '.MasterUsername')
database=$(echo $dbdata | jq -r '.DatabaseName')

echo "DB Endpoint: ${endpoint}"
echo "DB Secret: ${secret}"
echo "DB Database: ${database}"
echo "DB Username: ${username}"
echo "adminRole: ${adminRole}"

echo "Fetching password..."

password=$(aws secretsmanager get-secret-value --secret-id $secret | jq -r '.SecretString' | jq -r '.password')

echo "Connecting to db..."

dsn="host=$endpoint port=5432 dbname=$database user=$username password=$password"
psql "$dsn"  <<EOF
CREATE ROLE ${adminRole} WITH LOGIN CREATEDB CREATEROLE;
GRANT rds_iam TO ${adminRole};
SET ROLE ${adminRole};
CREATE DATABASE ${adminRole};
EOF

# This script doesn't grant rds_iam to the master user
# It is unclear to me if this is the optimal approach.
# - Any role with rds_iam can *only* log in using IAM.
# - Granting to master makes the username and password invalid which seems undesirable.
# - Granting {adminRole} to the master user makes master inherit rds_iam,
#   - even with INHERIT FALSE, which is surprising (at least to me)
# - The master user can still "SET ROLE {adminRole}"
#   - probably because of the rds_superuser role.
