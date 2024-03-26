O5 Deployer for AWS
===================


## IAM Roles

### O5-Deployer-Assume-Role

The environment config file specifies a role which is created when the platform
infrastructure is created (e.g. by terraform).

This role is assumed by the deployer when deploying the to the environment, it
should be able to run cloudformation, set up SNS topics, and read the
environment configs.

TODO: Explicit details on the permissions this role requires.

The role which is running the deployer, either locally or running on
infrastructure, needs to be granted permission to assume this role.

### O5-Deployer-Grant-Roles

In the environment config file, this is an array of roles which the O5 deployer,
when itself deployed to that environment, should be given permission to assume.

In a simple setup, this should have only one entry: The O5-Deployer-Assume-Role
value above.

However, a deployer running on infrastructure may be responsible for deployments
to multiple environments.

When the Application config file sets the boolean 'grantMetaDeployPermissions',
that indicates to the deployer that it is deploying ... itself, and so it is is
granted permission to assume the roles in this array of the config file.

### ECS Task Execution Role

Another role which is pre-defined from the platform infrastructure. This is the
role which the AWS ECS service uses to manage the running tasks. It is not used
at runtime


### Runtime Role

And finally, this role is managed as part of the cloudformation stack for the
Application, it is the role which the runtime service is granted. It is given
permissions based on resources defined in the application config (e.g. when a
bucket is referenced, this role gets permission to use it).

Unfortunately, AWS uses the same role for all tasks in a service, so while the
application itself should rarely (if ever) directly access AWS resources, it is
nonetheless given full permission to do so.

