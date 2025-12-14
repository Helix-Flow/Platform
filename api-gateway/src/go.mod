module helixflow/api-gateway

go 1.22.2

require (
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.3
	github.com/redis/go-redis/v9 v9.17.2
	google.golang.org/grpc v1.64.0
	helixflow/api-gateway/auth v0.0.0-00010101000000-000000000000
	helixflow/api-gateway/inference v0.0.0-00010101000000-000000000000
	helixflow/api-gateway/monitoring v0.0.0-00010101000000-000000000000
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	golang.org/x/net v0.22.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/text v0.17.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240318140521-94a12d6c2237 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)

replace helixflow/api-gateway/inference => ./inference

replace helixflow/api-gateway/auth => ./auth

replace helixflow/api-gateway/monitoring => ./monitoring
