import './App.css';
import FeatureDiscovery from './pages/OnlineFeatureStore/components/Discovery/FeatureDiscovery';
import StoreDiscovery from './pages/OnlineFeatureStore/components/Discovery/StoreDiscovery';
import JobDiscovery from './pages/OnlineFeatureStore/components/Discovery/JobDiscovery';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import StoreRegistry from './pages/OnlineFeatureStore/components/FeatureRegistry/StoreRegistry';
import JobRegistry from './pages/OnlineFeatureStore/components/FeatureRegistry/JobRegistry';
import EntityRegistry from './pages/OnlineFeatureStore/components/FeatureRegistry/EntityRegistry';
import FeatureGroupRegistry from './pages/OnlineFeatureStore/components/FeatureRegistry/FeatureGroupRegistry';
import FeatureAddition from './pages/OnlineFeatureStore/components/FeatureRegistry/FeatureAddition';
import StoreApproval from './pages/OnlineFeatureStore/components/FeatureApproval/StoreApproval';
import JobApproval from './pages/OnlineFeatureStore/components/FeatureApproval/JobApproval';
import EntityApproval from './pages/OnlineFeatureStore/components/FeatureApproval/EntityApproval';
import FeatureGroupApproval from './pages/OnlineFeatureStore/components/FeatureApproval/FeatureGroupApproval';
import FeatureAdditionApproval from './pages/OnlineFeatureStore/components/FeatureApproval/FeatureAdditionApproval';
import NumerixConfigDiscoveryRegistry from './pages/Numerix/DiscoveryRegistry/NumerixConfigDiscoveryRegistry';
import NumerixConfigApproval from './pages/Numerix/Approval/NumerixConfigApproval';
import UserManagement from './pages/UserManagement';
import ErrorBoundary from './common/ErrorBoundary';
import ClientDiscovery from './pages/OnlineFeatureStore/components/Discovery/ClientDiscovery';
import DeployableInferflowRegistry from './pages/InferFlow/DiscoveryRegistry/Deployable/DeployableInferflowRegistry';
import InferflowConfigApproval from './pages/InferFlow/Approval/InferflowConfigApproval';
import InferflowConfigRegistry from './pages/InferFlow/DiscoveryRegistry/InferflowRegistry/InferflowConfigRegistry';
import DeployableRegistry from './pages/Predator/components/Registry/DeployableRegistry';
import ModelRegistry from './pages/Predator/components/Registry/ModelRegistry';
import ModelApproval from './pages/Predator/components/Approval/ModelApproval';
import { AuthProvider } from './pages/Auth/AuthContext';
import ProtectedRoute from './pages/Auth/ProtectedRoute';
import Login from './pages/Auth/Login';
import Register from './pages/Auth/Register';
import HealthCheck from './Health';
import Unauthorized from './pages/Auth/Unauthorized';
import { 
  isOnlineFeatureStoreEnabled, 
  isInferFlowEnabled, 
  isNumerixEnabled, 
  isPredatorEnabled 
} from './config';

function App() {
  return (
    <ErrorBoundary fallbackMessage="Application encountered an unexpected error. Please refresh the page.">
      <Router basename={process.env.PUBLIC_USER_BASE_URL}>
        <AuthProvider>
          <Routes>
            {/* Public Routes */}
            <Route path="/login" element={<Login />} />
            <Route path="/register" element={<Register />} />
            <Route path="/unauthorized" element={<Unauthorized />} />

            {/* Online Feature Store Routes */}
            {isOnlineFeatureStoreEnabled() && (
              <>
                <Route
                  path="/feature-discovery"
                  element={
                    <ProtectedRoute>
                      <FeatureDiscovery />
                    </ProtectedRoute>
                  }
                />
                <Route
                  path="/store-discovery"
                  element={
                    <ProtectedRoute>
                      <StoreDiscovery />
                    </ProtectedRoute>
                  }
                />
                <Route
                  path="/job-discovery"
                  element={
                    <ProtectedRoute>
                      <JobDiscovery />
                    </ProtectedRoute>
                  }
                />
                <Route
                  path="/client-discovery"
                  element={
                    <ProtectedRoute>
                      <ClientDiscovery />
                    </ProtectedRoute>
                  }
                />
                <Route
                  path="/feature-registry/store"
                  element={
                    <ProtectedRoute>
                      <StoreRegistry />
                    </ProtectedRoute>
                  }
                />
                <Route
                  path="/feature-registry/job"
                  element={
                    <ProtectedRoute>
                      <JobRegistry />
                    </ProtectedRoute>
                  }
                />
                <Route
                  path="/feature-registry/entity"
                  element={
                    <ProtectedRoute>
                      <EntityRegistry />
                    </ProtectedRoute>
                  }
                />
                <Route
                  path="/feature-registry/feature-group"
                  element={
                    <ProtectedRoute>
                      <FeatureGroupRegistry />
                    </ProtectedRoute>
                  }
                />
                <Route
                  path="/feature-registry/feature"
                  element={
                    <ProtectedRoute>
                      <FeatureAddition />
                    </ProtectedRoute>
                  }
                />
                <Route
                  path="/feature-approval/store"
                  element={
                    <ProtectedRoute>
                      <StoreApproval />
                    </ProtectedRoute>
                  }
                />
                <Route
                  path="/feature-approval/job"
                  element={
                    <ProtectedRoute>
                      <JobApproval />
                    </ProtectedRoute>
                  }
                />
                <Route
                  path="/feature-approval/entity"
                  element={
                    <ProtectedRoute>
                      <EntityApproval />
                    </ProtectedRoute>
                  }
                />
                <Route
                  path="/feature-approval/feature-group"
                  element={
                    <ProtectedRoute>
                      <FeatureGroupApproval />
                    </ProtectedRoute>
                  }
                />
                <Route
                  path="/feature-approval/features"
                  element={
                    <ProtectedRoute>
                      <FeatureAdditionApproval />
                    </ProtectedRoute>
                  }
                />
              </>
            )}
            <Route
              path="/user-management"
              element={
                <ProtectedRoute>
                  <UserManagement />
                </ProtectedRoute>
              }
            />

            {/* Numerix Routes */}
            {isNumerixEnabled() && (
              <>
                <Route
                  path="/numerix/config"
                  element={
                    <ProtectedRoute service="numerix" screenType="numerix-config">
                      <NumerixConfigDiscoveryRegistry />
                    </ProtectedRoute>
                  }
                />
                <Route
                  path="/numerix/config-approval"
                  element={
                    <ProtectedRoute service="numerix" screenType="numerix-config-approval">
                      <NumerixConfigApproval />
                    </ProtectedRoute>
                  }
                />
              </>
            )}

            {/* Predator Routes */}
            {isPredatorEnabled() && (
              <>
                <Route
                  path="/predator/discovery-registry/deployable"
                  element={
                    <ProtectedRoute service="predator" screenType="deployable">
                      <DeployableRegistry />
                    </ProtectedRoute>
                  }
                />
                <Route
                  path="/predator/discovery-registry/model"
                  element={
                    <ProtectedRoute service="predator" screenType="model">
                      <ModelRegistry />
                    </ProtectedRoute>
                  }
                />
                <Route
                  path="/predator/approval/model"
                  element={
                    <ProtectedRoute service="predator" screenType="model-approval">
                      <ModelApproval />
                    </ProtectedRoute>
                  }
                />
              </>
            )}

            {/* InferFlow Routes */}
            {isInferFlowEnabled() && (
              <>
                <Route
                  path="/inferflow/deployable"
                  element={
                    <ProtectedRoute service="inferflow" screenType="deployable">
                      <DeployableInferflowRegistry />
                    </ProtectedRoute>
                  }
                />
                <Route
                  path="/inferflow/config-registry"
                  element={
                    <ProtectedRoute service="inferflow" screenType="inferflow-config">
                      <InferflowConfigRegistry />
                    </ProtectedRoute>
                  }
                />
                <Route
                  path="/inferflow/config-approval"
                  element={
                    <ProtectedRoute service="inferflow" screenType="inferflow-config-approval">
                      <InferflowConfigApproval />
                    </ProtectedRoute>
                  }
                />
              </>
            )}

            {/* Redirect to Homepage(Feature Discovery) */}
            <Route path="*" element={<Navigate to="/feature-discovery" replace/>} />
            <Route path="/health" element={<HealthCheck />} />
          </Routes>
        </AuthProvider>
      </Router>
    </ErrorBoundary>
  );
}

export default App;
