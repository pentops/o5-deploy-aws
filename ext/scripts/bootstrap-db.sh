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

secretArn=$(echo $dbdata | jq -r '.MasterUserSecret.SecretArn')
dbArn=$(echo $dbdata | jq -r '.DBClusterArn')

echo "DB Secret: ${secretArn}"
echo "DB ARN: ${dbArn}"
echo "adminRole: ${adminRole}"


function run() {
  aws rds-data execute-statement \
    --resource-arn "${dbArn}" \
    --secret-arn "${secretArn}" \
    --sql "$1"
}

run "CREATE ROLE ${adminRole} WITH LOGIN CREATEDB CREATEROLE"
run "GRANT rds_iam TO ${adminRole} WITH ADMIN TRUE"
run "CREATE DATABASE ${adminRole}"
run "GRANT CONNECT ON DATABASE ${adminRole} TO ${adminRole}"

# This script doesn't grant rds_iam to the master user
# It is unclear to me if this is the optimal approach.
# - Any role with rds_iam can *only* log in using IAM.
# - Granting to master makes the username and password invalid which seems undesirable.
# - Granting {adminRole} to the master user makes master inherit rds_iam,
#   - even with INHERIT FALSE, which is surprising (at least to me)
# - The master user can still "SET ROLE {adminRole}"
#   - probably because of the rds_superuser role.
#
#

