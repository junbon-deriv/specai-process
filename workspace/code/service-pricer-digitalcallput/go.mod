module github.com/regentmarkets/service-pricer-digitalcallput

go 1.25.4

require (
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.4
	github.com/regentmarkets/service-feed v0.0.0-00010101000000-000000000000
	github.com/shopspring/decimal v1.4.0
	google.golang.org/grpc v1.78.0
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.6.0
	google.golang.org/protobuf v1.36.11
	gopkg.in/yaml.v3 v3.0.1
)

require (
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20251222181119-0a764e51fe1b // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251222181119-0a764e51fe1b // indirect
)

replace github.com/regentmarkets/service-feed => github.com/junbon-deriv/service-feed v0.0.0-20251222062003-73b93b86843a
