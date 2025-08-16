module github.com/Meesho/BharatMLStack/go-caller

go 1.25.0

replace github.com/Meesho/BharatMLStack/go-sdk => ../go-sdk

require (
	github.com/Meesho/BharatMLStack/go-sdk v0.0.0
	google.golang.org/grpc v1.74.2
	google.golang.org/protobuf v1.36.7
)