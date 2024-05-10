module github.com/pentops/o5-deploy-aws

go 1.21.4

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.34.1-20240508200655-46a4cf4ba109.1
	github.com/aws/aws-sdk-go-v2 v1.24.1
	github.com/aws/aws-sdk-go-v2/config v1.25.10
	github.com/aws/aws-sdk-go-v2/credentials v1.16.8
	github.com/aws/aws-sdk-go-v2/service/cloudformation v1.41.1
	github.com/aws/aws-sdk-go-v2/service/ecs v1.35.1
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.26.1
	github.com/aws/aws-sdk-go-v2/service/s3 v1.47.1
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.25.1
	github.com/aws/aws-sdk-go-v2/service/sns v1.26.1
	github.com/aws/aws-sdk-go-v2/service/sts v1.26.1
	github.com/aws/smithy-go v1.19.0
	github.com/awslabs/goformation/v7 v7.12.11
	github.com/bradleyfalzon/ghinstallation v1.1.1
	github.com/bufbuild/protovalidate-go v0.6.2
	github.com/elgris/sqrl v0.0.0-20210727210741-7e0198b30236
	github.com/goccy/go-yaml v1.11.2
	github.com/google/go-github/v47 v47.1.0
	github.com/google/uuid v1.6.0
	github.com/iancoleman/strcase v0.3.0
	github.com/lib/pq v1.10.9
	github.com/pentops/flowtest v0.0.0-20240510221023-0114d43723cb
	github.com/pentops/jsonapi v0.0.0-20240510221406-91c55f7e9b17
	github.com/pentops/log.go v0.0.0-20231218074934-67aedcab3fa4
	github.com/pentops/o5-go v0.0.0-20240510231138-3ff8b9ff087e
	github.com/pentops/outbox.pg.go v0.0.0-20231222014950-493c01cfbcc7
	github.com/pentops/pgtest.go v0.0.0-20231220005207-f01c870bad2e
	github.com/pentops/protostate v0.0.0-20240510232107-56452b1fdca1
	github.com/pentops/runner v0.0.0-20240423175422-9ce56dcd8e33
	github.com/pentops/sqrlx.go v0.0.0-20240108202916-8687fdf983c0
	github.com/pressly/goose v2.7.0+incompatible
	github.com/stretchr/testify v1.9.0
	github.com/tidwall/sjson v1.2.5
	golang.org/x/oauth2 v0.17.0
	golang.org/x/text v0.15.0
	google.golang.org/genproto/googleapis/api v0.0.0-20240509183442-62759503f434
	google.golang.org/grpc v1.63.2
	google.golang.org/protobuf v1.34.1
	gopkg.daemonl.com/envconf v0.0.0-20220909014755-d65ec77bd452
)

require (
	github.com/antlr4-go/antlr/v4 v4.13.0 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.5.4 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.14.8 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.2.10 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.5.10 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.7.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.2.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.10.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.2.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.10.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.16.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.18.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.21.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/cel-go v0.20.1 // indirect
	github.com/google/go-github/v29 v29.0.3 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/pprof v0.0.0-20210720184732-4bb14d4b1be1 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stoewer/go-strcase v1.3.0 // indirect
	github.com/tidwall/gjson v1.17.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	golang.org/x/crypto v0.23.0 // indirect
	golang.org/x/exp v0.0.0-20240506185415-9bf2ced13842 // indirect
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240509183442-62759503f434 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
