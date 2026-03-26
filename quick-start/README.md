# BharatML Stack Quick Start Guide
[![Discord](https://img.shields.io/badge/Discord-Join%20Chat-7289da?style=flat&logo=discord&logoColor=white)](https://discord.gg/XkT7XsV2AU)

A quick way to get the BharatML Stack Online Feature Store platform up and running locally for development and testing.

## Prerequisites

- Docker and Docker Compose
- Go 1.22 or later
- `nc` (netcat) command for connectivity checks
- Bash shell
- `grpcurl` for testing gRPC API endpoints (install from [https://github.com/fullstorydev/grpcurl](https://github.com/fullstorydev/grpcurl))

### Optional Prerequisites

**For Kubernetes support:**

- **kind** (Kubernetes in Docker) - Required if you want to use Kubernetes features

  **macOS:**
  ```bash
  brew install kind
  ```

  **Linux:**
  ```bash
  curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
  chmod +x ./kind
  sudo mv ./kind /usr/local/bin/kind
  ```

  For other platforms or the latest version, visit: https://kind.sigs.k8s.io/docs/user/quick-start/#installation

## System Components

BharatMLStack's Online Feature Store consists of several interconnected services:

**Infrastructure Services:**
- **ScyllaDB**: NoSQL database for high-performance feature storage
- **MySQL**: Relational database for metadata and configuration
- **Redis**: In-memory data store for caching
- **etcd**: Distributed key-value store for service coordination
- **Apache Kafka**: Message broker for feature ingestion pipeline

**Application Services:**
- **Online Feature Store gRPC API Server**: High-performance gRPC interface (runs on port 8089)
- **ONFS Consumer**: Kafka consumer service for real-time feature ingestion from message streams (runs on port 8090)
- **Horizon**: Backend API service (runs on port 8082)
- **Numerix**: Matrix operations service (runs on port 8083)
- **Trufflebox UI**: Frontend web interface (runs on port 3000)
- **Inferflow**: Inference workflow service (runs on port 8085)

**Management Tools:**
- **etcd Workbench**: etcd management interface (runs on port 8081)
- **Kafka UI**: Kafka cluster management interface (runs on port 8084)
- **Kubernetes (kind)**: Local Kubernetes cluster (optional, requires kind installation)

All services are orchestrated using Docker Compose with pre-built images from GitHub Container Registry (GHCR).

## Quick Start

### Starting the System

The start script provides an interactive service selector that allows you to choose which application services to run:

```bash
./start.sh
```

**Interactive Options:**
1. **All Services** - Starts all application services (API Server, Consumer, Horizon, Numerix, TruffleBox UI, Inferflow)
2. **Custom Selection** - Choose individual services to start
3. **Exit** - Exit without starting

**Infrastructure services (ScyllaDB, MySQL, Redis, etcd, Kafka) and Management Tools (etcd-workbench, kafka-ui) are always started.**

**Optional Services:**
- **Kubernetes (kind cluster)** - Local Kubernetes cluster for container orchestration

**Note:** Predator is not a Docker Compose service. It is deployed via ArgoCD in Kubernetes. If you need to set up Predator, see the [Predator Local Setup Guide](PREDATOR_SETUP.md) for complete instructions.

During interactive mode, you'll be prompted to optionally include Kubernetes. For Predator setup, you'll need to follow the comprehensive guide in [PREDATOR_SETUP.md](PREDATOR_SETUP.md) which includes ArgoCD installation, GitHub App setup, and repository configuration.

### Initializing with Dummy Data

For testing and development, you can initialize the databases with sample dummy data. This includes:
- Sample entities (user, catalog)
- Feature groups and features
- Example configurations for Inferflow and Numerix
- Test data in MySQL, ScyllaDB, and etcd

**Usage:**

```bash
# Interactive mode with dummy data
./start.sh --dummy-data

# Start all services with dummy data (non-interactive)
./start.sh --all --dummy-data

# Combine with version specification
ONFS_VERSION=v1.2.0 ./start.sh --all --dummy-data

# Combine with local builds
ONFS_VERSION=local ./start.sh --dummy-data
```

**What gets initialized:**

- **ScyllaDB**: Creates keyspace, tables, and inserts sample feature data
- **MySQL**: Creates all tables and inserts:
  - API resolvers
  - Role permissions
  - Sample entities, feature groups, and features
  - Inferflow and Numerix configurations
  - Service deployable configurations
- **etcd**: Sets up configuration keys with:
  - Security tokens
  - Entity and feature group configurations
  - Inferflow component configurations
  - Model and expression configurations

**Note:** The dummy data initialization scripts run automatically when you use the `--dummy-data` flag. The regular initialization scripts (which create empty schemas) are used by default.

### Service Independence

Services can run independently based on your needs:
- **ONFS API Server** - For direct gRPC feature operations
- **ONFS Consumer** - For real-time Kafka-based feature ingestion (independent of API Server)
- **Horizon + TruffleBox UI** - For web-based feature store management
- **Numerix** - For matrix operations
- **Inferflow** - For inference workflow management

### Specifying Service Versions

You can specify versions for application services using environment variables:

```bash
# Specify individual service versions
ONFS_VERSION=v1.2.3 ./start.sh

# Specify consumer version
ONFS_CONSUMER_VERSION=v1.0.0-beta-d74137 ./start.sh

# Combine multiple versions
ONFS_VERSION=v1.2.0 HORIZON_VERSION=v2.1.0 TRUFFLEBOX_VERSION=v1.0.5 INFERFLOW_VERSION=v1.0.0 ./start.sh

# Start all services with specific versions
ONFS_VERSION=v1.2.0 ONFS_CONSUMER_VERSION=v1.0.0-beta-d74137 INFERFLOW_VERSION=v1.0.0 ./start.sh

# Build services from local source code
ONFS_VERSION=local HORIZON_VERSION=local INFERFLOW_VERSION=local ./start.sh
```

**Available Environment Variables:**
- `ONFS_VERSION` - Online Feature Store API Server version
- `ONFS_CONSUMER_VERSION` - ONFS Consumer version
- `HORIZON_VERSION` - Horizon Backend version
- `NUMERIX_VERSION` - Numerix Matrix Operations version
- `TRUFFLEBOX_VERSION` - TruffleBox UI version
- `INFERFLOW_VERSION` - Inferflow version

**Version Formats:**
- `latest` (default) - Latest stable release
- `main` - Latest development build  
- `v1.2.3` - Specific version tag
- `sha-abcd1234` - Specific commit SHA
- `local` - Build from local Dockerfile (requires source code in workspace)

**Non-interactive Mode:**
```bash
# Start all services without prompts
./start.sh --all

# Start with specific versions non-interactively
ONFS_VERSION=v1.2.0 ./start.sh --all
```

### Building from Local Source

You can build services from local source code by setting the version to `local`. This is useful for development and testing changes:

```bash
# Build a single service from local source
ONFS_VERSION=local ./start.sh

# Build multiple services from local source
ONFS_VERSION=local HORIZON_VERSION=local INFERFLOW_VERSION=local ./start.sh

# Build all services from local source
ONFS_VERSION=local ONFS_CONSUMER_VERSION=local HORIZON_VERSION=local \
  NUMERIX_VERSION=local TRUFFLEBOX_VERSION=local INFERFLOW_VERSION=local ./start.sh
```

**Requirements for Local Builds:**
- Source code must be available in the parent directory (relative to `quick-start/`)
- Python 3 must be installed (for modifying docker-compose.yml)
- Docker must be able to build the Dockerfiles

**How it works:**
1. The script copies the source directory to the workspace
2. Modifies `docker-compose.yml` to use `build` instead of `image`
3. Docker Compose builds the image from the local Dockerfile
4. The built image is used to start the container

**Note:** Local builds require the source code structure to match the expected Dockerfile locations:
- `online-feature-store/cmd/api-server/DockerFile` for ONFS API Server
- `online-feature-store/cmd/consumer/DockerFile` for ONFS Consumer
- `horizon/cmd/horizon/Dockerfile` for Horizon
- `numerix/Dockerfile` for Numerix
- `trufflebox-ui/DockerFile` for TruffleBox UI
- `inferflow/cmd/inferflow/Dockerfile` for Inferflow

**Advanced: Direct docker-compose Usage**

You can also work directly with docker-compose in the workspace directory:

```bash
# Change to workspace directory
cd workspace

# Start specific service with version
ONFS_VERSION=main docker-compose up -d onfs-api-server

# Start multiple services with different versions
ONFS_VERSION=v1.2.0 HORIZON_VERSION=v2.1.0 docker-compose up -d onfs-api-server horizon

# Restart a service with a different version
ONFS_CONSUMER_VERSION=v1.0.0-beta-d74137 docker-compose up -d onfs-consumer

# View specific service
docker-compose ps onfs-api-server
```

This will:
1. Check for Go installation (1.22+ required)
2. Create a workspace directory with configuration files
3. Pull and start all services using `docker-compose up -d`
4. Wait for services to become healthy
5. Initialize databases with required schemas
6. Display access information and helpful commands

Once complete, you can access:
- **Trufflebox UI**: http://localhost:3000
- **Horizon API**: http://localhost:8082
- **Online Feature Store gRPC API**: http://localhost:8089
- **ONFS Consumer**: http://localhost:8090 (health check)
- **Numerix**: http://localhost:8083
- **Inferflow**: http://localhost:8085
- **etcd Workbench**: http://localhost:8081
- **Kafka UI**: http://localhost:8084
- **Kubernetes** (if enabled): Use `kubectl cluster-info --context kind-bharatml-stack`
- **ArgoCD** (if installed): http://localhost:8087 (requires port-forward - see [Predator Setup Guide](PREDATOR_SETUP.md))

**Note:** Predator is accessed through ArgoCD UI after completing the setup in [PREDATOR_SETUP.md](PREDATOR_SETUP.md).

### Stopping the System

To stop all services:

```bash
./stop.sh
```

To stop and completely purge all containers, volumes, and workspace:

```bash
./stop.sh --purge
```

## Accessing Services

### Frontend UI
- **URL**: http://localhost:3000
- **Default admin credentials**:
  - Email: `admin@admin.com`
  - Password: `admin`

### API Endpoints
- **Horizon API**: http://localhost:8082
  - Health check: http://localhost:8082/health
- **ONFS gRPC API**: http://localhost:8089
  - Health check: http://localhost:8089/health/self
- **ONFS Consumer**: http://localhost:8090
  - Health check: http://localhost:8090/health/self
- **Numerix**: http://localhost:8083
  - Health check: http://localhost:8083/health
- **Inferflow**: http://localhost:8085
  - Health check: http://localhost:8085/health/self

### Kubernetes (Optional)

**Kubernetes (kind cluster):**
- **Cluster Name**: `bharatml-stack`
- **Context**: `kind-bharatml-stack`
- **Access**: Use `kubectl` commands with the context:
  ```bash
  kubectl cluster-info --context kind-bharatml-stack
  kubectl get nodes --context kind-bharatml-stack
  ```

**Note:** For Predator setup on Kubernetes, see the [Predator Local Setup Guide](PREDATOR_SETUP.md) which includes cluster creation, node labeling, ArgoCD installation, and all required configurations.

### Predator and ArgoCD Setup

**Predator** is a service deployment and management system that runs on Kubernetes and uses ArgoCD for GitOps-based deployments. It is **not** a Docker Compose service and requires a comprehensive setup process.

**For complete Predator and ArgoCD setup instructions, see the [Predator Local Setup Guide](PREDATOR_SETUP.md).**

The PREDATOR_SETUP.md guide includes:
- Automated setup script for quick installation
- Step-by-step manual setup instructions
- Creating and configuring a Kubernetes cluster (kind, minikube, or Docker Desktop)
- Installing and configuring ArgoCD
- Setting up GitHub App for repository access
- Installing required CRDs (Flagger, KEDA) and PriorityClass
- Configuring ArgoCD repository connections
- Setting up automated application onboarding
- Complete Predator deployment workflow

The setup script handles all the complexity and provides an easy one-command installation. Follow the guide for detailed instructions.

### Connecting Inferflow to Kubernetes-Hosted Predator

Once you have Predator deployed in Kubernetes (via ArgoCD), you can connect Inferflow (running in Docker) to Predator for model inference. This setup enables Inferflow to call ML models hosted in your Kubernetes cluster.

#### Prerequisites

1. **Kubernetes cluster running** with Predator deployed (follow [PREDATOR_SETUP.md](PREDATOR_SETUP.md))
2. **ArgoCD installed** and accessible
3. **Predator service deployed** via ArgoCD with `app_name: predator` in your namespace (e.g., `prd-predator`)

#### Why Extra Hosts Configuration?

Inferflow runs in a Docker container, while Predator runs in Kubernetes. By default, Docker containers cannot reach services running on the host machine (localhost). The `extra_hosts` configuration in `docker-compose.yml` creates a special network route that maps a hostname to `host-gateway`, which is Docker's special DNS name that resolves to the host machine's IP address.

This is why you'll see this configuration in the Inferflow service:

```yaml
inferflow:
  extra_hosts:
    - "predator.prd.meesho.int:host-gateway"
```

This allows the Inferflow container to reach Predator services running on your host machine (via port-forward).

#### Step-by-Step Integration

**1. Deploy Predator in Kubernetes**

Follow the complete [Predator Setup Guide](PREDATOR_SETUP.md) to deploy Predator. Ensure your Predator deployment has:
- **App Name**: `predator` (or your chosen name matching the namespace)
- **Namespace**: `prd-predator` (or your environment-specific namespace)
- **Service exposed**: The Predator service should be accessible via Kubernetes service

**2. Port-Forward Predator Service to Host**

Predator services in Kubernetes are not directly accessible from Docker containers. You need to create a port-forward tunnel from your host machine to the Kubernetes service:

```bash
# Port-forward Predator service to localhost:8090
# Format: kubectl -n <namespace> port-forward svc/<service-name> <local-port>:<service-port>
kubectl -n prd-predator port-forward svc/prd-predator 8090:80 &
```

**Important Notes:**
- Port `8090` is the local port on your host machine
- Port `80` is the Kubernetes service port (not the target pod port 8001)
- Keep this terminal session running or run it in the background with `&`
- The port-forward must be active for Inferflow to reach Predator

**3. Configure Model Endpoint in etcd**

In your Inferflow model configuration (stored in etcd), set the endpoint to match the `extra_hosts` configuration:

```json
{
  "model_end_point": "predator.prd.meesho.int:8090"
}
```

**Key Points:**
- Use the hostname from `extra_hosts` configuration: `predator.prd.meesho.int`
- Use port `8090` (the local port from port-forward command)
- Inferflow will resolve this hostname to the host gateway and connect via port-forward

**4. Update Inferflow Deadline (if needed)**

If you experience `DeadlineExceeded` errors, increase the timeout in `workspace/docker-compose.yml`:

```yaml
inferflow:
  environment:
    - EXTERNAL_SERVICE_PREDATOR_DEADLINE=5000  # Increase from 200ms to 5000ms
```

Port-forwarding to Kubernetes adds latency, so higher timeouts are recommended.

**5. Configure Horizon with Deployable ID**

After deploying Predator through Horizon's deployment workflow, you'll receive a `deployable_id`. This ID is stored in the `service_deployable_configs` table in MySQL.

**To configure Horizon for testing:**

a. **Find the Deployable ID:**
   ```bash
   # Connect to MySQL
   docker exec -it mysql mysql -uroot -proot testdb
   
   # Query for your Predator deployable
   SELECT id, service_name, app_name, environment 
   FROM service_deployable_configs 
   WHERE service_name = 'predator' 
   ORDER BY id DESC LIMIT 5;
   ```

b. **Copy the `id` value** from the query result

c. **Update Horizon's environment variable** in `workspace/docker-compose.yml`:
   ```yaml
   horizon:
     environment:
       - TEST_DEPLOYABLE_ID=<your-deployable-id>
       - TEST_GPU_DEPLOYABLE_ID=<your-deployable-id>  # If using GPU
   ```

d. **Restart Horizon:**
   ```bash
   cd workspace && docker-compose restart horizon
   ```

**6. Verify the Connection**

Test the connectivity from Inferflow to Predator:

```bash
# Check if port-forward is active
netstat -an | grep 8090

# Test from within Docker network
docker exec inferflow-healthcheck nc -zv 172.18.0.1 8090

# Check Inferflow logs for connection status
docker logs inferflow 2>&1 | grep -i "predator\|deadline" | tail -20
```

**7. Monitor for Issues**

Common issues and their solutions:

| Issue | Solution |
|-------|----------|
| `DeadlineExceeded` errors | Increase `EXTERNAL_SERVICE_PREDATOR_DEADLINE` to 5000+ ms |
| `connection refused` | Verify port-forward is running: `ps aux \| grep port-forward` |
| `Invalid auth token` | Check Predator authentication configuration |
| DNS resolution fails | Verify `extra_hosts` in docker-compose.yml |

**8. Complete Configuration Example**

Your final `workspace/docker-compose.yml` should have:

```yaml
inferflow:
  extra_hosts:
    - "predator.prd.meesho.int:host-gateway"
  environment:
    - EXTERNAL_SERVICE_PREDATOR_PORT=8090
    - EXTERNAL_SERVICE_PREDATOR_GRPC_PLAIN_TEXT=true
    - EXTERNAL_SERVICE_PREDATOR_CALLER_ID=inferflow
    - EXTERNAL_SERVICE_PREDATOR_CALLER_TOKEN=inferflow
    - EXTERNAL_SERVICE_PREDATOR_DEADLINE=5000  # 5 seconds

horizon:
  environment:
    - TEST_DEPLOYABLE_ID=<your-deployable-id>
    - TEST_GPU_DEPLOYABLE_ID=<your-deployable-id>
```

### Database Access

- **MySQL**:
  - Host: localhost
  - Port: 3306
  - Username: root
  - Password: root
  - Database: testdb

- **ScyllaDB**:
  - Host: localhost
  - Port: 9042
  - Keyspace: onfs

- **Redis**:
  - Host: localhost
  - Port: 6379

- **etcd**:
  - Endpoint: http://localhost:2379
  - Workbench: http://localhost:8081

- **Kafka**:
  - Bootstrap Servers: localhost:9092
  - Kafka UI: http://localhost:8084
  - Default Topic: online-feature-store.feature_ingestion

## Managing Services

### Viewing Logs

```bash
# View logs for all services
cd workspace && docker-compose logs -f

# View logs for specific services
cd workspace && docker-compose logs -f horizon
cd workspace && docker-compose logs -f trufflebox-ui
cd workspace && docker-compose logs -f onfs-api-server
cd workspace && docker-compose logs -f onfs-consumer
cd workspace && docker-compose logs -f inferflow
cd workspace && docker-compose logs -f kafka

# View logs for multiple services
cd workspace && docker-compose logs -f onfs-api-server onfs-consumer inferflow
```

### Service Management

```bash
# Restart a specific service
cd workspace && docker-compose restart horizon

# Stop all services
cd workspace && docker-compose down

# Start services again
cd workspace && docker-compose up -d

# Check service status
cd workspace && docker-compose ps
```

## Troubleshooting

### Common Issues

1. **Port conflicts**: The following ports must be available:
   
   | Port | Service |
   |------|---------|
   | 3000 | TruffleBox UI |
   | 8081 | etcd Workbench |
   | 8082 | Horizon API |
   | 8083 | Numerix |
   | 8084 | Kafka UI |
   | 8085 | Inferflow |
   | 8089 | ONFS gRPC API |
   | 8090 | ONFS Consumer |
   | 9092 | Kafka |
   | 9042 | ScyllaDB |
   | 3306 | MySQL |
   | 6379 | Redis |
   | 2379 | etcd |
   
   If any ports are in use, stop conflicting services or modify port mappings in `docker-compose.yml`.

2. **Docker network issues**: If containers can't communicate, try recreating:
   ```bash
   docker network rm onfs-network
   docker network create onfs-network
   ```

3. **Service health checks failing**: Check if all infrastructure services (databases) are running:
   ```bash
   cd workspace && docker-compose ps
   ```

4. **Image pull issues**: Ensure you have access to GitHub Container Registry:
   ```bash
   docker login ghcr.io
   ```

5. **Kafka consumer not receiving messages**: 
   ```bash
   # Check if Kafka topic exists
   docker exec -it broker kafka-topics.sh --bootstrap-server localhost:9092 --list
   
   # Check consumer group status
   docker exec -it broker kafka-consumer-groups.sh --bootstrap-server localhost:9092 --group onfs-consumer-group --describe
   
   # Verify Kafka UI
   open http://localhost:8084
   ```

6. **Consumer lag issues**: Monitor consumer performance via Kafka UI (http://localhost:8084) to identify bottlenecks

7. [How to use Etcd Workbench ?](https://github.com/tzfun/etcd-workbench/blob/master/README.md)

### Service Dependencies

Services start in the following order:
1. Infrastructure services (ScyllaDB, MySQL, Redis, etcd, Kafka)
2. Infrastructure init services (kafka-init, db-init) - only started if containers don't exist (preserves modifications)
3. Application services (can run independently):
   - **ONFS API Server** - depends on databases (ScyllaDB, MySQL, Redis, etcd)
   - **ONFS Consumer** - depends on Kafka and databases (independent of API Server)
   - **Horizon** - depends on databases and ScyllaDB
   - **Numerix** - depends on etcd
   - **Inferflow** - depends on etcd, ONFS API Server, and Numerix
4. Trufflebox UI (depends on Horizon)

**Key Points:**
- ONFS Consumer can run without ONFS API Server
- Services can be started individually based on use case
- All services depend on their respective infrastructure components

If a service fails to start, check its dependencies are healthy first.

## Development

The workspace directory contains all runtime configuration:

- `workspace/docker-compose.yml` - Complete service orchestration
- `workspace/check_db_and_init.sh` - Database initialization script

You can modify environment variables in the docker-compose.yml file and restart services.

## Contributing

We welcome contributions from the community! Please see our [Contributing Guide](CONTRIBUTING.md) for details on how to get started.

## Community & Support

- 💬 **Discord**: Join our [community chat](https://discord.gg/XkT7XsV2AU)
- 🐛 **Issues**: Report bugs and request features on [GitHub Issues](https://github.com/Meesho/BharatMLStack/issues)
- 📧 **Email**: Contact us at [ml-oss@meesho.com](mailto:ml-oss@meesho.com )

## License

BharatMLStack is open-source software licensed under the [BharatMLStack Business Source License 1.1](LICENSE.md).

---

<div align="center">
  <strong>Built with ❤️ for the ML community from Meesho</strong>
</div>
<div align="center">
  <strong>If you find this useful, ⭐️ the repo — your support means the world to us!</strong>
</div>