![Build Status](https://github.com/Meesho/BharatMLStack/actions/workflows/trufflebox-ui.yml/badge.svg)
![Static Badge](https://img.shields.io/badge/release-v1.1.0-blue?style=flat)
[![Discord](https://img.shields.io/badge/Discord-Join%20Chat-7289da?style=flat&logo=discord&logoColor=white)](https://discord.gg/XkT7XsV2AU)

# TruffleBox UI

TruffleBox UI is the comprehensive web-based management interface for BharatMLStack's ML infrastructure. It provides an intuitive dashboard for managing feature stores, model inference, embedding platforms, compute configurations, and administering users across your ML ecosystem.

## 🌟 Overview

TruffleBox UI serves as the primary frontend interface for the BharatMLStack ecosystem, offering:

- **Online Feature Store** - Feature discovery, cataloging, and management
- **InferFlow (Model Proxy)** - Model proxy configuration and deployment management
- **Numerix** - Compute configuration management with infix expression support
- **Predator** - Model registry and deployment management
- **Embedding Platform** - Vector database management, variant deployment, and embedding operations
- **Approval Workflows** - Streamlined approval processes across all services
- **User Management** - Role-based access control and user administration
- **Real-time Monitoring** - Monitor service health and performance

## 🏗️ Architecture

Built with modern web technologies:

- **Frontend**: React 18.3+ with Material-UI (v6) and Bootstrap styling
- **Routing**: React Router v6 for single-page application navigation
- **State Management**: React Context API and React Redux
- **Authentication**: JWT-based authentication with protected routes and role-based permissions
- **Backend Integration**: RESTful API integration with Horizon, Skye, and Model Inference services
- **UI Components**: Material-UI components with custom theming
- **Expression Editing**: MathQuill integration for infix expression editing
- **JSON Visualization**: Advanced JSON viewer and diff tools
- **Deployment**: Dockerized with Nginx for production serving

## 🚀 Quick Start

### Prerequisites

- Node.js 16+ and yarn
- Docker and Docker Compose (for containerized deployment)
- Access to BharatMLStack backend services (Horizon, Skye, Model Inference)

### Development Setup

1. **Clone and Navigate**
   ```bash
   cd trufflebox-ui
   ```

2. **Install Dependencies**
   ```bash
   yarn install
   ```

3. **Configure Environment**
   ```bash
   cp env.example .env
   # Edit .env with your backend service URLs and feature flags
   ```

4. **Start Development Server**
   ```bash
   yarn start
   ```

   Open [http://localhost:3000](http://localhost:3000) to view the application.

### Production Deployment

#### Using Docker

```bash
# Build the Docker image
docker build -t trufflebox-ui .

# Run with environment variables
docker run -p 80:80 \
  -e REACT_APP_HORIZON_BASE_URL=http://your-horizon-url:8082 \
  -e REACT_APP_HORIZON_PROD_BASE_URL=http://your-horizon-prod-url:8085 \
  -e REACT_APP_SKYE_BASE_URL=http://your-skye-url:8083 \
  -e REACT_APP_MODEL_INFERENCE_BASE_URL=http://your-model-inference-url:8084 \
  trufflebox-ui
```

#### Using Docker Compose

```bash
docker-compose up -d
```

## 📱 Features

### Online Feature Store

The original feature store management capabilities:

- **Entity Explorer** - Browse available entities in your feature store
- **Feature Group Navigation** - Explore feature groups within entities
- **Feature Catalog** - Detailed view of individual features with metadata
- **Client Discovery** - Identify applications consuming features
- **Store Registry** - Register and configure new feature stores
- **Job Registry** - Manage feature engineering jobs and pipelines
- **Entity Registry** - Define and register business entities
- **Feature Group Registry** - Create and manage feature groups
- **Feature Addition** - Add new features to existing groups
- **Multi-level Approvals** - Configurable approval chains for stores, jobs, entities, feature groups, and features

### InferFlow (Model Proxy)

Model proxy configuration and management system:

- **Deployable Registry** - Register and manage deployable model proxy instances
- **Model Proxy Config Registry** - Create and manage model proxy configurations
- **Config Management** - Onboard, edit, clone, and promote model proxy configurations
- **Config Testing** - Test model proxy configurations with custom requests
- **Ranker Configuration** - Configure multiple rankers with batch processing, calibration, and deadlines
- **Re-ranker Support** - Multi-stage ranking pipeline configuration
- **Response Configuration** - Configure response schemas, logging, and feature inclusion
- **Config Mapping** - Map configurations to deployable instances
- **Approval Workflows** - Review and approve model proxy configurations before deployment
- **Production Promotion** - Promote configurations to production with credential verification

### Numerix

Compute configuration management with mathematical expression support:

- **Config Discovery & Registry** - Browse and manage compute configurations
- **Infix Expression Editor** - Visual editor for mathematical expressions using MathQuill
- **Expression Validation** - Real-time validation of infix expressions
- **Postfix Conversion** - Automatic conversion from infix to postfix notation
- **Supported Functions** - Built-in support for mathematical functions:
  - Single argument: `exp(x)`, `log(x)`, `abs(x)`, `norm_min_max(x)`, `percentile_rank(x)`, `norm_percentile_0_99(x)`, `norm_percentile_5_95(x)`
  - Two arguments: `min(x, y)`, `max(x, y)`
- **Config Testing** - Test compute configurations with sample data
- **Production Promotion** - Promote configurations to production environment
- **Approval Workflows** - Review and approve compute configurations
- **Expression Guidelines** - Built-in help and guidelines for expression syntax

### Predator

Model registry and deployment management:

- **Model Registry** - Upload, register, and manage ML models
- **Model Discovery** - Browse and discover available models
- **Deployable Registry** - Manage deployable instances for model serving
- **Model Upload** - Upload models with metadata including:
  - GCS path configuration
  - Input/output specifications
  - Feature type definitions
  - Partial upload support
- **Model Testing** - Test models with custom inputs
- **Model Metadata** - Comprehensive model metadata management
- **Deployment Configuration** - Configure deployment strategies, resource limits, and scaling
- **Approval Workflows** - Review and approve model registrations

### Embedding Platform

Vector database and embedding management platform:

- **Deployment Operations** - Comprehensive deployment management:
  - **Dashboard** - Monitor deployment status and health
  - **Cluster Management** - Create and manage Qdrant clusters
  - **Variant Promotion** - Promote variants with canary/blue-green strategies
  - **Variant Onboarding** - Onboard variants to vector databases
- **Store Management** - Register and manage embedding stores
- **Entity Management** - Define and manage entities for embeddings
- **Model Management** - Register and manage embedding models
- **Variant Management** - Create and manage model variants
- **Filter Management** - Configure and manage embedding filters
- **Job Frequency Management** - Schedule and manage embedding job frequencies
- **Discovery Interfaces** - Browse stores, entities, models, variants, filters, and job frequencies
- **Approval Workflows** - Multi-level approvals for all embedding platform components
- **Variant Database Onboarding** - Onboard variants to vector databases with configuration

### Approval Workflows

Unified approval system across all services:

- **Multi-level Approvals** - Configurable approval chains for different components
- **Status Tracking** - Track approval status with detailed history
- **Bulk Operations** - Approve or reject multiple items at once
- **Filtering & Search** - Advanced filtering by status, requester, date, and more
- **Approval History** - View complete approval history with timestamps
- **Production Credentials** - Secure credential verification for production promotions
- **Role-based Access** - Admin-only access to approval interfaces

### User Administration

- **Role-based Access Control** - Manage user permissions and roles
- **Permission System** - Fine-grained permissions per service and screen type
- **User Management** - Add, modify, and deactivate user accounts
- **Authentication** - Secure login and registration system
- **Session Management** - Automatic token refresh and logout

## 🔧 Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `REACT_APP_HORIZON_BASE_URL` | Horizon backend service URL (dev/staging) | `http://localhost:8082` |
| `REACT_APP_HORIZON_PROD_BASE_URL` | Horizon backend service URL (production) | `http://localhost:8085` |
| `REACT_APP_SKYE_BASE_URL` | Skye service URL | `http://localhost:8083` |
| `REACT_APP_MODEL_INFERENCE_BASE_URL` | Model Inference service URL | `http://localhost:8084` |
| `REACT_APP_ENVIRONMENT` | Application environment (production/staging/development) | `production` |
| `REACT_APP_ONLINE_FEATURE_STORE_ENABLED` | Enable/disable Online Feature Store | `true` |
| `REACT_APP_INFERFLOW_ENABLED` | Enable/disable InferFlow | `true` |
| `REACT_APP_NUMERIX_ENABLED` | Enable/disable Numerix | `true` |
| `REACT_APP_PREDATOR_ENABLED` | Enable/disable Predator | `true` |
| `REACT_APP_EMBEDDING_PLATFORM_ENABLED` | Enable/disable Embedding Platform | `false` |
| `PUBLIC_USER_BASE_URL` | Base path for React Router (subpath deployment) | `/` |

### Service Feature Flags

All services can be enabled or disabled via environment variables, allowing you to customize the UI based on your deployment needs. Services are controlled by feature flags:

- `REACT_APP_ONLINE_FEATURE_STORE_ENABLED` - Online Feature Store
- `REACT_APP_INFERFLOW_ENABLED` - InferFlow (Model Proxy)
- `REACT_APP_NUMERIX_ENABLED` - Numerix
- `REACT_APP_PREDATOR_ENABLED` - Predator
- `REACT_APP_EMBEDDING_PLATFORM_ENABLED` - Embedding Platform

### Runtime Configuration

The application generates runtime configuration in `env.js` to support dynamic environment variable injection in containerized deployments.

## 🛠️ Development

### Available Scripts

| Command | Description |
|---------|-------------|
| `yarn start` | Start development server with hot reload |
| `yarn test` | Run test suite |
| `yarn run build` | Build optimized production bundle |
| `yarn run eject` | Eject from Create React App (⚠️ irreversible) |
| `yarn lint` | Run linting (currently placeholder) |

### Project Structure

```
src/
├── pages/
│   ├── Auth/                          # Authentication components
│   │   ├── AuthContext.jsx            # Auth context and hooks
│   │   ├── Login.jsx                  # Login page
│   │   ├── Register.jsx              # Registration page
│   │   ├── ProtectedRoute.jsx        # Route protection
│   │   └── Unauthorized.jsx          # Unauthorized access page
│   ├── Header/                        # Navigation and header
│   ├── Layout/                        # Layout components
│   ├── OnlineFeatureStore/            # Feature store functionality
│   │   ├── components/
│   │   │   ├── Discovery/             # Feature discovery components
│   │   │   ├── FeatureRegistry/      # Feature registration
│   │   │   └── FeatureApproval/       # Approval workflows
│   │   └── common/                    # Shared components
│   ├── InferFlow/                     # Model Proxy management
│   │   ├── Approval/                  # Config approval workflows
│   │   └── DiscoveryRegistry/         # Config and deployable registry
│   │       ├── Deployable/            # Deployable management
│   │       └── MPConfigRegistry/      # Model proxy config management
│   ├── Numerix/                       # Compute configuration
│   │   ├── Approval/                  # Config approval
│   │   ├── DiscoveryRegistry/         # Config discovery and registry
│   │   └── shared/                    # Shared components and tables
│   ├── Predator/                      # Model registry
│   │   ├── components/
│   │   │   ├── Approval/              # Model approval workflows
│   │   │   └── Registry/              # Model and deployable registry
│   ├── EmbeddingPlatform/             # Embedding platform
│   │   └── components/
│   │       ├── DeploymentOperations/  # Deployment management
│   │       ├── EntityManagement/      # Entity management
│   │       ├── ModelManagement/       # Model management
│   │       ├── VariantManagement/     # Variant management
│   │       ├── FilterManagement/      # Filter management
│   │       ├── JobFrequencyManagement/# Job frequency management
│   │       └── StoreManagement/       # Store management
│   └── UserManagement/                # User administration
├── components/                        # Reusable UI components
│   ├── ExpressionViewModal.jsx        # Expression viewer modal
│   ├── InfixExpressionEditor.jsx      # MathQuill-based expression editor
│   ├── JsonDiffView.jsx               # JSON diff visualization
│   └── JsonViewer.jsx                 # JSON viewer component
├── common/                            # Common utilities and components
│   ├── ErrorBoundary.jsx              # Error boundary component
│   ├── ProductionCredentialModal.jsx  # Production credential modal
│   └── PromoteWithProdCredentials.jsx # Promotion with credentials
├── constants/                         # Application constants
│   ├── databaseTypes.js               # Database type definitions
│   ├── dataTypes.js                   # Data type definitions
│   ├── permissions.js                 # Permission constants
│   └── serviceMapping.js              # Service and permission mappings
├── hooks/                             # Custom React hooks
│   └── useFormatDate.jsx              # Date formatting hook
├── services/                          # Service integrations
│   ├── embeddingPlatform/             # Embedding platform API
│   └── httpInterceptor.js             # HTTP request interceptor
├── utils/                             # Utility functions
│   └── infixToPostfix.js              # Expression conversion utilities
└── config.js                          # Configuration management
```

### Key Components

#### Online Feature Store
- **FeatureDiscovery** - Main feature exploration interface
- **EntityDiscovery** - Entity browsing and selection
- **FeatureGroupDiscovery** - Feature group navigation
- **FeatureList** - Detailed feature listing and metadata
- **StoreRegistry** - Feature store registration
- **EntityRegistry** - Entity registration and management

#### InferFlow
- **DeployableModelProxyRegistry** - Deployable instance management
- **ModelProxyConfigRegistry** - Model proxy configuration management
- **OnboardMPConfigModal** - Onboard new configurations
- **EditMPConfigModal** - Edit existing configurations
- **CloneMPConfigModal** - Clone configurations
- **PromoteMPConfigModal** - Promote to production
- **MPConfigTestingModal** - Test configurations
- **MPConfigForm** - Comprehensive configuration form

#### Numerix
- **NumerixConfigDiscoveryRegistry** - Config discovery and management
- **InfixExpressionEditor** - Visual expression editor
- **TestConfigModal** - Configuration testing interface
- **ConfigDetailsModal** - Configuration details viewer
- **NumerixConfigApproval** - Approval workflow interface

#### Predator
- **ModelRegistry** - Model registration and management
- **DeployableRegistry** - Deployable instance management
- **UploadModelModal** - Model upload interface
- **ModelTestingModal** - Model testing interface
- **ModelApproval** - Model approval workflow

#### Embedding Platform
- **DeploymentOperations** - Deployment management dashboard
- **DeploymentRegistry** - Deployment registry
- **DeploymentDashboard** - Deployment monitoring dashboard
- **DeploymentApproval** - Deployment approval workflow
- **EntityRegistry** - Entity management
- **ModelRegistry** - Model management
- **VariantRegistry** - Variant management
- **FilterRegistry** - Filter management
- **JobFrequencyRegistry** - Job frequency management

### Shared Components

- **GenericTable** - Reusable table component with pagination and search
- **GenericNumerixTable** - Specialized table for Numerix configs
- **GenericMPConfigRegistryTable** - Specialized table for MP configs
- **GenericDeployableTable** - Reusable deployable table component
- **JsonViewer** - Advanced JSON visualization
- **JsonDiffView** - Side-by-side JSON diff viewer
- **ExpressionViewModal** - Mathematical expression viewer
- **InfixExpressionEditor** - MathQuill-based expression editor

## 🔐 Authentication & Authorization

TruffleBox UI implements comprehensive authentication and authorization:

- **JWT-based Authentication** - Secure token-based authentication
- **Protected Routes** - Secure access to authenticated features
- **Role-based Authorization** - Different access levels based on user roles (admin/user)
- **Permission System** - Fine-grained permissions per service and screen type:
  - View permissions
  - Create/Upload permissions
  - Edit permissions
  - Delete permissions
  - Approval permissions
  - Partial upload permissions
- **Session Management** - Automatic token refresh and logout
- **Registration Flow** - New user onboarding process
- **Unauthorized Access Handling** - Graceful handling of unauthorized access attempts

### Permission Model

The application uses a service-based permission model where:
- Each service (InferFlow, Numerix, Predator, Embedding Platform) has its own permission namespace
- Screen types define different areas within a service (e.g., `mp-config`, `model`, `deployable`)
- Actions define what operations can be performed (e.g., `VIEW`, `CREATE`, `EDIT`, `APPROVE`)
- Permissions are checked at the route and component level

## 🚢 Deployment

### Container Configuration

The application uses a multi-stage Docker build:

1. **Build Stage** - Compiles React application with Node.js
2. **Runtime Stage** - Serves static files with Nginx Alpine

### Health Checks

Health check endpoint available at `/health` for monitoring deployment status.

### Release Management

Version management through `VERSION` file and automated release scripts (`release.sh`). Current version: **v1.1.0**

### Nginx Configuration

The application includes a custom Nginx configuration for optimal production serving with:
- Gzip compression
- Static file caching
- SPA routing support
- Security headers

## 🔗 Integration

TruffleBox UI integrates seamlessly with BharatMLStack components:

- **Horizon** - Primary backend service for all feature store and ML infrastructure management
- **Skye** - Advanced analytics and monitoring
- **Model Inference** - Real-time model serving integration
- **ONFS CLI** - Command-line tool compatibility

### API Integration

The UI integrates with multiple backend services:

- **Horizon API** - Main API for all services (Feature Store, InferFlow, Numerix, Predator, Embedding Platform)
- **Skye API** - Analytics and monitoring data
- **Model Inference API** - Model serving and inference endpoints

## 🎨 UI/UX Features

- **Material-UI v6** - Modern, accessible component library
- **Responsive Design** - Mobile-friendly interface
- **Dark Theme Support** - Custom theming with brand colors
- **Loading States** - Skeleton loaders and progress indicators
- **Error Handling** - Comprehensive error boundaries and user-friendly error messages
- **Toast Notifications** - User feedback for actions
- **Modal Dialogs** - Rich modal interfaces for complex operations
- **Form Validation** - Real-time form validation with error messages
- **Search & Filtering** - Advanced search and filtering across all tables
- **Pagination** - Efficient pagination for large datasets
- **JSON Visualization** - Advanced JSON viewing and diff capabilities
- **Expression Editing** - Visual mathematical expression editor with MathQuill

## 📚 Learn More

- [BharatMLStack Documentation](../README.md)
- [Feature Store Architecture](../online-feature-store/docs/)
- [API Documentation](../online-feature-store/docs/api/)
- [Deployment Guide](../quick-start/)

## Contributing

We welcome contributions from the community! Please see our [Contributing Guide](CONTRIBUTING.md) for details on how to get started.

## Community & Support

- 💬 **Discord**: Join our [community chat](https://discord.gg/XkT7XsV2AU)
- 🐛 **Issues**: Report bugs and request features on [GitHub Issues](https://github.com/Meesho/BharatMLStack/issues)
- 📧 **Email**: Contact us at [ml-oss@meesho.com](mailto:ml-oss@meesho.com)

## License

BharatMLStack is open-source software licensed under the [BharatMLStack Business Source License 1.1](LICENSE.md).

---

<div align="center">
  <strong>Built with ❤️ for the ML community from Meesho</strong>
</div>
<div align="center">
  <strong>If you find this useful, ⭐️ the repo — your support means the world to us!</strong>
</div>


