![Build Status](https://github.com/Meesho/BharatMLStack/actions/workflows/trufflebox-ui.yml/badge.svg)
![Static Badge](https://img.shields.io/badge/release-v1.0.0-blue?style=flat)
[![Discord](https://img.shields.io/badge/Discord-Join%20Chat-7289da?style=flat&logo=discord&logoColor=white)](https://discord.gg/XkT7XsV2AU)

# TruffleBox UI

TruffleBox UI is the web-based management interface for BharatMLStack's Online Feature Store. It provides an intuitive dashboard for managing feature stores, discovering features, handling approval workflows, and administering users across your ML infrastructure.

## 🌟 Overview

TruffleBox UI serves as the primary frontend interface for the BharatMLStack ecosystem, offering:

- **Feature Discovery & Cataloging** - Browse and explore features across entities and feature groups
- **Feature Store Management** - Register and manage feature stores, jobs, and entities
- **Approval Workflows** - Streamlined approval processes for feature store components
- **User Management** - Role-based access control and user administration
- **Real-time Monitoring** - Monitor feature store health and performance

## 🏗️ Architecture

Built with modern web technologies:

- **Frontend**: React 18.3+ with Material-UI and Bootstrap styling
- **Routing**: React Router for single-page application navigation
- **Authentication**: JWT-based authentication with protected routes
- **Backend Integration**: RESTful API integration with Horizon, Skye, and Model Inference services
- **Deployment**: Dockerized with Nginx for production serving

## 🚀 Quick Start

### Prerequisites

- Node.js 16+ and npm
- Docker and Docker Compose (for containerized deployment)
- Access to BharatMLStack backend services (Horizon, Skye)

### Development Setup

1. **Clone and Navigate**
   ```bash
   cd trufflebox-ui
   ```

2. **Install Dependencies**
   ```bash
   npm install
   ```

3. **Configure Environment**
   ```bash
   cp env.example .env
   # Edit .env with your backend service URLs
   ```

4. **Start Development Server**
   ```bash
   npm start
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
  trufflebox-ui
```

#### Using Docker Compose

```bash
docker-compose up -d
```

## 📱 Features

### Feature Discovery
- **Entity Explorer** - Browse available entities in your feature store
- **Feature Group Navigation** - Explore feature groups within entities
- **Feature Catalog** - Detailed view of individual features with metadata
- **Client Discovery** - Identify applications consuming features

### Feature Management
- **Store Registry** - Register and configure new feature stores
- **Job Registry** - Manage feature engineering jobs and pipelines
- **Entity Registry** - Define and register business entities
- **Feature Group Registry** - Create and manage feature groups
- **Feature Addition** - Add new features to existing groups

### Approval Workflows
- **Multi-level Approvals** - Configurable approval chains for different components
- **Store Approvals** - Review and approve new feature stores
- **Job Approvals** - Validate feature engineering jobs before deployment
- **Feature Approvals** - Ensure quality and compliance of new features

### User Administration
- **Role-based Access Control** - Manage user permissions and roles
- **User Management** - Add, modify, and deactivate user accounts
- **Authentication** - Secure login and registration system

## 🔧 Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `REACT_APP_HORIZON_BASE_URL` | Horizon backend service URL | `http://localhost:8082` |
| `PUBLIC_USER_BASE_URL` | Base URL for public routes | `/` |

### Runtime Configuration

The application generates runtime configuration in `env.js` to support dynamic environment variable injection in containerized deployments.

## 🛠️ Development

### Available Scripts

| Command | Description |
|---------|-------------|
| `npm start` | Start development server with hot reload |
| `npm test` | Run test suite |
| `npm run build` | Build optimized production bundle |
| `npm run eject` | Eject from Create React App (⚠️ irreversible) |

### Project Structure

```
src/
├── pages/
│   ├── Auth/                    # Authentication components
│   ├── Header/                  # Navigation and header
│   ├── Layout/                  # Layout components
│   ├── OnlineFeatureStore/      # Feature store functionality
│   │   └── components/
│   │       ├── Discovery/       # Feature discovery components
│   │       ├── FeatureRegistry/ # Feature registration
│   │       └── FeatureApproval/ # Approval workflows
│   └── UserManagement/          # User administration
├── common/                      # Shared components and utilities
├── constants/                   # Application constants
└── config.js                    # Configuration management
```

### Key Components

- **FeatureDiscovery** - Main feature exploration interface
- **EntityDiscovery** - Entity browsing and selection
- **FeatureGroupDiscovery** - Feature group navigation
- **FeatureList** - Detailed feature listing and metadata
- **UserManagement** - Complete user administration panel

## 🔐 Authentication

TruffleBox UI implements JWT-based authentication with:

- **Protected Routes** - Secure access to authenticated features
- **Role-based Authorization** - Different access levels based on user roles
- **Session Management** - Automatic token refresh and logout
- **Registration Flow** - New user onboarding process

## 🚢 Deployment

### Container Configuration

The application uses a multi-stage Docker build:

1. **Build Stage** - Compiles React application with Node.js
2. **Runtime Stage** - Serves static files with Nginx Alpine

### Health Checks

Health check endpoint available at `/health` for monitoring deployment status.

### Release Management

Version management through `VERSION` file and automated release scripts (`release.sh`).

## 🔗 Integration

TruffleBox UI integrates seamlessly with BharatMLStack components:

- **Horizon** - Primary backend service for feature store management
- **Skye** - Advanced analytics and monitoring
- **Model Inference** - Real-time model serving integration
- **ONFS CLI** - Command-line tool compatibility

## 📚 Learn More

- [BharatMLStack Documentation](../README.md)
- [Feature Store Architecture](../online-feature-store/docs/)
- [API Documentation](../online-feature-store/docs/api/)
- [Deployment Guide](../quick-start/)

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is part of BharatMLStack and follows the same licensing terms.

---

<div align="center">
  <strong>TruffleBox UI - Your Gateway to Friendly MLOps</strong>
  <br/>
  Built with ❤️ for the BharatMLStack ecosystem
</div>
