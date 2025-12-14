module helixflow/auth-service

go 1.22.2

toolchain go1.22.2

require (
	github.com/golang-jwt/jwt/v5 v5.0.0
	golang.org/x/crypto v0.17.0
	google.golang.org/grpc v1.59.0
	helixflow/auth v0.0.0-00010101000000-000000000000
	helixflow/database v0.0.0-00010101000000-000000000000
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/mattn/go-sqlite3 v1.14.32 // indirect
	github.com/redis/go-redis/v9 v9.17.2 // indirect
	golang.org/x/net v0.17.0 // indirect
	golang.org/x/sys v0.17.0 // indirect
	golang.org/x/text v0.17.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230410155749-daa745c078e1 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)

replace helixflow/auth => ../../helixflow/auth

replace helixflow/database => ../../internal/database
