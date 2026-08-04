module onyxdb-e2e-test

go 1.24.4

require (
	github.com/Meesho/BharatMLStack/onyxdb/controlplane v0.0.0
	github.com/Meesho/BharatMLStack/onyxdb/onyxdb-go-sdk v0.0.0
	go.etcd.io/etcd/client/v3 v3.5.21
)

require (
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/rs/zerolog v1.34.0 // indirect
	go.etcd.io/etcd/api/v3 v3.5.21 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.21 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.17.0 // indirect
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/text v0.27.0 // indirect
	google.golang.org/genproto v0.0.0-20230822172742-b8732ec3820d // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230822172742-b8732ec3820d // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230822172742-b8732ec3820d // indirect
	google.golang.org/grpc v1.59.0 // indirect
	google.golang.org/protobuf v1.36.9 // indirect
)

replace (
	github.com/Meesho/BharatMLStack/onyxdb/controlplane => ../../../onyxdb/controlplane
	github.com/Meesho/BharatMLStack/onyxdb/onyxdb-go-sdk => ../../../onyxdb/onyxdb-go-sdk
)
