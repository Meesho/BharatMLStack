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


## inferflow-client

## Install

```xml

<dependency>
    <groupId>com.meesho.ml</groupId>
    <artifactId>inferflow-client</artifactId>
    <version>1.0.2-RELEASE</version>
</dependency>
```

### Properties:-

application.yml

```yml

grpc:
    inferflow-enabled: true 

client:
  inferflow-grpc:
    host: ${INFERFLOW_GRPC_HOST}
    port: ${INFERFLOW_GRPC_PORT}
    http2-config:
      grpc-deadline: ${INFERFLOW_GRPC_DEADLINE}
      connect-timeout: ${INFERFLOW_GRPC_CONNECT_TIMEOUT}
      keepAliveTime: ${INFERFLOW_GRPC_KEEP_ALIVE_TIMEOUT}
      connection-request-timeout: ${INFERFLOW_GRPC_CONN_REQUEST_TIMEOUT}
      pool-size: ${INFERFLOW_GRPC_CHANNEL_POOL_SIZE}
      thread-pool-size: ${INFERFLOW_GRPC_THREAD_POOL_SIZE}
      bounded-queue-size: ${INFERFLOW_GRPC_QUEUE_POOL_SIZE}
      is-plain-text: ${INFERFLOW_GRPC_PLAIN_TEXT}
      
```
    
Prod:
```properties
    INFERFLOW_GRPC_HOST=inferflow.cluster.meeshoint.in
    INFERFLOW_GRPC_PORT=80
    INFERFLOW_GRPC_DEADLINE=500
    INFERFLOW_GRPC_CONNECT_TIMEOUT=100
    INFERFLOW_GRPC_KEEP_ALIVE_TIMEOUT=10000
    INFERFLOW_GRPC_CONN_REQUEST_TIMEOUT=100
    INFERFLOW_GRPC_CHANNEL_POOL_SIZE=1
    INFERFLOW_GRPC_THREAD_POOL_SIZE=100
    INFERFLOW_GRPC_QUEUE_POOL_SIZE=100
    INFERFLOW_GRPC_PLAIN_TEXT=true
```    

To Connect Prod from Local:
```properties
    INFERFLOW_GRPC_HOST=inferflow.meesho.com
    INFERFLOW_GRPC_PORT=443
    INFERFLOW_GRPC_DEADLINE=50000
    INFERFLOW_GRPC_CONNECT_TIMEOUT=10000
    INFERFLOW_GRPC_KEEP_ALIVE_TIMEOUT=10000
    INFERFLOW_GRPC_CONN_REQUEST_TIMEOUT=10000
    INFERFLOW_GRPC_CHANNEL_POOL_SIZE=1
    INFERFLOW_GRPC_THREAD_POOL_SIZE=100
    INFERFLOW_GRPC_QUEUE_POOL_SIZE=100
    INFERFLOW_GRPC_PLAIN_TEXT=false
```
### Usage

API-1: Get model score for entities (eg.catalog)

Class: @Qualifier(InferflowConstants.BeanNames.INFERFLOW_SERVICE) IInferflow Inferflow;

Method: retrieveModelScore

* Note : Implement batching at client side and configure batch size as per your latency needs.



temp change