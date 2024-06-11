module github.com/pentops/o5-deploy-aws

go 1.22.3

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.34.1-20240508200655-46a4cf4ba109.1
	github.com/aws/aws-sdk-go-v2 v1.27.0
	github.com/aws/aws-sdk-go-v2/config v1.27.16
	github.com/aws/aws-sdk-go-v2/credentials v1.17.16
	github.com/aws/aws-sdk-go-v2/service/cloudformation v1.41.1
	github.com/aws/aws-sdk-go-v2/service/ecs v1.35.1
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.26.1
	github.com/aws/aws-sdk-go-v2/service/s3 v1.54.3
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.25.1
	github.com/aws/aws-sdk-go-v2/service/sns v1.26.1
	github.com/aws/aws-sdk-go-v2/service/sts v1.28.10
	github.com/aws/smithy-go v1.20.2
	github.com/awslabs/goformation/v7 v7.12.11
	github.com/bradleyfalzon/ghinstallation v1.1.1
	github.com/bufbuild/protovalidate-go v0.6.2
	github.com/elgris/sqrl v0.0.0-20210727210741-7e0198b30236
	github.com/goccy/go-yaml v1.11.2
	github.com/google/go-github/v47 v47.1.0
	github.com/google/uuid v1.6.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0
	github.com/iancoleman/strcase v0.3.0
	github.com/lib/pq v1.10.9
	github.com/pentops/envconf.go v0.0.0-20240607014745-65480a3da726
	github.com/pentops/flowtest v0.0.0-20240607020132-7dda31076e79
	github.com/pentops/go-grpc-helpers v0.0.0-20230815045451-2524ee695ebb
	github.com/pentops/j5 v0.0.0-20240606040938-7f6c198fdc0d
	github.com/pentops/log.go v0.0.0-20240523172444-85c9292a83db
	github.com/pentops/o5-go v0.0.0-20240611065114-e8143d8387f4
	github.com/pentops/o5-messaging.go v0.0.0-20240610220232-02f44a1806f1
	github.com/pentops/pgtest.go v0.0.0-20240604005819-2035f4562734
	github.com/pentops/protostate v0.0.0-20240610211526-361e6c3ed773
	github.com/pentops/runner v0.0.0-20240525192419-d655233635e9
	github.com/pentops/sqrlx.go v0.0.0-20240523172712-b615a994d8c0
	github.com/pressly/goose v2.7.0+incompatible
	github.com/stretchr/testify v1.9.0
	github.com/tidwall/sjson v1.2.5
	golang.org/x/oauth2 v0.18.0
	golang.org/x/text v0.16.0
	google.golang.org/genproto/googleapis/api v0.0.0-20240610135401-a8a62080eff3
	google.golang.org/grpc v1.64.0
	google.golang.org/protobuf v1.34.2-0.20240529085009-ca837e5c658b
)

require (
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.2 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.7 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.7 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.0 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.11.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.3.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.11.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.17.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.20.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.24.3 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/fatih/color v1.17.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/cel-go v0.20.1 // indirect
	github.com/google/go-github/v29 v29.0.3 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stoewer/go-strcase v1.3.0 // indirect
	github.com/tidwall/gjson v1.17.1 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	golang.org/x/crypto v0.24.0 // indirect
	golang.org/x/exp v0.0.0-20240604190554-fc45aab8b7f8 // indirect
	golang.org/x/net v0.26.0 // indirect
	golang.org/x/sys v0.21.0 // indirect
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240610135401-a8a62080eff3 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
