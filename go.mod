module github.com/pentops/o5-deploy-aws

go 1.24.0

toolchain go1.24.1

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.6-20250307204501-0409229c3780.1
	buf.build/go/protoyaml v0.3.1
	github.com/aws/aws-sdk-go-v2 v1.36.3
	github.com/aws/aws-sdk-go-v2/config v1.29.9
	github.com/aws/aws-sdk-go-v2/credentials v1.17.62
	github.com/aws/aws-sdk-go-v2/feature/rds/auth v1.5.11
	github.com/aws/aws-sdk-go-v2/service/cloudformation v1.58.1
	github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs v1.46.1
	github.com/aws/aws-sdk-go-v2/service/ecs v1.54.1
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.44.1
	github.com/aws/aws-sdk-go-v2/service/s3 v1.78.1
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.35.2
	github.com/aws/aws-sdk-go-v2/service/sns v1.34.1
	github.com/aws/aws-sdk-go-v2/service/sts v1.33.17
	github.com/aws/smithy-go v1.22.3
	github.com/awslabs/goformation/v7 v7.14.9
	github.com/bradleyfalzon/ghinstallation v1.1.1
	github.com/bufbuild/protovalidate-go v0.9.2
	github.com/elgris/sqrl v0.0.0-20210727210741-7e0198b30236
	github.com/google/go-cmp v0.7.0
	github.com/google/go-github/v47 v47.1.0
	github.com/google/uuid v1.6.0
	github.com/iancoleman/strcase v0.3.0
	github.com/lib/pq v1.10.9
	github.com/pentops/envconf.go v0.0.0-20241008010024-9864aef6219d
	github.com/pentops/flowtest v0.0.0-20250521181823-71b0be743b08
	github.com/pentops/grpc.go v0.0.0-20250326042738-bcdfc2b43fa9
	github.com/pentops/j5 v0.0.0-20250530005724-d535ce1060fa
	github.com/pentops/log.go v0.0.16
	github.com/pentops/o5-messaging v0.0.0-20250520213617-fba07334e9aa
	github.com/pentops/pgtest.go v0.0.0-20241223222214-7638cc50e15b
	github.com/pentops/protostate v0.0.0-20250520005750-91b4a48c3407
	github.com/pentops/realms v0.0.0-20250327015025-d65dc0463c4e
	github.com/pentops/runner v0.0.0-20250530005558-0b8e943f923e
	github.com/pentops/sqrlx.go v0.0.0-20250520210217-2f46de329c7a
	github.com/pkg/errors v0.9.1
	github.com/pressly/goose v2.7.0+incompatible
	github.com/stretchr/testify v1.10.0
	github.com/tidwall/sjson v1.2.5
	golang.org/x/exp v0.0.0-20250305212735-054e65f0b394
	golang.org/x/oauth2 v0.28.0
	golang.org/x/text v0.25.0
	google.golang.org/genproto/googleapis/api v0.0.0-20250324211829-b45e905df463
	google.golang.org/grpc v1.71.0
	google.golang.org/protobuf v1.36.6
)

require (
	cel.dev/expr v0.22.0 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.10 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.30 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.34 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.6.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.18.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.25.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.29.1 // indirect
	github.com/bufbuild/protocompile v0.14.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/cel-go v0.24.1 // indirect
	github.com/google/go-github/v29 v29.0.3 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.3.1 // indirect
	github.com/jhump/protoreflect v1.17.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pentops/golib v0.0.0-20250326060930-8c83d58ddb63 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/stoewer/go-strcase v1.3.0 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	golang.org/x/crypto v0.38.0 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/sync v0.14.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250324211829-b45e905df463 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/awslabs/goformation/v7 => github.com/twisp/goformation/v7 v7.0.0-20250225211955-a6db284e8d3f // lint:ignore
