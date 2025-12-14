module helixflow/api-gateway

go 1.24.0

toolchain go1.24.11

require (
	github.com/gorilla/mux v1.8.1
	github.com/redis/go-redis/v9 v9.17.2
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	golang.org/x/net v0.46.1-0.20251013234738-63d1a5100f82 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251022142026-3a174f9686a8 // indirect
	google.golang.org/grpc v1.77.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	helixflow/api-gateway/auth v0.0.0-00010101000000-000000000000 // indirect
	helixflow/api-gateway/inference v0.0.0-00010101000000-000000000000 // indirect
	helixflow/api-gateway/monitoring v0.0.0-00010101000000-000000000000 // indirect
)

replace helixflow/api-gateway/inference => ./inference

replace helixflow/api-gateway/auth => ./auth

replace helixflow/api-gateway/monitoring => ./monitoring
