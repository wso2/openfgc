/*
 * Copyright (c) 2026, WSO2 LLC. (https://www.wso2.com).
 *
 * WSO2 LLC. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package consent

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	authmodel "github.com/wso2/openfgc/internal/authresource/model"
	"github.com/wso2/openfgc/internal/consent/model"
	"github.com/wso2/openfgc/internal/consent/validator"
	"github.com/wso2/openfgc/internal/system/config"
	dbmodel "github.com/wso2/openfgc/internal/system/database/model"
	"github.com/wso2/openfgc/internal/system/error/serviceerror"
	"github.com/wso2/openfgc/internal/system/log"
	"github.com/wso2/openfgc/internal/system/stores"
	"github.com/wso2/openfgc/internal/system/utils"
)

// ConsentService defines the exported service interface
type ConsentService interface {
	CreateConsent(ctx context.Context, req model.ConsentAPIRequest, clientID, orgID string) (*model.ConsentResponse, *serviceerror.ServiceError)
	GetConsent(ctx context.Context, consentID, orgID string) (*model.ConsentResponse, *serviceerror.ServiceError)
	SearchConsentsDetailed(ctx context.Context, filters model.ConsentSearchFilters) (*model.ConsentDetailSearchResponse, *serviceerror.ServiceError)
	UpdateConsent(ctx context.Context, req model.ConsentAPIUpdateRequest, clientID, orgID, consentID string) (*model.ConsentResponse, *serviceerror.ServiceError)
	RevokeConsent(ctx context.Context, consentID, orgID string, req model.ConsentRevokeRequest) (*model.ConsentRevokeResponse, *serviceerror.ServiceError)
	ValidateConsent(ctx context.Context, req model.ValidateRequest, orgID string) (*model.ValidateResponse, *serviceerror.ServiceError)
	SearchConsentsByAttribute(ctx context.Context, key, value, orgID string) (*model.ConsentAttributeSearchResponse, *serviceerror.ServiceError)
	// GetConsentDelegates returns all registered delegates for a single consent.
	// Reads from CONSENT_AUTH_RESOURCE rows (RESOURCES JSON blob) and CONSENT_ATTRIBUTE rows.
	GetConsentDelegates(ctx context.Context, consentID, orgID string) (*model.DelegateListResponse, *serviceerror.ServiceError)
}

// consentService implements the ConsentService interface
type consentService struct {
	stores *stores.StoreRegistry
}

// newConsentService creates a new consent service
func newConsentService(registry *stores.StoreRegistry) ConsentService {
	return &consentService{
		stores: registry,
	}
}

// ── Delegation helpers ────────────────────────────────────────────────────────

// parseDelegationConfig reads delegation attributes from a consent's attribute map
// and returns a DelegationConfig. Returns an empty DelegationConfig (IsGuardianConsent=false)
// when no delegation.type attribute is present.
func parseDelegationConfig(attMap map[string]string) model.DelegationConfig {
	validUntil := int64(0)
	if s := attMap[model.AttrGuardianValidUntil]; s != "" {
		if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
			validUntil = ts
		}
	}
	return model.DelegationConfig{
		Type:             attMap[model.AttrDelegationType],
		PrincipalID:      attMap[model.AttrDelegationPrincipalID],
		ValidUntil:       validUntil,
		RevocationPolicy: model.RevocationPolicy(attMap[model.AttrGuardianRevocationPolicy]),
	}
}

// isCallerAuthorizedForPrincipal returns true when callerID is allowed to read or
// act on consents where principalID is the data subject
//
// Access is granted when:
//   - callerID == principalID  (people can always access their own data), OR
//   - callerID has an APPROVED auth resource row on any consent whose
//     delegation.principal_id == principalID, with onBehalfOf == principalID
//     in the RESOURCES JSON blob, AND the delegation has not yet expired.
//
// Uses FindConsentIDsByAttribute (queries CONSENT_ATTRIBUTE)
func (consentService *consentService) isCallerAuthorizedForPrincipal(
	ctx context.Context,
	callerID, principalID, orgID string,
) (bool, error) {
	// Principals can always access their own data.
	if callerID == principalID {
		return true, nil
	}

	// Find all consents where this person is the data subject.
	// FindConsentIDsByAttribute queries CONSENT_ATTRIBUTE directly.
	consentStore := consentService.stores.Consent
	consentIDs, err := consentStore.FindConsentIDsByAttribute(
		ctx, model.AttrDelegationPrincipalID, principalID, orgID,
	)
	if err != nil {
		return false, fmt.Errorf("failed to look up delegated consents: %w", err)
	}
	if len(consentIDs) == 0 {
		return false, nil
	}

	approvedStatus := string(config.Get().Consent.AuthStatusMappings.ApprovedState)

	for _, consentID := range consentIDs {
		// Load attributes to check whether delegation has expired
		attributes, err := consentStore.GetAttributesByConsentID(ctx, consentID, orgID)
		if err != nil {
			continue
		}
		attMap := make(map[string]string)
		for _, a := range attributes {
			attMap[a.AttKey] = a.AttValue
		}
		cfg := parseDelegationConfig(attMap)
		if cfg.IsExpired() {
			// Delegation period ended — this delegate's rights have lapsed.
			continue
		}

		// Check whether callerID has an active auth resource with onBehalfOf = principalID.
		authResources, err := consentService.stores.AuthResource.GetByConsentID(ctx, consentID, orgID)
		if err != nil {
			continue
		}
		for _, ar := range authResources {
			if ar.AuthStatus != approvedStatus {
				continue
			}
			if ar.UserID == nil || *ar.UserID != callerID {
				continue
			}
			if ar.Resources == nil || *ar.Resources == "" {
				continue
			}
			var res map[string]interface{}
			if json.Unmarshal([]byte(*ar.Resources), &res) != nil {
				continue
			}
			if onBehalfOf, _ := res["onBehalfOf"].(string); onBehalfOf == principalID {
				return true, nil
			}
		}
	}
	return false, nil
}

// ── end delegation helpers ────────────────────────────────────────────────────

// CreateConsent creates a new consent with all related entities in a single transaction
func (consentService *consentService) CreateConsent(ctx context.Context, req model.ConsentAPIRequest, clientID, orgID string) (*model.ConsentResponse, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)

	logger.Info("Creating consent",
		log.String("client_id", clientID),
		log.String("org_id", orgID),
		log.String("consent_type", req.Type))

	if err := validator.ValidateConsentCreateRequest(req, clientID, orgID); err != nil {
		logger.Warn("Consent create request validation failed", log.Error(err))
		return nil, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error())
	}

	logger.Debug("initial request validation successful")

	// Note: ValidateDelegationAttributes is called in the handler before this method
	// is reached, so delegation attribute validation has already been applied.

	// Convert API request to internal format
	createReq, err := req.ToConsentCreateRequest()
	if err != nil {
		logger.Error("Failed to convert API request to internal format", log.Error(err))
		return nil, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error())
	}

	// HANDLE PURPOSES (validate and resolve all purposes)
	var resolvedPurposes []model.ConsentPurposeCreateRequest
	if len(createReq.Purposes) > 0 {
		var err error
		resolvedPurposes, err = consentService.validatePurposes(ctx, createReq.Purposes, clientID, orgID)
		if err != nil {
			logger.Error("Purpose validation failed", log.Error(err))
			return nil, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error())
		}
		logger.Debug("Purposes validated and resolved",
			log.Int("purpose_count", len(resolvedPurposes)))
	}

	// Extract auth statuses
	authStatuses := make([]string, 0, len(createReq.AuthResources))
	for _, ar := range createReq.AuthResources {
		authStatuses = append(authStatuses, ar.AuthStatus)
	}

	// Derive consent status from authorization states
	consentStatus := validator.EvaluateConsentStatusFromAuthStatuses(authStatuses)
	logger.Debug("Consent status derived from authorizations",
		log.String("consent_status", consentStatus),
		log.Int("auth_count", len(authStatuses)))

	// Generate IDs and timestamp
	consentID := utils.GenerateUUID()
	currentTime := utils.GetCurrentTimeMillis()

	logger.Debug("Generated consent ID", log.String("consent_id", consentID))

	// Create consent entity
	consent := &model.Consent{
		ConsentID:                  consentID,
		CreatedTime:                currentTime,
		UpdatedTime:                currentTime,
		ClientID:                   clientID,
		ConsentType:                createReq.ConsentType,
		CurrentStatus:              consentStatus,
		ConsentFrequency:           createReq.ConsentFrequency,
		ValidityTime:               createReq.ValidityTime,
		RecurringIndicator:         createReq.RecurringIndicator,
		DataAccessValidityDuration: createReq.DataAccessValidityDuration,
		OrgID:                      orgID,
	}

	// Get stores from registry
	consentStore := consentService.stores.Consent
	authResourceStore := consentService.stores.AuthResource

	// Build list of transactional operations
	queries := []func(tx dbmodel.TxInterface) error{
		// Create consent
		func(tx dbmodel.TxInterface) error {
			return consentStore.Create(tx, consent)
		},
	}

	// Add attributes if provided
	if len(createReq.Attributes) > 0 {
		logger.Debug("Adding consent attributes", log.Int("attribute_count", len(createReq.Attributes)))
		attributes := make([]model.ConsentAttribute, 0, len(createReq.Attributes))
		for key, value := range createReq.Attributes {
			attribute := model.ConsentAttribute{
				ConsentID: consentID,
				AttKey:    key,
				AttValue:  value,
				OrgID:     orgID,
			}
			attributes = append(attributes, attribute)
		}
		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return consentStore.CreateAttributes(tx, attributes)
		})
	}

	// Create audit record
	auditID := utils.GenerateUUID()
	actionBy := clientID // Client ID as the action initiator
	reason := "Initial consent creation"
	audit := &model.ConsentStatusAudit{
		StatusAuditID:  auditID,
		ConsentID:      consentID,
		CurrentStatus:  consent.CurrentStatus,
		ActionTime:     currentTime,
		Reason:         &reason,   // Pointer to string value
		ActionBy:       &actionBy, // Pointer to string value
		PreviousStatus: nil,       // nil = no previous status (first creation)
		OrgID:          orgID,
	}
	queries = append(queries, func(tx dbmodel.TxInterface) error {
		return consentStore.CreateStatusAudit(tx, audit)
	})

	// Add authorization resources if provided
	if len(createReq.AuthResources) > 0 {
		logger.Debug("Adding authorization resources", log.Int("authorization_count", len(createReq.AuthResources)))
	}
	for _, authReq := range createReq.AuthResources {
		authID := utils.GenerateUUID()

		// Marshal resources to JSON if present
		var resourcesJSON *string
		if authReq.Resources != nil {
			resourcesBytes, err := json.Marshal(authReq.Resources)
			if err != nil {
				logger.Error("Failed to marshal authorization resources",
					log.Error(err),
					log.String("auth_id", authID))
				return nil, serviceerror.CustomServiceError(ErrorValidationFailed, fmt.Sprintf("failed to marshal resources: %v", err))
			}
			resourcesStr := string(resourcesBytes)
			resourcesJSON = &resourcesStr
		}

		// Convert to internal format
		var userIDPtr *string
		if authReq.UserID != nil && *authReq.UserID != "" {
			userIDPtr = authReq.UserID
		}

		authResource := &authmodel.AuthResource{
			AuthID:      authID,
			ConsentID:   consentID,
			AuthType:    authReq.AuthType,
			UserID:      userIDPtr,
			AuthStatus:  authReq.AuthStatus,
			UpdatedTime: currentTime,
			Resources:   resourcesJSON,
			OrgID:       orgID,
		}

		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return authResourceStore.Create(tx, authResource)
		})
	}

	// Add purpose and approval records
	for _, resolvedpurpose := range resolvedPurposes {
		// Link consent to purpose
		purposeID := resolvedpurpose.PurposeID
		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return consentStore.CreateConsentPurposeMapping(tx, consentID, purposeID, orgID)
		})

		// Create approval records for each element in the purpose
		for _, element := range resolvedpurpose.Elements {
			approval := &model.ConsentElementApprovalRecord{
				ConsentID:      consentID,
				PurposeID:      purposeID,
				ElementID:      element.ElementID,
				IsUserApproved: element.IsUserApproved,
				Value:          element.Value,
				OrgID:          orgID,
			}

			queries = append(queries, func(tx dbmodel.TxInterface) error {
				return consentStore.CreatePurposeElementApproval(tx, approval)
			})
		}
	}

	// Execute all operations in a single transaction
	logger.Debug("Executing transaction", log.Int("operation_count", len(queries)))
	if err := consentService.stores.ExecuteTransaction(queries); err != nil {
		logger.Error("Failed to create consent in transaction",
			log.Error(err),
			log.String("consent_id", consentID))
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, fmt.Sprintf("failed to create consent: %v", err))
	}

	logger.Info("Consent created successfully", log.String("consent_id", consentID))

	// Check if consent is expired and update status accordingly
	expiredStatusName := string(config.Get().Consent.GetExpiredConsentStatus())
	if consent.ValidityTime != nil && validator.IsConsentExpired(*consent.ValidityTime) {
		// Consent was created with an expired validity time - expire it immediately
		if consent.CurrentStatus != expiredStatusName {
			if err := consentService.expireConsent(ctx, consent, orgID); err != nil {
				logger.Error("Failed to expire consent after creation", log.Error(err))
				// Continue with response - consent object is updated in-memory
			} else {
				// Re-fetch consent to get latest state from DB
				if updatedConsent, fetchErr := consentService.stores.Consent.GetByID(ctx, consentID, orgID); fetchErr == nil && updatedConsent != nil {
					consent = updatedConsent
				}
			}
		}
	}

	// Retrieve related data after creation
	logger.Debug("Retrieving related data for response")
	authResources, _ := authResourceStore.GetByConsentID(ctx, consentID, orgID)
	attributes, _ := consentService.stores.Consent.GetAttributesByConsentID(ctx, consentID, orgID)

	// Convert attributes slice to map[string]string
	attributesMap := make(map[string]string)
	for _, a := range attributes {
		attributesMap[a.AttKey] = a.AttValue
	}

	// Use the generic method to resolve purposes with all purposes
	purposes, err := consentService.getResolvedConsentPurposes(ctx, consentID, orgID)
	if err != nil {
		logger.Error("Failed to resolve purposes for response",
			log.String("consent_id", consentID),
			log.Error(err))
		// Return with empty purposes on error rather than failing the whole response
		purposes = []model.ConsentPurposeItem{}
	}

	// Build complete response using the resolved purposes data
	response := buildConsentResponse(consent, purposes, attributesMap, authResources)

	logger.Info("Consent creation completed",
		log.String("consent_id", consentID),
		log.String("status", consent.CurrentStatus),
		log.Int("auth_resources", len(authResources)),
		log.Int("purpose_count", len(purposes)),
		log.Int("attributes", len(attributesMap)))

	return response, nil
}

// GetConsent retrieves a consent by ID with all related data
func (consentService *consentService) GetConsent(ctx context.Context, consentID, orgID string) (*model.ConsentResponse, *serviceerror.ServiceError) {

	logger := log.GetLogger().WithContext(ctx)
	logger.Debug("Retrieving consent",
		log.String("consent_id", consentID),
		log.String("org_id", orgID),
	)

	// Get stores
	consentStore := consentService.stores.Consent
	authResourceStore := consentService.stores.AuthResource

	// Get consent
	consent, err := consentStore.GetByID(ctx, consentID, orgID)
	if err != nil {
		logger.Error("Failed to retrieve consent",
			log.Error(err),
			log.String("consent_id", consentID),
		)
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, err.Error())
	}
	if consent == nil {
		logger.Warn("Consent not found", log.String("consent_id", consentID))
		return nil, serviceerror.CustomServiceError(ErrorConsentNotFound, fmt.Sprintf("Consent with ID '%s' not found", consentID))
	}

	// Check if consent is expired and update status accordingly
	expiredStatusName := string(config.Get().Consent.GetExpiredConsentStatus())
	if consent.ValidityTime != nil && validator.IsConsentExpired(*consent.ValidityTime) {
		// Update consent status to expired if not already expired
		if consent.CurrentStatus != expiredStatusName {
			if err := consentService.expireConsent(ctx, consent, orgID); err != nil {
				logger.Error("Failed to expire consent", log.Error(err))
				// Continue with response - consent object is updated in-memory
			} else {
				// Re-fetch consent to get latest state from DB
				if updatedConsent, fetchErr := consentStore.GetByID(ctx, consentID, orgID); fetchErr == nil && updatedConsent != nil {
					consent = updatedConsent
				}
			}
		}
	}

	// Retrieve all related data
	attributes, _ := consentStore.GetAttributesByConsentID(ctx, consentID, orgID)
	authResources, _ := authResourceStore.GetByConsentID(ctx, consentID, orgID)

	// Convert attributes slice to map
	attributesMap := make(map[string]string)
	for _, attribute := range attributes {
		attributesMap[attribute.AttKey] = attribute.AttValue
	}

	// Resolve purposes with all purposes
	purposes, err := consentService.getResolvedConsentPurposes(ctx, consentID, orgID)
	if err != nil {
		logger.Error("Failed to resolve purposes", log.Error(err))
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, fmt.Sprintf("failed to resolve purposes: %v", err))
	}

	// Build complete response with all related data
	response := buildConsentResponse(consent, purposes, attributesMap, authResources)

	logger.Debug("Consent retrieved successfully",
		log.String("consent_id", consentID),
		log.String("status", consent.CurrentStatus),
		log.Int("auth_resources", len(authResources)),
		log.Int("purpose_count", len(response.Purposes)),
	)
	return response, nil
}

// SearchConsentsDetailed retrieves consents with nested authorization resources, purposes, and attributes
func (consentService *consentService) SearchConsentsDetailed(ctx context.Context, filters model.ConsentSearchFilters) (*model.ConsentDetailSearchResponse, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Info("Searching consents with detailed data",
		log.String("org_id", filters.OrgID),
		log.Int("client_ids_count", len(filters.ClientIDs)),
		log.Int("user_ids_count", len(filters.UserIDs)),
		log.Int("purpose_names_count", len(filters.PurposeNames)),
		log.Int("statuses_count", len(filters.ConsentStatuses)),
		log.Int("limit", filters.Limit))

	// Validate pagination
	if filters.Limit <= 0 {
		filters.Limit = 10
	}
	if filters.Offset < 0 {
		filters.Offset = 0
	}

	// When a caller requests another person's consents via dataPrincipalId,
	// verify they are a registered delegate for that principal before proceeding with the search.
	// Skipped when CallerID is empty (internal/admin calls without a user header).
	if filters.DataPrincipalID != "" && filters.CallerID != "" {
		authorized, authErr := consentService.isCallerAuthorizedForPrincipal(
			ctx, filters.CallerID, filters.DataPrincipalID, filters.OrgID,
		)
		if authErr != nil {
			logger.Error("Failed to verify delegate authorization",
				log.Error(authErr),
				log.String("caller_id", filters.CallerID),
				log.String("principal_id", filters.DataPrincipalID))
			return nil, serviceerror.CustomServiceError(ErrorInternalServerError, authErr.Error())
		}
		if !authorized {
			logger.Warn("Caller not authorized for principal",
				log.String("caller_id", filters.CallerID),
				log.String("principal_id", filters.DataPrincipalID))
			return nil, serviceerror.CustomServiceError(
				ErrorNotAuthorizedForPrincipal,
				fmt.Sprintf("caller '%s' is not a registered delegate for principal '%s'",
					filters.CallerID, filters.DataPrincipalID),
			)
		}
	}

	// Step 1: Search consents
	consentStore := consentService.stores.Consent
	consents, total, err := consentStore.Search(ctx, filters)
	if err != nil {
		logger.Error("Failed to search consents", log.Error(err))
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, err.Error())
	}

	if len(consents) == 0 {
		return &model.ConsentDetailSearchResponse{
			Data: []model.ConsentDetailResponse{},
			Metadata: model.ConsentSearchMetadata{
				Total:  0,
				Limit:  filters.Limit,
				Offset: filters.Offset,
				Count:  0,
			},
		}, nil
	}

	// Step 2: Extract consent IDs
	consentIDs := make([]string, len(consents))
	for i, c := range consents {
		consentIDs[i] = c.ConsentID
	}

	// Step 3: Batch fetch related data
	authResourceStore := consentService.stores.AuthResource

	authResources, err := authResourceStore.GetByConsentIDs(ctx, consentIDs, filters.OrgID)
	if err != nil {
		logger.Error("Failed to get authorization resources", log.Error(err))
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, err.Error())
	}

	attributesByConsent, err := consentStore.GetAttributesByConsentIDs(ctx, consentIDs, filters.OrgID)
	if err != nil {
		logger.Error("Failed to get consent attributes", log.Error(err))
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, err.Error())
	}

	// Step 4: Group auth resources by consent ID
	authsByConsent := make(map[string][]authmodel.AuthResource)
	for _, auth := range authResources {
		authsByConsent[auth.ConsentID] = append(authsByConsent[auth.ConsentID], auth)
	}

	// Step 5: Assemble detailed responses
	detailedResponses := make([]model.ConsentDetailResponse, 0, len(consents))
	for _, consent := range consents {
		// Build authorizations - initialize as empty slice
		authorizations := make([]model.AuthorizationDetail, 0)
		for _, auth := range authsByConsent[consent.ConsentID] {
			var resources interface{}
			if auth.Resources != nil && *auth.Resources != "" {
				_ = json.Unmarshal([]byte(*auth.Resources), &resources)
			}

			userID := ""
			if auth.UserID != nil {
				userID = *auth.UserID
			}

			authorizations = append(authorizations, model.AuthorizationDetail{
				ID:          auth.AuthID,
				UserID:      userID,
				Type:        auth.AuthType,
				Status:      auth.AuthStatus,
				UpdatedTime: auth.UpdatedTime,
				Resources:   resources,
			})
		}

		// Resolve purposes for this consent
		purposes, err := consentService.getResolvedConsentPurposes(ctx, consent.ConsentID, filters.OrgID)
		if err != nil {
			logger.Warn("Failed to resolve purposes for consent",
				log.String("consent_id", consent.ConsentID),
				log.Error(err))
			// Continue with empty purposes rather than failing
			purposes = []model.ConsentPurposeItem{}
		}

		// Get attributes (already grouped by consent ID)
		attributes := attributesByConsent[consent.ConsentID]
		if attributes == nil {
			attributes = make(map[string]string)
		}

		// Dereference pointer fields for response
		frequency := 0
		if consent.ConsentFrequency != nil {
			frequency = *consent.ConsentFrequency
		}
		validityTime := int64(0)
		if consent.ValidityTime != nil {
			validityTime = *consent.ValidityTime
		}
		recurringIndicator := false
		if consent.RecurringIndicator != nil {
			recurringIndicator = *consent.RecurringIndicator
		}
		dataAccessValidityDuration := int64(0)
		if consent.DataAccessValidityDuration != nil {
			dataAccessValidityDuration = *consent.DataAccessValidityDuration
		}

		detailedResponses = append(detailedResponses, model.ConsentDetailResponse{
			ID:                         consent.ConsentID,
			Purposes:                   purposes,
			CreatedTime:                consent.CreatedTime,
			UpdatedTime:                consent.UpdatedTime,
			ClientID:                   consent.ClientID,
			Type:                       consent.ConsentType,
			Status:                     consent.CurrentStatus,
			Frequency:                  frequency,
			ValidityTime:               validityTime,
			RecurringIndicator:         recurringIndicator,
			DataAccessValidityDuration: dataAccessValidityDuration,
			Attributes:                 attributes,
			Authorizations:             authorizations,
			// DelegationExpired is true when the guardian period has passed.
			// The data principal (now adult/capable) holds full authority.
			// The portal uses this flag to prompt the principal to review inherited consents.
			DelegationExpired: parseDelegationConfig(attributes).IsGuardianConsent() && parseDelegationConfig(attributes).IsExpired(),
		})
	}

	logger.Info("Consents searched with details successfully",
		log.Int("count", len(detailedResponses)),
		log.Int("total", total))

	return &model.ConsentDetailSearchResponse{
		Data: detailedResponses,
		Metadata: model.ConsentSearchMetadata{
			Total:  total,
			Limit:  filters.Limit,
			Offset: filters.Offset,
			Count:  len(detailedResponses),
		},
	}, nil
}

// UpdateConsent updates an existing consent
func (consentService *consentService) UpdateConsent(ctx context.Context, req model.ConsentAPIUpdateRequest, clientID, orgID, consentID string) (*model.ConsentResponse, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Info("Updating consent",
		log.String("consent_id", consentID),
		log.String("client_id", clientID),
		log.String("org_id", orgID))

	// Get stores
	authResourceStore := consentService.stores.AuthResource
	consentStore := consentService.stores.Consent

	if err := validator.ValidateConsentUpdateRequest(req); err != nil {
		logger.Warn("Consent update request validation failed", log.Error(err))
		return nil, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error())
	}

	// Convert to internal format
	updateReq, convertErr := req.ToConsentUpdateRequest()
	if convertErr != nil {
		logger.Warn("Failed to convert update request", log.Error(convertErr))
		return nil, serviceerror.CustomServiceError(ErrorValidationFailed, convertErr.Error())
	}

	logger.Debug("Request validation successful")

	// Check if consent exists
	existing, err := consentStore.GetByID(ctx, consentID, orgID)
	if err != nil {
		logger.Error("Failed to retrieve consent", log.Error(err), log.String("consent_id", consentID))
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, err.Error())
	}
	if existing == nil {
		logger.Warn("Consent not found", log.String("consent_id", consentID))
		return nil, serviceerror.CustomServiceError(ErrorConsentNotFound, fmt.Sprintf("Consent with ID '%s' not found", consentID))
	}

	// Validate clientID matches - only the owner client can update the consent
	if existing.ClientID != clientID {
		logger.Warn("ClientID mismatch - unauthorized update attempt",
			log.String("consent_client_id", existing.ClientID),
			log.String("request_client_id", clientID),
			log.String("consent_id", consentID))
		return nil, serviceerror.CustomServiceError(ErrorValidationFailed,
			fmt.Sprintf("Client '%s' is not authorized to update consent '%s'", clientID, consentID))
	}

	currentTime := utils.GetCurrentTimeMillis()
	previousStatus := existing.CurrentStatus

	if req.DataAccessValidityDuration != nil {
		// Validate that it's non-negative
		if *req.DataAccessValidityDuration < 0 {
			logger.Warn("Invalid data access validity duration", log.Any("duration", *req.DataAccessValidityDuration))
			return nil, serviceerror.CustomServiceError(ErrorValidationFailed, "dataAccessValidityDuration must be non-negative")
		}
		updateReq.DataAccessValidityDuration = req.DataAccessValidityDuration
	}

	// Derive new consent status from authorization states if auth resources are being updated
	var newStatus string
	var statusChanged bool
	if updateReq.AuthResources != nil {

		// Extract auth statuses
		authStatuses := make([]string, 0, len(updateReq.AuthResources))
		for _, ar := range updateReq.AuthResources {
			authStatuses = append(authStatuses, ar.AuthStatus)
		}

		newStatus = validator.EvaluateConsentStatusFromAuthStatuses(authStatuses)
		statusChanged = (newStatus != previousStatus)
		if statusChanged {
			logger.Debug("Consent status changed",
				log.String("previous_status", previousStatus),
				log.String("new_status", newStatus))
		}
	} else {
		newStatus = existing.CurrentStatus
		statusChanged = false
	}

	// Update consent fields (clientID is not updated - validated to match existing)
	consent := &model.Consent{
		ConsentID:                  consentID,
		UpdatedTime:                currentTime,
		CurrentStatus:              newStatus,
		ConsentType:                updateReq.ConsentType,
		ConsentFrequency:           updateReq.ConsentFrequency,
		ValidityTime:               updateReq.ValidityTime,
		RecurringIndicator:         updateReq.RecurringIndicator,
		DataAccessValidityDuration: updateReq.DataAccessValidityDuration,
		OrgID:                      orgID,
	}

	// Build transactional operations
	queries := []func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return consentStore.Update(tx, consent)
		},
	}

	if statusChanged {

		// Append status update to existing queries (don't replace the array)
		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return consentStore.UpdateStatus(tx, consentID, orgID, newStatus, currentTime)
		})

		// Create status audit if status changed
		auditID := utils.GenerateUUID()
		actionBy := existing.ClientID // Use client ID as action initiator
		reason := "Consent status updated based on authorization states during consent update"
		audit := &model.ConsentStatusAudit{
			StatusAuditID:  auditID,
			ConsentID:      consentID,
			CurrentStatus:  newStatus,
			ActionTime:     currentTime,
			Reason:         &reason,
			ActionBy:       &actionBy,
			PreviousStatus: &previousStatus,
			OrgID:          orgID,
		}

		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return consentStore.CreateStatusAudit(tx, audit)
		})

	}

	// Update attributes - delete old and create new if provided
	if updateReq.Attributes != nil {
		// Delete existing attributes
		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return consentStore.DeleteAttributesByConsentID(tx, consentID, orgID)
		})

		// Create new attributes if not empty
		if len(updateReq.Attributes) > 0 {
			attributes := make([]model.ConsentAttribute, 0, len(updateReq.Attributes))
			for key, value := range updateReq.Attributes {
				attribute := model.ConsentAttribute{
					ConsentID: consentID,
					AttKey:    key,
					AttValue:  value,
					OrgID:     orgID,
				}
				attributes = append(attributes, attribute)
			}

			queries = append(queries, func(tx dbmodel.TxInterface) error {
				return consentStore.CreateAttributes(tx, attributes)
			})
		}
	}

	// Update authorization resources if provided
	if updateReq.AuthResources != nil {

		// Delete existing auth resources
		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return authResourceStore.DeleteByConsentID(tx, consentID, orgID)
		})

		// Create new auth resources if not empty
		if len(updateReq.AuthResources) > 0 {
			for _, authReq := range updateReq.AuthResources {
				authID := utils.GenerateUUID()

				// Marshal resources to JSON if present
				var resourcesJSON *string
				if authReq.Resources != nil {
					resourcesBytes, err := json.Marshal(authReq.Resources)
					if err != nil {
						return nil, serviceerror.CustomServiceError(ErrorValidationFailed, fmt.Sprintf("failed to marshal resources: %v", err))
					}
					resourcesStr := string(resourcesBytes)
					resourcesJSON = &resourcesStr
				}

				authResource := &authmodel.AuthResource{
					AuthID:      authID,
					ConsentID:   consentID,
					AuthType:    authReq.AuthType,
					UserID:      authReq.UserID,
					AuthStatus:  authReq.AuthStatus,
					UpdatedTime: currentTime,
					Resources:   resourcesJSON,
					OrgID:       orgID,
				}

				queries = append(queries, func(tx dbmodel.TxInterface) error {
					return authResourceStore.Create(tx, authResource)
				})
			}
		}
	}

	// HANDLE PURPOSES UPDATE (validate and resolve all purposes if provided)
	var resolvedPurposes []model.ConsentPurposeCreateRequest
	if updateReq.Purposes != nil {
		var err error
		resolvedPurposes, err = consentService.validatePurposes(ctx, updateReq.Purposes, existing.ClientID, orgID)
		if err != nil {
			logger.Error("Purpose validation failed", log.Error(err))
			return nil, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error())
		}
		logger.Debug("Purposes validated and resolved",
			log.Int("purpose_count", len(resolvedPurposes)))

		// Delete existing purpose mappings and approvals
		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return consentStore.DeleteConsentPurposeMappingsByConsentID(tx, consentID, orgID)
		})

		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return consentStore.DeletePurposeElementApprovalsByConsentID(tx, consentID, orgID)
		})

		// Add new purpose and approval records
		for _, purpose := range resolvedPurposes {
			// Link consent to purpose
			purposeID := purpose.PurposeID
			queries = append(queries, func(tx dbmodel.TxInterface) error {
				return consentStore.CreateConsentPurposeMapping(tx, consentID, purposeID, orgID)
			})

			// Create approval records for each purpose in the purpose
			for _, element := range purpose.Elements {
				approval := &model.ConsentElementApprovalRecord{
					ConsentID:      consentID,
					PurposeID:      purposeID,
					ElementID:      element.ElementID,
					IsUserApproved: element.IsUserApproved,
					Value:          element.Value,
					OrgID:          orgID,
				}

				queries = append(queries, func(tx dbmodel.TxInterface) error {
					return consentStore.CreatePurposeElementApproval(tx, approval)
				})
			}
		}
	}

	// Execute transaction
	logger.Debug("Executing update transaction", log.Int("operation_count", len(queries)))
	if err := consentService.stores.ExecuteTransaction(queries); err != nil {
		logger.Error("Failed to update consent in transaction",
			log.Error(err),
			log.String("consent_id", consentID))
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, err.Error())
	}

	// Get updated consent
	logger.Debug("Retrieving updated consent data")
	updated, getErr := consentStore.GetByID(ctx, consentID, orgID)
	if getErr != nil {
		logger.Error("Failed to retrieve updated consent", log.Error(getErr))
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, getErr.Error())
	}

	// Check if consent expiration status needs to be updated
	expiredStatusName := string(config.Get().Consent.GetExpiredConsentStatus())

	// Case 1: Consent is expired and should be marked as expired
	if updated.ValidityTime != nil && validator.IsConsentExpired(*updated.ValidityTime) {
		if updated.CurrentStatus != expiredStatusName {
			if err := consentService.expireConsent(ctx, updated, orgID); err != nil {
				logger.Error("Failed to expire consent after update", log.Error(err))
			} else {
				// Re-fetch consent to get latest state from DB
				if refreshedConsent, fetchErr := consentStore.GetByID(ctx, consentID, orgID); fetchErr == nil && refreshedConsent != nil {
					updated = refreshedConsent
				}
			}
		}
	} else if updated.CurrentStatus == expiredStatusName {
		// Case 2: Consent was expired but is no longer expired (validityTime updated to future)
		// Re-evaluate status based on authorization states
		allAuthResources, err := authResourceStore.GetByConsentID(ctx, consentID, orgID)
		if err == nil {
			authStatuses := make([]string, 0, len(allAuthResources))
			for _, ar := range allAuthResources {
				authStatuses = append(authStatuses, ar.AuthStatus)
			}

			// Derive new consent status from auth resources
			derivedStatus := validator.EvaluateConsentStatusFromAuthStatuses(authStatuses)

			// Update consent to active status if it should no longer be expired
			if derivedStatus != expiredStatusName {
				currentTime := utils.GetCurrentTimeMillis()

				// Create audit entry
				auditID := utils.GenerateUUID()
				reason := "Consent reactivated - validity time extended to future"
				actionBy := existing.ClientID
				previousStatus := updated.CurrentStatus
				audit := &model.ConsentStatusAudit{
					StatusAuditID:  auditID,
					ConsentID:      consentID,
					CurrentStatus:  derivedStatus,
					ActionTime:     currentTime,
					Reason:         &reason,
					ActionBy:       &actionBy,
					PreviousStatus: &previousStatus,
					OrgID:          orgID,
				}

				// Update status in transaction
				err := consentService.stores.ExecuteTransaction([]func(tx dbmodel.TxInterface) error{
					func(tx dbmodel.TxInterface) error {
						return consentStore.UpdateStatus(tx, consentID, orgID, derivedStatus, currentTime)
					},
					func(tx dbmodel.TxInterface) error {
						return consentStore.CreateStatusAudit(tx, audit)
					},
				})

				if err != nil {
					logger.Error("Failed to reactivate consent after update", log.Error(err))
				} else {
					logger.Info("Consent reactivated after validity time update",
						log.String("consent_id", consentID),
						log.String("previous_status", previousStatus),
						log.String("new_status", derivedStatus))

					// Re-fetch consent to get latest state from DB
					if refreshedConsent, fetchErr := consentStore.GetByID(ctx, consentID, orgID); fetchErr == nil && refreshedConsent != nil {
						updated = refreshedConsent
					}
				}
			}
		}
	}

	authResources, _ := authResourceStore.GetByConsentID(ctx, consentID, orgID)
	attributes, _ := consentStore.GetAttributesByConsentID(ctx, consentID, orgID)

	// Convert attributes slice to map[string]string
	attributesMap := make(map[string]string)
	for _, a := range attributes {
		attributesMap[a.AttKey] = a.AttValue
	}

	// Resolve purposes with all purposes
	purposes, err := consentService.getResolvedConsentPurposes(ctx, consentID, orgID)
	if err != nil {
		logger.Error("Failed to resolve purposes", log.Error(err))
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, fmt.Sprintf("failed to resolve purposes: %v", err))
	}

	// Build complete response
	response := buildConsentResponse(updated, purposes, attributesMap, authResources)

	logger.Info("Consent updated successfully",
		log.String("consent_id", consentID),
		log.String("status", updated.CurrentStatus),
		log.Int("auth_resources", len(authResources)),
		log.Int("purpose_count", len(response.Purposes)),
		log.Int("attributes", len(attributesMap)))

	return response, nil
}

// RevokeConsent updates consent status and creates audit entry
func (consentService *consentService) RevokeConsent(ctx context.Context, consentID, orgID string, req model.ConsentRevokeRequest) (*model.ConsentRevokeResponse, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Info("Revoking consent",
		log.String("consent_id", consentID),
		log.String("org_id", orgID),
		log.String("action_by", req.ActionBy))

	// Validate action by
	if req.ActionBy == "" {
		logger.Warn("Validation failed: ActionBy is required")
		return nil, serviceerror.CustomServiceError(ErrorValidationFailed, "ActionBy is required")
	}

	logger.Debug("Request validation successful")

	revokedStatusName := config.Get().Consent.GetRevokedConsentStatus()

	// Check if consent exists
	store := consentService.stores.Consent
	existing, err := store.GetByID(ctx, consentID, orgID)
	if err != nil {
		logger.Error("Failed to retrieve consent", log.Error(err), log.String("consent_id", consentID))
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, err.Error())
	}
	if existing == nil {
		logger.Warn("Consent not found", log.String("consent_id", consentID))
		return nil, serviceerror.CustomServiceError(ErrorConsentNotFound, fmt.Sprintf("Consent with ID '%s' not found", consentID))
	}

	// Check if consent is already revoked
	if existing.CurrentStatus == string(revokedStatusName) {
		logger.Warn("Consent is already revoked",
			log.String("consent_id", consentID),
			log.String("status", existing.CurrentStatus))
		return nil, serviceerror.CustomServiceError(ErrorConsentAlreadyRevoked, fmt.Sprintf("Consent with ID '%s' is already revoked", consentID))
	}

	// ── Delegation-aware revocation check ────────────
	// For normal self-consented records this entire block is a no-op because
	// parseDelegationConfig returns IsGuardianConsent()=false.
	//
	// For delegated consents the rules are:
	//   if delegation expired  → only the principal (now adult) may revoke
	//   if delegation active   → principal cannot revoke directly (guardian controls)
	//   if delegation active   → delegate must have canRevoke=true AND policy≠SUBJECT_ONLY
	{
		attributes, attErr := store.GetAttributesByConsentID(ctx, consentID, orgID)
		if attErr != nil {
			logger.Error("Failed to load delegation attributes for revocation",
				log.Error(attErr),
				log.String("consent_id", consentID))
			return nil, serviceerror.CustomServiceError(ErrorInternalServerError, attErr.Error())
		}
		attMap := make(map[string]string)
		for _, a := range attributes {
			attMap[a.AttKey] = a.AttValue
		}
		delegCfg := parseDelegationConfig(attMap)

		if delegCfg.IsGuardianConsent() {
			principalID := delegCfg.PrincipalID
			callerID := req.ActionBy
			isPrincipal := callerID == principalID

			if delegCfg.IsExpired() {
				// delegation ended (e.g., child is now an adult).
				// Only the principal has authority; delegates are locked out.
				if !isPrincipal {
					logger.Warn("Revocation denied: delegation expired, caller is not principal",
						log.String("caller_id", callerID),
						log.String("principal_id", principalID))
					return nil, serviceerror.CustomServiceError(
						ErrorDelegationExpired,
						fmt.Sprintf("delegation for principal '%s' has expired; "+
							"only the principal may revoke this consent", principalID),
					)
				}
				// Principal is now an adult — allow revocation to continue.
				logger.Debug("Delegation expired; principal revoking their own consent",
					log.String("principal_id", principalID))

			} else {

				// SUBJECT_ONLY means only the principal may revoke; delegates are blocked.
				// ANY means both the principal and permitted delegates may revoke.
				if delegCfg.RevocationPolicy == model.RevocationPolicySubjectOnly {
					if !isPrincipal {
						// Delegate blocked by policy.
						logger.Warn("Revocation denied: policy is SUBJECT_ONLY",
							log.String("caller_id", callerID))
						return nil, serviceerror.CustomServiceError(
							ErrorRevocationNotPermitted,
							"revocation policy is SUBJECT_ONLY; only the data principal may revoke",
						)
					}
					// Principal is explicitly allowed under SUBJECT_ONLY — fall through.
					logger.Debug("SUBJECT_ONLY policy: principal revoking their own consent",
						log.String("principal_id", principalID))

				} else if isPrincipal {
					// Active guardianship with ANY policy: block the principal from
					// revoking directly — the guardian controls this consent while active.
					logger.Warn("Revocation denied: principal cannot revoke under active guardianship",
						log.String("principal_id", principalID))
					return nil, serviceerror.CustomServiceError(
						ErrorRevocationNotPermitted,
						"the data principal cannot revoke a guardian-controlled consent directly; "+
							"contact your guardian",
					)

				} else {
					// Delegate attempting revocation under ANY policy.
					// Check: delegate must have canRevoke=true on the delegation row for principalID.
					approvedStatus := string(config.Get().Consent.AuthStatusMappings.ApprovedState)
					authResources, _ := consentService.stores.AuthResource.GetByConsentID(ctx, consentID, orgID)
					callerCanRevoke := false
					for _, ar := range authResources {
						if ar.UserID == nil || *ar.UserID != callerID {
							continue
						}
						if ar.AuthStatus != approvedStatus {
							continue
						}
						if ar.Resources == nil || *ar.Resources == "" {
							continue
						}
						var res map[string]interface{}
						if json.Unmarshal([]byte(*ar.Resources), &res) != nil {
							continue
						}
						// principalID — prevents an unrelated canRevoke=true row from
						// accidentally granting revoke rights over a different principal.
						onBehalfOf, _ := res["onBehalfOf"].(string)
						if canRevoke, _ := res["canRevoke"].(bool); canRevoke && onBehalfOf == principalID {
							callerCanRevoke = true
							break
						}
					}
					if !callerCanRevoke {
						logger.Warn("Revocation denied: caller lacks canRevoke permission",
							log.String("caller_id", callerID))
						return nil, serviceerror.CustomServiceError(
							ErrorRevocationNotPermitted,
							fmt.Sprintf("caller '%s' does not have canRevoke permission on this consent", callerID),
						)
					}
					logger.Debug("Delegate canRevoke check passed", log.String("caller_id", callerID))
				}
			}
		}
	}
	// ── end delegation revocation check ──────────────────────────────────────────

	currentTime := utils.GetCurrentTimeMillis()

	// Create audit entry
	auditID := utils.GenerateUUID()
	reason := req.RevocationReason
	audit := &model.ConsentStatusAudit{
		StatusAuditID:  auditID,
		ConsentID:      consentID,
		CurrentStatus:  string(revokedStatusName),
		ActionTime:     currentTime,
		Reason:         &reason,
		ActionBy:       &req.ActionBy,
		PreviousStatus: &existing.CurrentStatus,
		OrgID:          orgID,
	}

	// Get auth resource store for cascading status update
	authResourceStore := consentService.stores.AuthResource

	// Execute transaction - update consent status, all auth resource statuses, and create audit
	logger.Debug("Executing revocation transaction")
	err = consentService.stores.ExecuteTransaction([]func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return store.UpdateStatus(tx, consentID, orgID, string(revokedStatusName), currentTime)
		},
		func(tx dbmodel.TxInterface) error {
			// Update all authorization statuses to system revoked status when consent is revoked
			sysRevokedStatus := string(config.Get().Consent.GetSystemRevokedAuthStatus())
			return authResourceStore.UpdateAllStatusByConsentID(tx, consentID, orgID, sysRevokedStatus, currentTime)
		},
		func(tx dbmodel.TxInterface) error {
			return store.CreateStatusAudit(tx, audit)
		},
	})
	if err != nil {
		logger.Error("Failed to revoke consent in transaction",
			log.Error(err),
			log.String("consent_id", consentID))
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, err.Error())
	}

	logger.Info("Consent revoked successfully",
		log.String("consent_id", consentID),
		log.String("previous_status", existing.CurrentStatus),
		log.String("new_status", string(revokedStatusName)))

	// Build and return response
	response := &model.ConsentRevokeResponse{
		ActionTime:       currentTime / 1000, // Convert milliseconds to seconds
		ActionBy:         req.ActionBy,
		RevocationReason: req.RevocationReason,
	}

	return response, nil
}

// ValidateConsent validates a consent for data access
func (consentService *consentService) ValidateConsent(ctx context.Context, req model.ValidateRequest, orgID string) (*model.ValidateResponse, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Info("Validating consent",
		log.String("consent_id", req.ConsentID),
		log.String("org_id", orgID))

	// Initialize response with invalid state
	response := &model.ValidateResponse{
		IsValid: false,
	}

	// Validate request
	if req.ConsentID == "" {
		logger.Warn("Validation failed: ConsentID is required")
		return nil, serviceerror.CustomServiceError(ErrorValidationFailed, "ConsentID is required")
	}

	logger.Debug("Request validation successful")

	// Get consent
	consentStore := consentService.stores.Consent
	consent, err := consentStore.GetByID(ctx, req.ConsentID, orgID)
	if err != nil {
		logger.Error("Failed to retrieve consent", log.Error(err), log.String("consent_id", req.ConsentID))
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, err.Error())
	}
	if consent == nil {
		logger.Warn("Consent not found", log.String("consent_id", req.ConsentID))
		return nil, serviceerror.CustomServiceError(ErrorConsentNotFound, fmt.Sprintf("Consent with ID '%s' not found", req.ConsentID))
	} else {
		// Check if consent is expired and update status accordingly (only if consent exists)
		expiredStatusName := string(config.Get().Consent.GetExpiredConsentStatus())
		if consent.ValidityTime != nil && validator.IsConsentExpired(*consent.ValidityTime) {
			// Update consent status to expired if not already expired
			if consent.CurrentStatus != expiredStatusName {
				if err := consentService.expireConsent(ctx, consent, orgID); err != nil {
					// Log error but continue with validation
					logger.Warn("Failed to expire consent, continuing with validation",
						log.Error(err),
						log.String("consent_id", consent.ConsentID),
						log.String("org_id", orgID))
				} else {
					// Re-fetch consent after expiring to get latest state
					if updatedConsent, fetchErr := consentStore.GetByID(ctx, req.ConsentID, orgID); fetchErr == nil && updatedConsent != nil {
						consent = updatedConsent
					}
				}
			}
		}
	}

	// Check consent status - only active consents are valid
	activeStatusName := string(config.Get().Consent.GetActiveConsentStatus())
	if consent != nil && consent.CurrentStatus != activeStatusName && response.ErrorCode == 0 {
		response.ErrorCode = 401
		response.ErrorMessage = "invalid_consent_status"
		response.ErrorDescription = fmt.Sprintf("Consent status is '%s', expected '%s'", consent.CurrentStatus, activeStatusName)
	}

	// Retrieve related data for consent information (only if consent exists)
	if consent != nil {
		authResourceStore := consentService.stores.AuthResource

		attributes, _ := consentStore.GetAttributesByConsentID(ctx, consent.ConsentID, orgID)
		authResources, _ := authResourceStore.GetByConsentID(ctx, consent.ConsentID, orgID)

		// Convert attributes slice to map
		attributesMap := make(map[string]string)
		for _, attribute := range attributes {
			attributesMap[attribute.AttKey] = attribute.AttValue
		}

		// Resolve purposes with all purposes
		purposes, err := consentService.getResolvedConsentPurposes(ctx, consent.ConsentID, orgID)
		if err != nil {
			logger.Error("Failed to resolve purposes", log.Error(err))
			return nil, serviceerror.CustomServiceError(ErrorInternalServerError, err.Error())
		} else {
			// Build complete consent response
			consentResponse := buildConsentResponse(consent, purposes, attributesMap, authResources)
			// Convert to ValidateConsentAPIResponse with enriched purpose details
			response.ConsentInformation = consentService.EnrichedValidateConsentAPIResponse(ctx, consentResponse, orgID)

			// Check if all mandatory purposes are approved (only if no previous errors)
			if response.ErrorCode == 0 {
				unapprovedMandatoryPurposes := make([]string, 0)
				for _, purpose := range purposes {
					for _, element := range purpose.Elements {
						if element.IsMandatory && !element.IsUserApproved {
							unapprovedMandatoryPurposes = append(unapprovedMandatoryPurposes, element.ElementName)
						}
					}
				}

				if len(unapprovedMandatoryPurposes) > 0 {
					response.ErrorCode = 403
					response.ErrorMessage = "mandatory_purposes_not_approved"
					response.ErrorDescription = fmt.Sprintf("The following mandatory purposes are not approved: %v", unapprovedMandatoryPurposes)
					logger.Warn("Mandatory purposes not approved",
						log.String("consent_id", req.ConsentID),
						log.Int("unapproved_count", len(unapprovedMandatoryPurposes)),
						log.Any("unapproved_purposes", unapprovedMandatoryPurposes))
				}
			}
		}
	}

	// If no errors, mark as valid
	if response.ErrorCode == 0 {
		response.IsValid = true
		logger.Info("Consent validation successful",
			log.String("consent_id", req.ConsentID),
			log.Bool("is_valid", true))
	} else {
		logger.Warn("Consent validation failed",
			log.String("consent_id", req.ConsentID),
			log.Bool("is_valid", false),
			log.Int("error_code", response.ErrorCode),
			log.String("error_message", response.ErrorMessage))
	}

	return response, nil
}

// expireConsent updates consent and all related auth resources to expired status
func (consentService *consentService) expireConsent(ctx context.Context, consent *model.Consent, orgID string) error {
	logger := log.GetLogger().WithContext(ctx)
	logger.Debug("Expiring consent",
		log.String("consent_id", consent.ConsentID),
		log.String("org_id", orgID))

	expiredStatusName := string(config.Get().Consent.GetExpiredConsentStatus())
	currentTime := utils.GetCurrentTimeMillis()

	// Create audit entry
	auditID := utils.GenerateUUID()
	reason := "Consent expired based on validityTime"
	actionBy := "SYSTEM"
	previousStatus := consent.CurrentStatus
	audit := &model.ConsentStatusAudit{
		StatusAuditID:  auditID,
		ConsentID:      consent.ConsentID,
		CurrentStatus:  expiredStatusName,
		ActionTime:     currentTime,
		Reason:         &reason,
		ActionBy:       &actionBy,
		PreviousStatus: &previousStatus,
		OrgID:          orgID,
	}

	// Get stores for cascading status update
	consentStore := consentService.stores.Consent
	authResourceStore := consentService.stores.AuthResource

	// Execute transaction - update consent status, all auth resource statuses, and create audit
	err := consentService.stores.ExecuteTransaction([]func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return consentStore.UpdateStatus(tx, consent.ConsentID, orgID, expiredStatusName, currentTime)
		},
		func(tx dbmodel.TxInterface) error {
			// Update all authorization statuses to system expired status when consent expires
			sysExpiredStatus := string(config.Get().Consent.GetSystemExpiredAuthStatus())
			return authResourceStore.UpdateAllStatusByConsentID(tx, consent.ConsentID, orgID, sysExpiredStatus, currentTime)
		},
		func(tx dbmodel.TxInterface) error {
			return consentStore.CreateStatusAudit(tx, audit)
		},
	})
	if err != nil {
		logger.Error("Failed to expire consent in transaction",
			log.Error(err),
			log.String("consent_id", consent.ConsentID))
		return err
	}

	// Update local consent object
	consent.CurrentStatus = expiredStatusName
	consent.UpdatedTime = currentTime

	logger.Debug("Consent expired successfully",
		log.String("consent_id", consent.ConsentID),
		log.String("new_status", expiredStatusName))

	return nil
}

// EnrichedValidateConsentAPIResponse builds ValidateConsentAPIResponse with enriched purpose details (type, description, attributes, isMandatory)
func (consentService *consentService) EnrichedValidateConsentAPIResponse(ctx context.Context, consent *model.ConsentResponse, orgID string) *model.ValidateConsentAPIResponse {
	logger := log.GetLogger().WithContext(ctx)

	// Check for nil consent first before dereferencing
	if consent == nil {
		logger.Debug("Consent is nil, returning nil", log.String("org_id", orgID))
		return nil
	}

	logger.Debug("Building enriched validate response with purpose details",
		log.String("consent_id", consent.ConsentID),
		log.String("org_id", orgID))

	consentElementStore := consentService.stores.ConsentElement

	// Build base response
	validateResponse := &model.ValidateConsentAPIResponse{
		ID:                         consent.ConsentID,
		Type:                       consent.ConsentType,
		ClientID:                   consent.ClientID,
		Status:                     consent.CurrentStatus,
		CreatedTime:                consent.CreatedTime,
		UpdatedTime:                consent.UpdatedTime,
		ValidityTime:               consent.ValidityTime,
		RecurringIndicator:         consent.RecurringIndicator,
		Frequency:                  consent.ConsentFrequency,
		DataAccessValidityDuration: consent.DataAccessValidityDuration,
		Attributes:                 consent.Attributes,
	}

	// Convert authorizations
	if len(consent.AuthResources) > 0 {
		validateResponse.Authorizations = make([]model.AuthorizationAPIResponse, 0, len(consent.AuthResources))
		for _, auth := range consent.AuthResources {
			// Parse resources JSON string to interface
			var resources interface{}
			if auth.Resources != nil && *auth.Resources != "" {
				if err := json.Unmarshal([]byte(*auth.Resources), &resources); err != nil {
					// If parsing fails, set to empty object
					resources = make(map[string]interface{})
				}
			} else {
				// If resources is nil or empty, set to empty object
				resources = make(map[string]interface{})
			}

			validateResponse.Authorizations = append(validateResponse.Authorizations, model.AuthorizationAPIResponse{
				ID:          auth.AuthID,
				UserID:      auth.UserID,
				Type:        auth.AuthType,
				Status:      auth.AuthStatus,
				UpdatedTime: auth.UpdatedTime,
				Resources:   resources,
			})
		}
	}

	// Enrich purposes with full element details (type, description, properties, isMandatory, isApproved, value)
	if len(consent.Purposes) > 0 {
		enrichedPurposes := make([]model.ConsentPurposeItemValidate, 0, len(consent.Purposes))

		for _, purposeItem := range consent.Purposes {
			enrichedPurposeItem := model.ConsentPurposeItemValidate{
				PurposeName: purposeItem.PurposeName,
				Elements:    make([]model.ConsentElementApprovalItemValidate, 0, len(purposeItem.Elements)),
			}

			for _, elementItem := range purposeItem.Elements {
				enrichedElement := model.ConsentElementApprovalItemValidate{
					ElementName:    elementItem.ElementName,
					IsUserApproved: elementItem.IsUserApproved,
					Value:          elementItem.Value,
					IsMandatory:    elementItem.IsMandatory,
				}

				// Fetch full element details from consent element service
				if elementItem.ElementName != "" {
					element, err := consentElementStore.GetByName(ctx, elementItem.ElementName, orgID)
					if err == nil && element != nil {
						// Enrich with element details
						enrichedElement.Type = element.Type

						// Dereference description pointer if not nil
						if element.Description != nil {
							enrichedElement.Description = *element.Description
						}

						// Fetch properties from CONSENT_ELEMENT_PROPERTY table
						properties, propErr := consentElementStore.GetPropertiesByElementID(ctx, element.ID, orgID)
						if propErr == nil && len(properties) > 0 {
							enrichedElement.Properties = make(map[string]interface{})
							for _, prop := range properties {
								enrichedElement.Properties[prop.Key] = prop.Value
							}
						}

						logger.Debug("Element details enriched for validate",
							log.String("element", elementItem.ElementName),
							log.String("type", element.Type),
							log.String("description", enrichedElement.Description),
							log.Bool("isMandatory", enrichedElement.IsMandatory),
							log.Int("properties_count", len(enrichedElement.Properties)))
					} else if err != nil {
						logger.Warn("Failed to fetch element details",
							log.String("element", elementItem.ElementName),
							log.Error(err))
					} else {
						logger.Warn("Element not found in database",
							log.String("element", elementItem.ElementName),
							log.String("org_id", orgID))
					}
				}

				enrichedPurposeItem.Elements = append(enrichedPurposeItem.Elements, enrichedElement)
			}

			enrichedPurposes = append(enrichedPurposes, enrichedPurposeItem)
		}

		// Set enriched purposes
		validateResponse.Purposes = enrichedPurposes
	}

	logger.Debug("Validate response enriched successfully",
		log.Int("purpose_count", len(validateResponse.Purposes)))

	return validateResponse
}

// buildConsentResponse constructs a complete ConsentResponse from already-resolved data.
// This is a pure data transformation function that takes pre-fetched data and builds the response.
// No database access is performed here - all data must be provided as parameters.
// This makes the function easily testable and free of side effects.
func buildConsentResponse(
	consent *model.Consent,
	purposes []model.ConsentPurposeItem,
	attributes map[string]string,
	authResources []authmodel.AuthResource,
) *model.ConsentResponse {
	// AuthResource is already a type alias for ConsentAuthResource - use directly
	authResourcesResp := authResources

	return &model.ConsentResponse{
		ConsentID:                  consent.ConsentID,
		Purposes:                   purposes,
		CreatedTime:                consent.CreatedTime,
		UpdatedTime:                consent.UpdatedTime,
		ClientID:                   consent.ClientID,
		ConsentType:                consent.ConsentType,
		CurrentStatus:              consent.CurrentStatus,
		ConsentFrequency:           consent.ConsentFrequency,
		ValidityTime:               consent.ValidityTime,
		RecurringIndicator:         consent.RecurringIndicator,
		DataAccessValidityDuration: consent.DataAccessValidityDuration,
		OrgID:                      consent.OrgID,
		Attributes:                 attributes,
		AuthResources:              authResourcesResp,
	}
}

// GetConsentDelegates returns all registered delegates for a consent.
// It reads CONSENT_AUTH_RESOURCE rows (RESOURCES JSON blob for per-delegate flags)
// and CONSENT_ATTRIBUTE rows (for delegation.principal_id, guardian.valid_until, etc.).

func (consentService *consentService) GetConsentDelegates(ctx context.Context, consentID, orgID string) (*model.DelegateListResponse, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Info("Getting consent delegates",
		log.String("consent_id", consentID),
		log.String("org_id", orgID))

	// Verify consent exists.
	store := consentService.stores.Consent
	existing, err := store.GetByID(ctx, consentID, orgID)
	if err != nil {
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, err.Error())
	}
	if existing == nil {
		return nil, serviceerror.CustomServiceError(ErrorConsentNotFound,
			fmt.Sprintf("consent with ID '%s' not found", consentID))
	}

	// Load attributes to read delegation metadata.
	attributes, err := store.GetAttributesByConsentID(ctx, consentID, orgID)
	if err != nil {
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, err.Error())
	}
	attMap := make(map[string]string)
	for _, a := range attributes {
		attMap[a.AttKey] = a.AttValue
	}
	delegCfg := parseDelegationConfig(attMap)

	// Load all auth resource rows for this consent.
	authResources, err := consentService.stores.AuthResource.GetByConsentID(ctx, consentID, orgID)
	if err != nil {
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, err.Error())
	}

	// Build DelegateInfo list from auth resources that carry onBehalfOf in RESOURCES.
	delegates := make([]model.DelegateInfo, 0)
	for _, ar := range authResources {
		if ar.Resources == nil || *ar.Resources == "" {
			continue
		}
		var res map[string]interface{}
		if json.Unmarshal([]byte(*ar.Resources), &res) != nil {
			continue
		}
		onBehalfOf, _ := res["onBehalfOf"].(string)
		if onBehalfOf == "" {
			continue // not a delegate auth resource
		}
		userID := ""
		if ar.UserID != nil {
			userID = *ar.UserID
		}
		canRevoke, _ := res["canRevoke"].(bool)
		canModify, _ := res["canModify"].(bool)
		delegationType, _ := res["delegationType"].(string)

		delegates = append(delegates, model.DelegateInfo{
			AuthID:         ar.AuthID,
			UserID:         userID,
			DelegationType: delegationType,
			Status:         ar.AuthStatus,
			CanRevoke:      canRevoke,
			CanModify:      canModify,
			OnBehalfOf:     onBehalfOf,
			UpdatedTime:    ar.UpdatedTime,
		})
	}

	response := &model.DelegateListResponse{
		ConsentID:           consentID,
		DataPrincipalID:     delegCfg.PrincipalID,
		RevocationPolicy:    string(delegCfg.RevocationPolicy),
		ValidUntil:          delegCfg.ValidUntil,
		IsDelegationExpired: delegCfg.IsExpired(),
		DelegateCount:       len(delegates),
		Delegates:           delegates,
	}

	logger.Info("Consent delegates retrieved",
		log.Int("delegate_count", len(delegates)),
		log.String("consent_id", consentID))

	return response, nil
}

// SearchConsentsByAttribute searches for consents by attribute key and optionally value
// If value is empty, it searches by key only
func (consentService *consentService) SearchConsentsByAttribute(ctx context.Context, key, value, orgID string) (*model.ConsentAttributeSearchResponse, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Info("Searching consents by attribute",
		log.String("key", key),
		log.String("value", value),
		log.String("org_id", orgID))

	consentStore := consentService.stores.Consent

	var consentIDs []string
	var err error

	// If value is provided and not empty, search by key-value pair
	// Otherwise, search by key only
	if value != "" {
		consentIDs, err = consentStore.FindConsentIDsByAttribute(ctx, key, value, orgID)
	} else {
		consentIDs, err = consentStore.FindConsentIDsByAttributeKey(ctx, key, orgID)
	}

	if err != nil {
		logger.Error("Failed to search consents by attribute",
			log.Error(err),
			log.String("key", key),
			log.String("value", value))
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, err.Error())
	}

	logger.Info("Consents searched by attribute successfully",
		log.Int("count", len(consentIDs)))

	return &model.ConsentAttributeSearchResponse{
		ConsentIDs: consentIDs,
		Count:      len(consentIDs),
	}, nil
}

// getResolvedConsentPurposes fetches purposes for a consent and resolves all purposes.
// This is a generic method that constructs the complete purpose structure with all purposes:
// 1. Fetches purpose mappings from PURPOSE_CONSENT_MAPPING table
// 2. For each linked purpose, fetches ALL elements defined in that purpose from DB
// 3. Creates a fresh map with default values (isUserApproved=false, value=nil)
// 4. Fetches approval records and uses them ONLY to update approval values
// Returns fully resolved purposes ready for response serialization.
func (s *consentService) getResolvedConsentPurposes(
	ctx context.Context,
	consentID, orgID string,
) ([]model.ConsentPurposeItem, error) {
	logger := log.GetLogger().WithContext(ctx)

	consentStore := s.stores.Consent
	purposeStore := s.stores.ConsentPurpose

	// Step 1: Fetch purpose mappings from PURPOSE_CONSENT_MAPPING table
	// This is the source of truth for which purposes are linked to this consent
	purposeMappings, err := consentStore.GetConsentPurposeMappingsByConsentID(ctx, consentID, orgID)
	if err != nil {
		logger.Error("Failed to fetch purpose mappings",
			log.String("consent_id", consentID),
			log.Error(err))
		return nil, fmt.Errorf("failed to fetch purpose mappings: %w", err)
	}

	// If no purposes are mapped, return empty list
	if len(purposeMappings) == 0 {
		logger.Debug("No purposes mapped to consent",
			log.String("consent_id", consentID))
		return []model.ConsentPurposeItem{}, nil
	}

	logger.Debug("Identified purposes from mappings",
		log.String("consent_id", consentID),
		log.Int("purpose_count", len(purposeMappings)))

	// Step 2: For each mapped purpose, fetch ALL elements defined in that purpose
	// and build a fresh map with default values
	purposesMap := make(map[string]*model.ConsentPurposeItem)

	for _, mapping := range purposeMappings {
		purposeID := mapping.PurposeID
		purposeName := mapping.PurposeName

		// Fetch all elements in this purpose from database
		purposeElements, err := purposeStore.GetPurposeElements(ctx, purposeID, orgID)
		if err != nil {
			logger.Error("Failed to fetch elements for purpose",
				log.String("purpose_name", purposeName),
				log.String("purpose_id", purposeID),
				log.Error(err))
			return nil, fmt.Errorf("failed to fetch elements for purpose '%s': %w", purposeName, err)
		}

		// Initialize purpose with all elements having default values
		purposesMap[purposeName] = &model.ConsentPurposeItem{
			PurposeName: purposeName,
			Elements:    make([]model.ConsentElementApprovalItem, 0, len(purposeElements)),
		}

		// Add all elements with default values (isUserApproved=false, value=nil, isMandatory from purpose definition)
		for _, elem := range purposeElements {
			purposesMap[purposeName].Elements = append(
				purposesMap[purposeName].Elements,
				model.ConsentElementApprovalItem{
					ElementName:    elem.ElementName,
					IsUserApproved: false,            // Default: not approved
					Value:          nil,              // Default: no value
					IsMandatory:    elem.IsMandatory, // From purpose definition
				},
			)
		}

		logger.Debug("Initialized purpose with default values",
			log.String("purpose_name", purposeName),
			log.Int("purpose_count", len(purposeElements)))
	}

	// Step 3: Fetch approval records from CONSENT_ELEMENT_APPROVAL table
	// These are used ONLY to update the approval status and values
	elementApprovals, err := consentStore.GetPurposeElementApprovalsByConsentID(ctx, consentID, orgID)
	if err != nil {
		logger.Error("Failed to fetch element approvals",
			log.String("consent_id", consentID),
			log.Error(err))
		return nil, fmt.Errorf("failed to fetch element approvals: %w", err)
	}

	// Step 4: Update the map with actual approval values from database
	for _, approval := range elementApprovals {
		// Parse value from JSON string
		var value interface{}
		if approval.Value != nil && *approval.Value != "" {
			if err := json.Unmarshal([]byte(*approval.Value), &value); err != nil {
				logger.Warn("Failed to unmarshal element value",
					log.String("element", approval.ElementName),
					log.Error(err))
			}
		}

		// Find and update the element in the purpose
		if purp, exists := purposesMap[approval.PurposeName]; exists {
			for i := range purp.Elements {
				if purp.Elements[i].ElementName == approval.ElementName {
					// Update with actual approval values from DB
					purp.Elements[i].IsUserApproved = approval.IsUserApproved
					purp.Elements[i].Value = value
					break
				}
			}
		}
	}

	logger.Debug("Updated elements with approval values",
		log.String("consent_id", consentID),
		log.Int("approval_count", len(elementApprovals)))

	// Step 5: Convert map to slice for response
	purposes := make([]model.ConsentPurposeItem, 0, len(purposesMap))
	for _, purpose := range purposesMap {
		purposes = append(purposes, *purpose)
	}

	logger.Debug("Resolved consent purposes",
		log.String("consent_id", consentID),
		log.Int("purpose_count", len(purposes)))

	return purposes, nil
}

// validateNoDuplicateElementsAcrossPurposes ensures no element appears in multiple purposes.
// This validation is called AFTER purpose resolution from database, so it checks the
// complete set of elements (including auto-filled ones), not just what user provided.
// This prevents an element from being assigned to multiple purposes, which would create

func (s *consentService) validateNoDuplicateElementsAcrossPurposes(
	purposes []model.ConsentPurposeCreateRequest,
) error {
	// Track which parent purpose each element belongs to
	elementNamesSeen := make(map[string]string) // element name -> parent purpose name

	for _, purpose := range purposes {
		for _, element := range purpose.Elements {
			if existingPurpose, found := elementNamesSeen[element.ElementName]; found {
				// Found duplicate - same element in multiple purposes
				return fmt.Errorf(
					"duplicate element '%s' found in purposes '%s' and '%s'",
					element.ElementName,
					existingPurpose,
					purpose.PurposeName,
				)
			}
			elementNamesSeen[element.ElementName] = purpose.PurposeName
		}
	}

	return nil
}

// validatePurposes validates purposes and resolves all elements from database.
// This method:
// 1. Fetches purpose definitions from DB (validates purpose existence)
// 2. Fetches all elements within each purpose from DB
// 3. Validates that user-provided elements belong to their respective purposes
// 4. Resolves missing purposes (not in request) with isUserApproved=false
// 5. Validates no duplicate elements exist across all resolved elements in purposes

func (s *consentService) validatePurposes(
	ctx context.Context,
	purposes []model.ConsentPurposeCreateRequest,
	clientID, orgID string,
) ([]model.ConsentPurposeCreateRequest, error) {

	logger := log.GetLogger().WithContext(ctx)

	// Resolve purposes and get their full definitions from database
	resolvedPurposes := make([]model.ConsentPurposeCreateRequest, 0, len(purposes))
	purposeStore := s.stores.ConsentPurpose

	for _, requestedIndividualPurpose := range purposes {
		// Fetch purpose metadata by name for this client
		// Note: Using limit=1 since purpose names should be unique per client
		retrievedPurposes, _, err := purposeStore.ListPurposes(ctx, orgID, requestedIndividualPurpose.PurposeName, []string{clientID}, nil, 0, 1)
		if err != nil {
			logger.Error("Failed to fetch purpose from database",
				log.String("purpose_name", requestedIndividualPurpose.PurposeName),
				log.String("client_id", clientID),
				log.Error(err))
			return nil, fmt.Errorf("failed to get purpose '%s': %w", requestedIndividualPurpose.PurposeName, err)
		}

		if len(retrievedPurposes) == 0 {
			logger.Warn("Purpose not found",
				log.String("purpose_name", requestedIndividualPurpose.PurposeName),
				log.String("client_id", clientID))
			return nil, fmt.Errorf("purpose '%s' not found for client '%s'", requestedIndividualPurpose.PurposeName, clientID)
		}

		retrievedPurpose := retrievedPurposes[0]
		logger.Debug("Purpose found",
			log.String("purpose_name", requestedIndividualPurpose.PurposeName),
			log.String("purpose_id", retrievedPurpose.ID))

		// Get ALL elements defined in this purpose.
		// This gives us the complete list of elements with their IDs, names, and mandatory flags
		purposeElementsFromDB := retrievedPurpose.Elements

		logger.Debug("Fetched elements for purpose",
			log.String("purpose_name", requestedIndividualPurpose.PurposeName),
			log.Int("total_elements", len(purposeElementsFromDB)),
			log.Int("requested_elements", len(requestedIndividualPurpose.Elements)))

		// Create lookup map for elements provided in the request
		// Key: element name, Value: approval details from request
		requestedElementsMap := make(map[string]model.ConsentElementApprovalCreateRequest)
		for _, requestedElements := range requestedIndividualPurpose.Elements {
			requestedElementsMap[requestedElements.ElementName] = requestedElements
		}

		// Create lookup map for valid elements from database
		// This is used to validate that user-provided elements actually belong to this purpose
		validElementNames := make(map[string]bool)
		for _, elem := range purposeElementsFromDB {
			validElementNames[elem.ElementName] = true
		}

		// Step 5: Validate that all requested elements belong to this purpose
		for elementName := range requestedElementsMap {
			if !validElementNames[elementName] {
				logger.Warn("Element does not belong to purpose",
					log.String("purpose_name", elementName),
					log.String("purpose_name", requestedIndividualPurpose.PurposeName))
				return nil, fmt.Errorf("purpose '%s' does not belong to purpose '%s'", elementName, requestedIndividualPurpose.PurposeName)
			}
		}

		// Resolve ALL elements in the purpose (merge requested + missing)
		// For elements in request: use their approval status and values
		// For elements not in request: add with isUserApproved=false (user didn't approve)
		allElements := make([]model.ConsentElementApprovalCreateRequest, 0, len(purposeElementsFromDB))

		for _, dbElement := range purposeElementsFromDB {
			if requestedElement, found := requestedElementsMap[dbElement.ElementName]; found {
				// Element was explicitly provided in request - use user's approval status
				requestedElement.ElementID = dbElement.ElementID
				requestedElement.IsMandatory = dbElement.IsMandatory
				allElements = append(allElements, requestedElement)
				logger.Debug("Using requested element approval",
					log.String("element", dbElement.ElementName),
					log.Bool("approved", requestedElement.IsUserApproved))
			} else {
				// Element was NOT in request - auto-fill with isUserApproved=false
				allElements = append(allElements, model.ConsentElementApprovalCreateRequest{
					ElementID:      dbElement.ElementID,
					ElementName:    dbElement.ElementName,
					IsUserApproved: false, // Not approved since user didn't provide it
					Value:          nil,
					IsMandatory:    dbElement.IsMandatory,
				})
				logger.Debug("Auto-filling missing element",
					log.String("element", dbElement.ElementName),
					log.Bool("approved", false))
			}
		}

		// Add fully resolved purpose to result
		resolvedPurposes = append(resolvedPurposes, model.ConsentPurposeCreateRequest{
			PurposeName: requestedIndividualPurpose.PurposeName,
			PurposeID:   retrievedPurpose.ID,
			Elements:    allElements, // Contains ALL elements (requested + auto-filled)
		})
	}

	// Validate no duplicate elements across ALL resolved purposes
	// This check happens AFTER resolution because we now have the complete picture
	// of all elements across all purposes (including auto-filled ones)
	if err := s.validateNoDuplicateElementsAcrossPurposes(resolvedPurposes); err != nil {
		logger.Warn("Duplicate element validation failed", log.Error(err))
		return nil, err
	}

	logger.Info("Purposes validated and resolved successfully",
		log.Int("purpose_count", len(resolvedPurposes)))

	return resolvedPurposes, nil
}
