# Welcome to Orion Feature Store Docs

Orion is a high-performance, scalable, and production-grade feature store built for modern machine learning systems. It supports both **real-time** and **batch** workflows, and is designed with a strong emphasis on **developer experience**, **system observability**, and **low-latency feature retrieval**.

![Orion Value Prop](../assets/value-prop.png)

---

## 📘 Documentation Index

### 🏗 Architecture
- [Orion Feature Store – Production Architecture](./architecture.md)

### ⚙️ Core Components
- [Feature Generation](./architecture.md#feature-generation)
- [Kafka Streaming Ingestion](./architecture.md#kafka-layer-streaming-ingestion)
- [Horizon Control Plane](./architecture.md#horizon-configuration--control-plane)
- [Orion Consumer](./architecture.md#orion-consumer)
- [Orion gRPC API Server](./architecture.md#5-orion-grpc-api-server)
- [Monitoring & Observability](./architecture.md#7-monitoring--observability)
- [SDKs](./architecture.md#8-sdks)

### 🧱 Low-Level Design
- [Permanent Storage Data Block (PSDB)](./psdb-design.md)
- [Cache Storage Data Block (CSDB)](./csdb-design.md)
- [DeserializedPSDB Structure](./ddb-design.md)
- [Key-Schema Isolation & etcd Mapping](./schema.md)

