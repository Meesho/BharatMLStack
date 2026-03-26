package gcp

import (
	"context"
	"fmt"

	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
	"golang.org/x/oauth2/google"
	"github.com/rs/zerolog/log"
)

// ServiceAccountRole represents the IAM role for workload identity binding
type ServiceAccountRole string

const (
	// IamWorkloadIdentityUser is the role that allows Kubernetes Service Accounts to impersonate GCP Service Accounts
	IamWorkloadIdentityUser ServiceAccountRole = "roles/iam.workloadIdentityUser"
	// IamWorkloadIdentityAdmin is the admin role for workload identity (not typically used)
	IamWorkloadIdentityAdmin ServiceAccountRole = "roles/iam.workloadIdentityAdmin"
)

// ServiceAccountConfig contains the configuration for workload identity binding
type ServiceAccountConfig struct {
	// ServiceAccount is the GCP service account email (e.g., "sa-name@project.iam.gserviceaccount.com")
	ServiceAccount string
	// Role is the IAM role to bind (typically IamWorkloadIdentityUser)
	Role ServiceAccountRole
	// ProjectID is the GCP project ID where the service account exists
	ProjectID string
	// Namespace is the Kubernetes namespace where the KSA will be created
	Namespace string
	// KSAName is the Kubernetes Service Account name (typically same as namespace)
	KSAName string
}

// ServiceAccountService provides methods to bind service accounts with workload identity
type ServiceAccountService interface {
	BindServiceAccount(ctx context.Context) error
}

type serviceAccountService struct {
	serviceAccount string
	role           ServiceAccountRole
	projectID      string
	namespace      string
	ksaName        string
}

// NewServiceAccountService creates a new service account service for workload identity binding
func NewServiceAccountService(config ServiceAccountConfig) ServiceAccountService {
	return &serviceAccountService{
		serviceAccount: config.ServiceAccount,
		role:           config.Role,
		projectID:      config.ProjectID,
		namespace:      config.Namespace,
		ksaName:        config.KSAName,
	}
}

// BindServiceAccount binds a Kubernetes Service Account (KSA) to a GCP Service Account (GSA) using workload identity
// This allows pods in the Kubernetes namespace to use the GCP service account credentials
// Member format: serviceAccount:{PROJECT_ID}.svc.id.goog[{NAMESPACE}/{KSA_NAME}]
// Role: roles/iam.workloadIdentityUser
func (sa *serviceAccountService) BindServiceAccount(ctx context.Context) error {
	log.Info().
		Str("serviceAccount", sa.serviceAccount).
		Str("projectID", sa.projectID).
		Str("namespace", sa.namespace).
		Str("ksaName", sa.ksaName).
		Str("role", string(sa.role)).
		Msg("Starting workload identity binding")

	// Fetch GCP credentials (uses Application Default Credentials)
	// This works with:
	// - GCE metadata service (when running on GCE/GKE) - primary method
	// - gcloud auth application-default login (for local development)
	// Note: JSON-based credentials (GOOGLE_APPLICATION_CREDENTIALS) are not used
	credentials, err := google.FindDefaultCredentials(ctx, iam.CloudPlatformScope)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed to fetch GCP credentials - ensure running on GCE/GKE or use gcloud auth application-default login")
		return fmt.Errorf("failed to fetch GCP credentials: %w", err)
	}

	log.Info().
		Str("projectID", credentials.ProjectID).
		Msg("GCP credentials fetched successfully")

	// Create IAM service client
	svc, err := iam.NewService(ctx, option.WithCredentials(credentials))
	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed to create IAM service client")
		return fmt.Errorf("failed to create IAM service: %w", err)
	}

	// Construct resource path: projects/{PROJECT_ID}/serviceAccounts/{SERVICE_ACCOUNT_EMAIL}
	// Extract service account email from full email (e.g., "sa-name@project.iam.gserviceaccount.com" -> "sa-name")
	resource := fmt.Sprintf("projects/%s/serviceAccounts/%s", sa.projectID, sa.serviceAccount)

	// Construct member: serviceAccount:{PROJECT_ID}.svc.id.goog[{NAMESPACE}/{KSA_NAME}]
	// This is the Kubernetes Service Account that will impersonate the GCP Service Account
	member := fmt.Sprintf("serviceAccount:%s.svc.id.goog[%s/%s]", sa.projectID, sa.namespace, sa.ksaName)
	role := string(sa.role)

	log.Info().
		Str("resource", resource).
		Str("member", member).
		Str("role", role).
		Msg("Constructed IAM policy binding parameters")

	// Get current IAM policy for the service account
	policyCall := svc.Projects.ServiceAccounts.GetIamPolicy(resource)
	policy, err := policyCall.Do()
	if err != nil {
		log.Error().
			Err(err).
			Str("resource", resource).
			Msg("Failed to get IAM policy for service account")
		return fmt.Errorf("failed to get IAM policy: %w", err)
	}

	log.Info().
		Int("bindingCount", len(policy.Bindings)).
		Msg("Retrieved current IAM policy")

	// Check if binding exists or needs to be added
	bindingExists := ensureBindingIncluded(policy, member, role)
	if !bindingExists {
		// Binding doesn't exist, update policy to add it
		if err := setPolicy(svc, resource, policy); err != nil {
			log.Error().
				Err(err).
				Str("resource", resource).
				Str("member", member).
				Str("role", role).
				Msg("Failed to set IAM policy")
			return err
		}
		log.Info().
			Str("member", member).
			Str("role", role).
			Msg("IAM policy binding added successfully")
	} else {
		log.Info().
			Str("member", member).
			Str("role", role).
			Msg("IAM policy binding already exists, no changes needed")
	}

	return nil
}

// ensureBindingIncluded checks if the binding exists and adds it if not
// Returns true if binding already exists, false if it was added
func ensureBindingIncluded(policy *iam.Policy, member string, role string) bool {
	// Look for existing binding with the same role
	for _, binding := range policy.Bindings {
		if binding.Role == role {
			// Check if member already exists in this binding
			for _, m := range binding.Members {
				if m == member {
					log.Info().
						Str("member", member).
						Str("role", role).
						Msg("Binding already exists in IAM policy")
					return true
				}
			}
			// Member doesn't exist, add it to the existing binding
			binding.Members = append(binding.Members, member)
			log.Info().
				Str("member", member).
				Str("role", role).
				Int("existingMembers", len(binding.Members)-1).
				Msg("Adding member to existing role binding")
			return false
		}
	}

	// No binding exists for this role, create a new one
	policy.Bindings = append(policy.Bindings, &iam.Binding{
		Role:    role,
		Members: []string{member},
	})
	log.Info().
		Str("member", member).
		Str("role", role).
		Msg("Creating new role binding")
	return false
}

// setPolicy updates the IAM policy for the service account
func setPolicy(svc *iam.Service, resource string, policy *iam.Policy) error {
	setReq := &iam.SetIamPolicyRequest{
		Policy: policy,
	}
	_, err := svc.Projects.ServiceAccounts.SetIamPolicy(resource, setReq).Do()
	if err != nil {
		return fmt.Errorf("failed to set IAM policy: %w", err)
	}
	return nil
}

