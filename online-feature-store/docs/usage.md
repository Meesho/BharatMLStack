# TruffleBox UI Guide

TruffleBox is a comprehensive and intuitive UI to help users onboard new features, models and related entities easily. We will build iteratively and add support overtime for entire feature lifecycle management.

## Table of Contents
- [User Flow](#user-flow)
  - [Getting Started with TruffleBox](#getting-started-with-trufflebox)
  - [Feature Discovery](#feature-discovery)
  - [Feature Registry](#feature-registry)
- [Admin Approval Flow](#admin-approval-flow)
  - [Request Management](#request-management)

---

## User Flow

### Getting Started with TruffleBox

#### Authentication
Users can access TruffleBox through registration or login:

**Registration**: New users should fill in all details and click Register.

<div align="center">
  <img src="../assets/trufflebox/trufflebox-registration.png" alt="Registration Screen" width="500"/>
</div>

**Login**: Existing users can login with their registered email and password.

<div align="center">
  <img src="../assets/trufflebox/trufflebox-login.png" alt="Login Screen" width="500"/>
</div>

#### Navigation
After logging in, you'll be redirected to the feature-discovery page. Access the Control Center by clicking the hamburger icon in the top left corner.

<div align="center">
  <img src="../assets/trufflebox/trufflebox-navigation.png" alt="Control Center Navigation" width="500"/>
</div>

---

### Feature Discovery

The Feature Discovery page displays approved entities, feature groups, and features.

<div align="center">
  <img src="../assets/trufflebox/trufflebox-feature-discovery.png" alt="Feature Discovery Landing Page" width="500"/>
</div>

You can:
- View details by clicking the info icon
- Edit entities, feature groups, and features as needed

#### Entity Management

<div align="center">
  <img src="../assets/trufflebox/trufflebox-feature-discovery-entity-details.png" alt="Entity Details" width="500"/>
</div>

View entity details and edit them (limited to In Memory Cache and Distributed Cache details excluding config ID). Submit changes via "Save Changes" to raise an edit request.

<div align="center">
  <img src="../assets/trufflebox/trufflebox-edit-entity.png" alt="Edit Entity" width="500"/>
</div>

#### Feature Group Management

<div align="center">
  <img src="../assets/trufflebox/trufflebox-feature-discovery-fg-details.png" alt="Feature Group Details" width="500"/>
</div>

Edit feature groups (TTL, In-Memory Cache Enabled, Distributed Cache Enabled, Layout Version) and submit changes to raise an edit request.

<div align="center">
  <img src="../assets/trufflebox/trufflebox-edit-fg.png" alt="Edit Feature Group" width="500"/>
</div>

#### Feature Management

<div align="center">
  <img src="../assets/trufflebox/trufflebox-feature-discovery-feature-details.png" alt="Feature Details" width="500"/>
</div>

Edit features (Default Value, Source Base Path, Source Data Column, Storage Provider) and submit changes to raise an edit request.

<div align="center">
  <img src="../assets/trufflebox/trufflebox-edit-features.png" alt="Edit Features" width="500"/>
</div>

#### Store Discovery
Access Store Discovery from the Control Center to view all stores in the database.

<div align="center">
  <img src="../assets/trufflebox/trufflebox-store-discovery.png" alt="Store Discovery" width="500"/>
</div>

You can search for specific stores but have view-only access.

#### Job Discovery
Access Job Discovery from the Control Center to view all jobs in the database.

<div align="center">
  <img src="../assets/trufflebox/trufflebox-job-discovery.png" alt="Job Discovery" width="500"/>
</div>

You can search for specific jobs but have view-only access.

---

### Feature Registry

In the Control Center, find the 'Feature Registry' accordion to access various registry options for component registration.

<div align="center">
  <img src="../assets/trufflebox/trufflebox-navigation.png" alt="Feature Registry Accordion" width="500"/>
</div>

#### Request Status Tracking
After raising a request, track its status in the respective registry page. For rejected requests, view the rejection reason by clicking the info icon in the Actions column.

#### Step-by-Step Registration Guide

For proper feature lifecycle management, register components in this order:
1. Store
2. Job
3. Entity
4. Feature Group
5. Features (if not added during Feature Group registration)

#### Store Registry

<div align="center">
  <img src="../assets/trufflebox/trufflebox-register-store.png" alt="Register Store" width="500"/>
</div>

Access Store Registry from the Control Center to view raised requests and register new stores. Fill required data and submit to raise a request.

<div align="center">
  <img src="../assets/trufflebox/trufflebox-register-store-details.png" alt="Store Details" width="500"/>
</div>

**Important Considerations:**
- Always add primary keys for proper data identification
- Accurate store configuration is crucial as changes later can be complex
- Admin approval creates a database table with your configuration

#### Job Registry

<div align="center">
  <img src="../assets/trufflebox/trufflebox-register-job.png" alt="Create Job" width="500"/>
</div>

Access Job Registry from the Control Center to view raised requests and create new jobs. Fill required data and submit your request.

<div align="center">
  <img src="../assets/trufflebox/trufflebox-register-job-details.png" alt="Job Details" width="500"/>
</div>

Ensure job details are accurate before proceeding to Entity Registry.

#### Entity Registry

<div align="center">
  <img src="../assets/trufflebox/trufflebox-register-entity.png" alt="Create Entity" width="500"/>
</div>

Access Entity Registry from the Control Center to view raised requests and create new entities. Fill required data and submit your request.

<div align="center">
  <img src="../assets/trufflebox/trufflebox-register-entity-details.png" alt="Entity Detail View" width="500"/>
</div>

**Important Considerations:**
- Ensure entity details align with your data model
- The entity serves as a logical container for feature groups

#### Feature Group Registry

<div align="center">
  <img src="../assets/trufflebox/trufflebox-register-fg.png" alt="Create Feature Group" width="500"/>
</div>

Access Feature Group Registry from the Control Center to view raised requests and create new feature groups. Fill required data and submit your request.

<div align="center">
  <img src="../assets/trufflebox/trufflebox-register-fg-details.png" alt="Feature Group Detail View" width="500"/>
</div>

**Important Considerations:**
- Primary keys must match the store primary keys
- TTL settings determine how long feature data is stored
- Configure cache settings based on access patterns
- Approved feature groups automatically add necessary columns to the database table

#### Feature Addition

<div align="center">
  <img src="../assets/trufflebox/trufflebox-add-features.png" alt="Add Features" width="500"/>
</div>

Access Feature Addition from the Control Center to view raised requests and add new features. Fill required data and submit your request.

<div align="center">
  <img src="../assets/trufflebox/trufflebox-add-features-details.png" alt="Feature Detail View" width="500"/>
</div>

**Important Considerations:**
- Ensure feature data types are compatible with source data
- Set appropriate default values and correct source data column mapping
- Approved features automatically add columns to the database table

#### Need Help?
Please reach out to the Orion core team for any questions about using TruffleBox.

---

## Admin Approval Flow

As an admin, you're responsible for reviewing and managing user requests.

### Request Management

#### Viewing All Requests
After logging in as an admin, you can see all pending requests across different components (Stores, Jobs, Entities, Feature Groups, Features).

<div align="center">
  <img src="../assets/trufflebox/trufflebox-approve-store.png" alt="Admin Dashboard" width="500"/>
</div>

#### Request Approval Process

1. **Review Details**: Click the info icon to view complete request details

<div align="center">
  <img src="../assets/trufflebox/trufflebox-approve-store.png" alt="Request Details" width="500"/>
</div>

2. **Approval Option**: After review, use the approve/reject buttons

<div align="center">
  <img src="../assets/trufflebox/trufflebox-approve-store.png" alt="Approval Buttons" width="500"/>
</div>

3. **Approval Process**:  
   Click "Approve" to process the request. The system will create database tables or add columns as needed. A success message confirms completion.

<div align="center">
  <img src="../assets/trufflebox/trufflebox-approve-store.png" alt="Approval Success" width="500"/>
</div>

4. **Rejection Process**:  
   Click "Reject" to deny a request. Provide a rejection reason to help users understand why their request wasn't approved.

<div align="center">
  <img src="../assets/trufflebox/trufflebox-reject-popup.png" alt="Rejection Reason" width="500"/>
</div>
   
Users can view the rejection reason in their respective registry page.

#### Admin Support
If you need assistance with admin functions, please contact the Orion core team.
