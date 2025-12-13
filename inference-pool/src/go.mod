module helixflow/inference-pool

go 1.24.0

toolchain go1.24.11

require (
	google.golang.org/grpc v1.77.0
	helixflow/inference v0.0.0-00010101000000-000000000000
)

require (
	golang.org/x/net v0.46.1-0.20251013234738-63d1a5100f82 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)

replace helixflow/inference => ../../helixflow/inference
