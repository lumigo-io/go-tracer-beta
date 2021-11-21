module github.com/lumigo-io/go-tracer

go 1.17

require (
	github.com/aws/aws-lambda-go v1.27.0
	github.com/aws/aws-sdk-go-v2 v1.11.1
	github.com/aws/aws-sdk-go-v2/config v1.10.2
	github.com/google/uuid v1.3.0
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/viper v1.9.0
	go.opentelemetry.io/contrib/detectors/aws/lambda v0.27.0
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda v0.27.0
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws v0.27.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.27.0
	go.opentelemetry.io/otel v1.2.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.2.0
	go.opentelemetry.io/otel/sdk v1.2.0
)

require (
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.0.0 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.6.2 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.8.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.0.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.5.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.5.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.9.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.19.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.6.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.10.1 // indirect
	github.com/aws/smithy-go v1.9.0 // indirect
	github.com/felixge/httpsnoop v1.0.2 // indirect
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/magiconair/properties v1.8.5 // indirect
	github.com/mitchellh/mapstructure v1.4.2 // indirect
	github.com/pelletier/go-toml v1.9.4 // indirect
	github.com/spf13/afero v1.6.0 // indirect
	github.com/spf13/cast v1.4.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.2.0 // indirect
	go.opentelemetry.io/otel/internal/metric v0.25.0 // indirect
	go.opentelemetry.io/otel/metric v0.25.0 // indirect
	go.opentelemetry.io/otel/trace v1.2.0 // indirect
	golang.org/x/sys v0.0.0-20210823070655-63515b42dcdf // indirect
	golang.org/x/text v0.3.6 // indirect
	gopkg.in/ini.v1 v1.63.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
