module helixflow/monitoring-service

go 1.22.2

require (
	github.com/redis/go-redis/v9 v9.17.2
	google.golang.org/grpc v1.64.0
	helixflow/monitoring v0.0.0-00010101000000-000000000000
)

replace helixflow/monitoring => ../../helixflow/monitoring

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	golang.org/x/net v0.22.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/text v0.17.0 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)
