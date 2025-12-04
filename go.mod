module github.com/hexiosec/asm-cloud-connector

go 1.24.4

require (
	cloud.google.com/go/asset v1.22.0
	cloud.google.com/go/certificatemanager v1.9.6
	cloud.google.com/go/iam v1.5.3
	cloud.google.com/go/storage v1.57.2
	github.com/Masterminds/semver/v3 v3.4.0
	github.com/aws/aws-lambda-go v1.50.0
	github.com/aws/aws-sdk-go-v2 v1.39.6
	github.com/aws/aws-sdk-go-v2/config v1.31.20
	github.com/aws/aws-sdk-go-v2/credentials v1.18.24
	github.com/aws/aws-sdk-go-v2/service/acm v1.37.13
	github.com/aws/aws-sdk-go-v2/service/apigateway v1.36.3
	github.com/aws/aws-sdk-go-v2/service/apigatewayv2 v1.32.13
	github.com/aws/aws-sdk-go-v2/service/cloudfront v1.56.2
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.269.0
	github.com/aws/aws-sdk-go-v2/service/eks v1.74.9
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.53.0
	github.com/aws/aws-sdk-go-v2/service/lambda v1.81.3
	github.com/aws/aws-sdk-go-v2/service/opensearch v1.53.2
	github.com/aws/aws-sdk-go-v2/service/organizations v1.46.4
	github.com/aws/aws-sdk-go-v2/service/rds v1.109.0
	github.com/aws/aws-sdk-go-v2/service/route53 v1.59.5
	github.com/aws/aws-sdk-go-v2/service/s3 v1.90.2
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.39.13
	github.com/aws/aws-sdk-go-v2/service/sts v1.40.2
	github.com/aws/smithy-go v1.23.2
	github.com/go-playground/validator/v10 v10.28.0
	github.com/go-resty/resty/v2 v2.16.5
	github.com/hashicorp/go-retryablehttp v0.7.8
	github.com/hexiosec/asm-sdk-go v1.0.0
	github.com/joho/godotenv v1.5.1
	github.com/mitchellh/mapstructure v1.5.0
	github.com/rs/zerolog v1.34.0
	github.com/sethvargo/go-envconfig v1.3.0
	github.com/stretchr/testify v1.11.1
	golang.org/x/net v0.47.0
	google.golang.org/api v0.256.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251124214823-79d6a2a48846
	google.golang.org/grpc v1.77.0
	google.golang.org/protobuf v1.36.10
	gopkg.in/yaml.v3 v3.0.1
)

require (
	cel.dev/expr v0.24.0 // indirect
	cloud.google.com/go v0.121.6 // indirect
	cloud.google.com/go/accesscontextmanager v1.9.6 // indirect
	cloud.google.com/go/auth v0.17.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	cloud.google.com/go/longrunning v0.7.0 // indirect
	cloud.google.com/go/monitoring v1.24.2 // indirect
	cloud.google.com/go/orgpolicy v1.15.0 // indirect
	cloud.google.com/go/osconfig v1.15.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.30.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.53.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.53.0 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.3 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.13 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.13 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.13 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.4 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.7 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cncf/xds/go v0.0.0-20251022180443-0feb69152e9f // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/envoyproxy/go-control-plane/envoy v1.35.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.2.1 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/gabriel-vasile/mimetype v1.4.10 // indirect
	github.com/go-jose/go-jose/v4 v4.1.3 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.7 // indirect
	github.com/googleapis/gax-go/v2 v2.15.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/spiffe/go-spiffe/v2 v2.6.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/detectors/gcp v1.38.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.61.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.61.0 // indirect
	go.opentelemetry.io/otel v1.38.0 // indirect
	go.opentelemetry.io/otel/metric v1.38.0 // indirect
	go.opentelemetry.io/otel/sdk v1.38.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.38.0 // indirect
	go.opentelemetry.io/otel/trace v1.38.0 // indirect
	golang.org/x/crypto v0.44.0 // indirect
	golang.org/x/oauth2 v0.33.0 // indirect
	golang.org/x/sync v0.18.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	golang.org/x/time v0.14.0 // indirect
	google.golang.org/genproto v0.0.0-20250603155806-513f23925822 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20251022142026-3a174f9686a8 // indirect
	gopkg.in/validator.v2 v2.0.1 // indirect
)
