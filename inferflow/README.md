## inferflow

### Build / Debug / Run

* To build the app run
    * export GOOS=linux && go build -o bin/inferflow_app cmd/inferflow/main.go
* To run or debug the app
    * In IDE please run launch.json in VSCode; you can select debug mode or auto mode.

### Notes
* Configs are present in cmd/inferflow/application.env file as environment variables

* Use camel case for naming config keys in application.env file

* Make sure to not have "__" (double underscores) in actual config keys

* Environment variables don't support "." or "-" so we are using "__" (double underscores) as delimiter

* eg. if you have nested config parent.child.name then your environemnt variables should look like parent__child__name. In code you can access the  config as kConfig.String("parent.child.name")

* Do not panic in code except during app start; if you want to panic just return an error

* Automaxprocs issue (GOMAXPROCS = 1) may cause throttling in kubernetes if cpu_limit is less than 1000m (1 core). Make sure you use atleast 1 core in EKS

* To add test case for a file abc.go add a file abc_test.go in the same package

* Always check if the variables being shared amongst goroutines are concurrently safe or not
    
* Headers are automatically converted to PascalCase with mux router

* TELEGRAF_UDP_HOST and TELEGRAF_UDP_PORT are used for EKS, don't add or change these environment variables in code. application.env doesn't have these variables because default values are handled in code


### Packaging structure

 * cmd/ contains main package of the application and application.env
 * internal/ application specific routers, errors, etc. packages
 * handlers/ handlers for various APIs exposed along with their commons & initialization
 * pkg/ utils that can be used by other apps
 * test/ packaged code that is required for unit testing
 * deployments/ contain deployment related files


## Go SDK Client

The Go client for Inferflow is available in the BharatMLStack Go SDK at [`go-sdk/pkg/clients/inferflow`](../go-sdk/pkg/clients/inferflow/).

It supports all three Predict service APIs over gRPC:

| API | Method | Use Case |
|-----|--------|----------|
| **PointWise** | `InferPointWise` | Per-target scoring (CTR, fraud, relevance) |
| **PairWise** | `InferPairWise` | Pair-level ranking (preference learning) |
| **SlateWise** | `InferSlateWise` | Group-level scoring (page optimization, diversity-aware reranking) |

### Quick Start

```go
import "github.com/Meesho/BharatMLStack/go-sdk/pkg/clients/inferflow"

client := inferflow.GetInferflowClientFromConfig(1, inferflow.ClientConfig{
    Host:             "inferflow.svc",
    Port:             "8080",
    DeadlineExceedMS: 500,
    PlainText:        true,
}, "my-service")

resp, err := client.InferPointWise(&grpc.PointWiseRequest{
    ModelConfigId: "ranking_model_v1",
    TrackingId:    "req-123",
    Targets:       targets,
})
```

See the full [Go SDK Inferflow README](../go-sdk/pkg/clients/inferflow/README.md) for detailed usage, configuration options, and examples for all three APIs.