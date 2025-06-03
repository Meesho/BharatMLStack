import './App.css';
import FeatureDiscovery from './pages/OnlineFeatureStore/components/Discovery/FeatureDiscovery';
import StoreDiscovery from './pages/OnlineFeatureStore/components/Discovery/StoreDiscovery';
import JobDiscovery from './pages/OnlineFeatureStore/components/Discovery/JobDiscovery';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
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
import { AuthProvider } from './pages/Auth/AuthContext';
import ProtectedRoute from './pages/Auth/ProtectedRoute';
import Login from './pages/Auth/Login';
import Register from './pages/Auth/Register';
import HealthCheck from './Health';

function App() {
    return (
      <AuthProvider>
        <Router basename={process.env.PUBLIC_USER_BASE_URL}>
          <Routes>
            {/* Public Routes */}
            <Route path="/login" element={<Login />} />
            <Route path="/register" element={<Register />} />
  
            {/* Protected Routes */}
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
  
            {/* Redirect to Login */}
            <Route path="*" element={<Login />} />
            <Route path="/health" component={HealthCheck} />
          </Routes>
        </Router>
      </AuthProvider>
    );
  }

export default App;
