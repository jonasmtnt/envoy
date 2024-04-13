module github.com/envoyproxy/envoy/examples/golang-sni-routing/simple

// the version should >= 1.18
go 1.22

// NOTICE: these lines could be generated automatically by "go mod tidy"
require (
	github.com/cncf/xds/go v0.0.0-20231128003011-0fa0005c9caa
	github.com/envoyproxy/envoy v1.24.0
	google.golang.org/protobuf v1.33.0
)

require (
	github.com/envoyproxy/protoc-gen-validate v1.0.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/traefik/traefik/v2 v2.11.1
	golang.org/x/crypto v0.22.0
	google.golang.org/genproto/googleapis/api v0.0.0-20240102182953-50ed04b92917 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240102182953-50ed04b92917 // indirect
)

require (
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/go-acme/lego/v4 v4.16.1 // indirect
	github.com/go-jose/go-jose/v4 v4.0.1 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/miekg/dns v1.1.58 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pires/go-proxyproto v0.6.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/traefik/paerser v0.2.0 // indirect
	golang.org/x/mod v0.17.0 // indirect
	golang.org/x/net v0.24.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/sys v0.19.0 // indirect
	golang.org/x/tools v0.20.0 // indirect
)

// TODO: remove when #26173 lands.
replace github.com/envoyproxy/envoy => ../../..
