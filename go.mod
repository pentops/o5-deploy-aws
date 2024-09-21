module github.com/pentops/o5-deploy-aws

go 1.22.4

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.34.2-20240717164558-a6c49f84cc0f.2
	github.com/aws/aws-sdk-go-v2 v1.30.3
	github.com/aws/aws-sdk-go-v2/config v1.27.27
	github.com/aws/aws-sdk-go-v2/credentials v1.17.27
	github.com/aws/aws-sdk-go-v2/service/cloudformation v1.53.3
	github.com/aws/aws-sdk-go-v2/service/ecs v1.44.3
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.34.0
	github.com/aws/aws-sdk-go-v2/service/s3 v1.58.3
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.32.4
	github.com/aws/aws-sdk-go-v2/service/sns v1.31.3
	github.com/aws/aws-sdk-go-v2/service/sts v1.30.3
	github.com/aws/smithy-go v1.20.3
	github.com/awslabs/goformation/v7 v7.14.9
	github.com/bradleyfalzon/ghinstallation v1.1.1
	github.com/bufbuild/protovalidate-go v0.6.3
	github.com/elgris/sqrl v0.0.0-20210727210741-7e0198b30236
	github.com/goccy/go-yaml v1.12.0
	github.com/google/go-github/v47 v47.1.0
	github.com/google/uuid v1.6.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0
	github.com/iancoleman/strcase v0.3.0
	github.com/lib/pq v1.10.9
	github.com/pentops/envconf.go v0.0.0-20240806040806-dcab509e8c71
	github.com/pentops/flowtest v0.0.0-20240909065422-e4f7f68d5fb0
	github.com/pentops/j5 v0.0.0-20240909174102-37b5e03dca13
	github.com/pentops/log.go v0.0.0-20240806161938-2742d05b4c24
	github.com/pentops/o5-messaging v0.0.0-20240810013929-db56de35f3ed
	github.com/pentops/pgtest.go v0.0.0-20240806042712-cca5bdfe6542
	github.com/pentops/protostate v0.0.0-20240909062736-d8c64478c58d
	github.com/pentops/realms v0.0.0-20240812224742-adcdac1a2705
	github.com/pentops/runner v0.0.0-20240806162317-0eb1ced9ab3d
	github.com/pentops/sqrlx.go v0.0.0-20240806064322-33adc0ac5bd4
	github.com/pressly/goose v2.7.0+incompatible
	github.com/stretchr/testify v1.9.0
	github.com/tidwall/sjson v1.2.5
	golang.org/x/oauth2 v0.22.0
	golang.org/x/text v0.18.0
	google.golang.org/genproto/googleapis/api v0.0.0-20240903143218-8af14fe29dc1
	google.golang.org/grpc v1.66.0
	google.golang.org/protobuf v1.34.2
)

require (
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.3 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.11 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.15 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.15 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.0 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.11.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.3.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.11.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.17.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.22.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.26.4 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/fatih/color v1.17.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/cel-go v0.21.0 // indirect
	github.com/google/go-github/v29 v29.0.3 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stoewer/go-strcase v1.3.0 // indirect
	github.com/tidwall/gjson v1.17.3 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	golang.org/x/crypto v0.26.0 // indirect
	golang.org/x/exp v0.0.0-20240904232852-e7e105dedf7e // indirect
	golang.org/x/net v0.28.0 // indirect
	golang.org/x/sys v0.25.0 // indirect
	golang.org/x/xerrors v0.0.0-20240716161551-93cc26a95ae9 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240903143218-8af14fe29dc1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
