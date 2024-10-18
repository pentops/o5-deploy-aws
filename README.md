O5 Deployer for AWS
===================


# Configuration

## Config Types

### Clusters and Environments

Configured with o5.environment.v1.Cluster and o5.environment.v1.Environment.

An environment is a namespace with shared configuration parameters and
resources. Each Environment belongs to a Cluster, which further groups the
resources and parameters.

The creation of the resources is not defined in this codebase.

### Applications

Configured with o5.application.v1.Application.

An Application is exposed as a Docker Container, generally represents a single
codebase and a single micro-service.

### Stack

A Stack combines an Environment and an Application - it represents a version of
a codebase running within an environment.

## Resources

The Application config file defines resources which can be used by the
application.

The resources are defined at the top level of the application config, then
referenced using environment variables of the runtime containers.

(The env vars are the only way an application knows about the resources)

### Blobstore

S3 buckets which are either owned by the stack or references to other stacks.

### Secret

### Database

Databases must be owned by the stack - applications must not share data.

Currently only PostgreSQL databases are supported, running on RDS, either using
IAM Auth for an Aurora cluster, or a secret containing a username and password.

Databases run on 'Server Groups', which are RDS hosts / clusters. The individual
named databases are created within the server group.

## Subscriptions and Targets

