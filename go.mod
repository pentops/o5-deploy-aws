module github.com/pentops/o5-deploy-aws

go 1.21.4

require (
	github.com/aws/aws-sdk-go-v2 v1.23.1
	github.com/aws/aws-sdk-go-v2/config v1.25.5
	github.com/aws/aws-sdk-go-v2/credentials v1.16.4
	github.com/aws/aws-sdk-go-v2/service/cloudformation v1.40.1
	github.com/aws/aws-sdk-go-v2/service/ecs v1.33.2
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.24.3
	github.com/aws/aws-sdk-go-v2/service/s3 v1.44.0
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.23.3
	github.com/aws/aws-sdk-go-v2/service/sns v1.25.3
	github.com/aws/aws-sdk-go-v2/service/sts v1.25.4
	github.com/aws/smithy-go v1.17.0
	github.com/awslabs/goformation/v7 v7.12.11
	github.com/bradleyfalzon/ghinstallation v1.1.1
	github.com/bufbuild/protovalidate-go v0.4.2
	github.com/elgris/sqrl v0.0.0-20210727210741-7e0198b30236
	github.com/goccy/go-yaml v1.11.2
	github.com/golang/protobuf v1.5.3
	github.com/google/go-github/v47 v47.1.0
	github.com/google/uuid v1.4.0
	github.com/lib/pq v1.10.9
	github.com/pentops/genericstate v0.0.0-20231121014125-b51b708919ca
	github.com/pentops/jsonapi v0.0.0-20231123011150-74903e21d297
	github.com/pentops/log.go v0.0.0-20230815045424-6ebbd9ef2576
	github.com/pentops/o5-go v0.0.0-20231123034227-ac9197d1359d
	github.com/pentops/outbox.pg.go v0.0.0-20230801052616-dc5e96f581f8
	github.com/pressly/goose v2.7.0+incompatible
	github.com/rs/cors v1.10.1
	github.com/stretchr/testify v1.8.4
	github.com/tidwall/sjson v1.2.5
	golang.org/x/oauth2 v0.14.0
	golang.org/x/sync v0.5.0
	golang.org/x/text v0.14.0
	google.golang.org/grpc v1.59.0
	google.golang.org/protobuf v1.31.1-0.20231027082548-f4a6c1f6e5c1
	gopkg.daemonl.com/envconf v0.0.0-20220909014755-d65ec77bd452
	gopkg.daemonl.com/log v0.0.13
	gopkg.daemonl.com/sqrlx v0.0.26-0.20231109230408-4e2718f3736f
)

require (
	buf.build/gen/go/bufbuild/buf/grpc/go v1.3.0-20231121234731-c3da3a155e4f.2 // indirect
	buf.build/gen/go/bufbuild/buf/protocolbuffers/go v1.31.0-20231121234731-c3da3a155e4f.2 // indirect
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.31.0-20231115204500-e097f827e652.2 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.0 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.5.1 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.14.5 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.2.4 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.5.4 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.7.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.2.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.10.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.2.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.10.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.16.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.17.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.20.1 // indirect
	github.com/bufbuild/protocompile v0.6.1-0.20231108163138-146b831231f7 // indirect
	github.com/bufbuild/protoyaml-go v0.1.7 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/go-yaml/yaml v2.1.0+incompatible // indirect
	github.com/google/cel-go v0.18.2 // indirect
	github.com/google/go-github/v29 v29.0.3 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/jhump/protoreflect v1.15.3 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pentops/listify-go v0.0.0-20231114210340-3454f5bb0a53 // indirect
	github.com/pentops/sugar-go v0.0.0-20231029194349-ec12ec0132c5 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stoewer/go-strcase v1.3.0 // indirect
	github.com/tidwall/gjson v1.17.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	golang.org/x/crypto v0.15.0 // indirect
	golang.org/x/exp v0.0.0-20231110203233-9a3e6036ecaa // indirect
	golang.org/x/net v0.18.0 // indirect
	golang.org/x/sys v0.14.0 // indirect
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto v0.0.0-20231120223509-83a465c0220f // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20231120223509-83a465c0220f // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231120223509-83a465c0220f // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// replace github.com/pentops/o5-go => ../o5-go

//replace github.com/pentops/genericstate => ../genericstate
