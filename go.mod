module github.com/pentops/o5-deploy-aws

go 1.24.0

toolchain go1.24.1

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.6-20250613105001-9f2d3c737feb.1
	buf.build/go/protoyaml v0.6.0
	github.com/aws/aws-sdk-go-v2 v1.36.5
	github.com/aws/aws-sdk-go-v2/config v1.29.17
	github.com/aws/aws-sdk-go-v2/credentials v1.17.70
	github.com/aws/aws-sdk-go-v2/feature/rds/auth v1.5.13
	github.com/aws/aws-sdk-go-v2/service/cloudformation v1.60.3
	github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs v1.51.0
	github.com/aws/aws-sdk-go-v2/service/ecs v1.57.6
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.45.5
	github.com/aws/aws-sdk-go-v2/service/s3 v1.81.0
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.35.7
	github.com/aws/aws-sdk-go-v2/service/sns v1.34.7
	github.com/aws/aws-sdk-go-v2/service/sts v1.34.0
	github.com/aws/smithy-go v1.22.4
	github.com/awslabs/goformation/v7 v7.14.9
	github.com/bradleyfalzon/ghinstallation v1.1.1
	github.com/bufbuild/protovalidate-go v0.10.1
	github.com/elgris/sqrl v0.0.0-20210727210741-7e0198b30236
	github.com/google/go-cmp v0.7.0
	github.com/google/go-github/v47 v47.1.0
	github.com/google/uuid v1.6.0
	github.com/iancoleman/strcase v0.3.0
	github.com/lib/pq v1.10.9
	github.com/pentops/envconf.go v0.0.0-20241008010024-9864aef6219d
	github.com/pentops/flowtest v0.0.0-20250611222350-b5c7162d9db1
	github.com/pentops/grpc.go v0.0.0-20250604193928-b45524df9c41
	github.com/pentops/j5 v0.0.0-20250627203711-1cb261630b87
	github.com/pentops/log.go v0.0.16
	github.com/pentops/o5-messaging v0.0.0-20250619024104-7e07c29129f0
	github.com/pentops/pgtest.go v0.0.0-20241223222214-7638cc50e15b
	github.com/pentops/protostate v0.0.0-20250627220531-ea6dfe7edca9
	github.com/pentops/realms v0.0.0-20250619030211-be302569b3fc
	github.com/pentops/runner v0.0.0-20250619010747-2bb7a5385324
	github.com/pentops/sqrlx.go v0.0.0-20250520210217-2f46de329c7a
	github.com/pkg/errors v0.9.1
	github.com/pressly/goose v2.7.0+incompatible
	github.com/stretchr/testify v1.10.0
	github.com/tidwall/sjson v1.2.5
	golang.org/x/exp v0.0.0-20250606033433-dcc06ee1d476
	golang.org/x/oauth2 v0.30.0
	golang.org/x/text v0.26.0
	google.golang.org/genproto/googleapis/api v0.0.0-20250603155806-513f23925822
	google.golang.org/grpc v1.73.0
	google.golang.org/protobuf v1.36.6
)

require (
	buf.build/go/protovalidate v0.13.1 // indirect
	cel.dev/expr v0.24.0 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.11 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.32 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.36 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.36 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.36 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.7.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.18.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.25.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.30.3 // indirect
	github.com/bufbuild/protocompile v0.14.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/cel-go v0.25.0 // indirect
	github.com/google/go-github/v29 v29.0.3 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.3.2 // indirect
	github.com/jhump/protoreflect v1.17.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pentops/golib v0.0.0-20250326060930-8c83d58ddb63 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/stoewer/go-strcase v1.3.1 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/sync v0.15.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250603155806-513f23925822 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/awslabs/goformation/v7 => github.com/twisp/goformation/v7 v7.0.0-20250225211955-a6db284e8d3f // lint:ignore
