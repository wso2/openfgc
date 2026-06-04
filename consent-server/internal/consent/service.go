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
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"

	jsonschema "github.com/google/jsonschema-go/jsonschema"
	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	authmodel "github.com/wso2/openfgc/internal/authresource/model"
	"github.com/wso2/openfgc/internal/consent/model"
	"github.com/wso2/openfgc/internal/consent/validator"
	purposemodel "github.com/wso2/openfgc/internal/consentpurpose/model"
	"github.com/wso2/openfgc/internal/system/config"
	dbmodel "github.com/wso2/openfgc/internal/system/database/model"
	"github.com/wso2/openfgc/internal/system/error/serviceerror"
	"github.com/wso2/openfgc/internal/system/log"
	"github.com/wso2/openfgc/internal/system/stores"
	"github.com/wso2/openfgc/internal/system/utils"
)

// =============================================================================
// Internal types
// =============================================================================

// resolvedPurposeLink is the result of validating one purpose from a consent request.
// purposeVersionID is the specific version this consent is created against.
// approvals are the element approval records ready for DB insertion (ConsentID is not yet set).
type resolvedPurposeLink struct {
	purposeVersionID string
	approvals        []model.ConsentElementApproval
}

// =============================================================================
// Service interface and constructor
// =============================================================================

// ConsentService defines the exported service interface.
type ConsentService interface {
	CreateConsent(ctx context.Context, input model.CreateConsentInput, orgID string) (*model.ConsentOutput, *serviceerror.ServiceError)
	GetConsent(ctx context.Context, consentID, orgID string) (*model.ConsentOutput, *serviceerror.ServiceError)
	SearchConsents(ctx context.Context, filters model.ConsentSearchFilter) (*model.ConsentListOutput, *serviceerror.ServiceError)
	UpdateConsent(ctx context.Context, consentID, groupID, orgID string, input model.UpdateConsentInput) (*model.ConsentOutput, *serviceerror.ServiceError)
	RevokeConsent(ctx context.Context, consentID, orgID string, input model.ConsentRevokeInput) (*model.ConsentRevokeOutput, *serviceerror.ServiceError)
	ValidateConsent(ctx context.Context, input model.ConsentValidateInput, orgID string) (*model.ConsentValidateOutput, *serviceerror.ServiceError)
	SearchConsentsByAttribute(ctx context.Context, key, value, orgID string) (*model.ConsentAttributeSearchOutput, *serviceerror.ServiceError)
	GetExpiredConsents(ctx context.Context, currentTimeMs int64, expirableStatuses []string) ([]model.Consent, *serviceerror.ServiceError)
	ExpireConsent(ctx context.Context, consent *model.Consent, orgID string) *serviceerror.ServiceError
}

// consentService implements ConsentService.
type consentService struct {
	stores *stores.StoreRegistry
}

// newConsentService creates a new consent service.
func newConsentService(registry *stores.StoreRegistry) ConsentService {
	return &consentService{stores: registry}
}

// =============================================================================
// CreateConsent
// =============================================================================

// CreateConsent creates a new consent with all related entities in a single transaction.
func (s *consentService) CreateConsent(ctx context.Context, input model.CreateConsentInput, orgID string) (*model.ConsentOutput, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Info("Creating consent",
		log.String("group_id", input.GroupID),
		log.String("org_id", orgID),
		log.String("consent_type", input.ConsentType))

	// Resolve and validate all purposes
	var resolvedLinks []resolvedPurposeLink
	if len(input.Purposes) > 0 {
		var err error
		resolvedLinks, err = s.validatePurposesAndResolve(ctx, input.Purposes, input.GroupID, orgID)
		if err != nil {
			logger.Error("Purpose resolution failed", log.Error(err))
			return nil, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error())
		}
	}

	// Derive consent status from authorization states (empty status → approved → active)
	authStatuses := make([]string, 0, len(input.Authorizations))
	for _, ar := range input.Authorizations {
		authStatuses = append(authStatuses, ar.AuthStatus)
	}
	consentStatus := validator.EvaluateConsentStatusFromAuthStatuses(authStatuses)

	consentID := utils.GenerateUUID()
	currentTime := utils.GetCurrentTimeMillis()
	logger.Debug("Generated consent ID", log.String("consent_id", consentID))

	consent := &model.Consent{
		ConsentID:                  consentID,
		CreatedTime:                currentTime,
		UpdatedTime:                currentTime,
		GroupID:                    input.GroupID,
		ConsentType:                input.ConsentType,
		CurrentStatus:              consentStatus,
		ConsentFrequency:           input.ConsentFrequency,
		ExpirationTime:             input.ExpirationTime,
		RecurringIndicator:         input.RecurringIndicator,
		DataAccessValidityDuration: input.DataAccessValidityDuration,
		OrgID:                      orgID,
	}

	consentStore := s.stores.Consent
	authResourceStore := s.stores.AuthResource

	queries := []func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error { return consentStore.Create(tx, consent) },
	}

	// Attributes
	if len(input.Attributes) > 0 {
		attrs := make([]model.ConsentAttribute, 0, len(input.Attributes))
		for k, v := range input.Attributes {
			attrs = append(attrs, model.ConsentAttribute{
				ConsentID: consentID,
				AttKey:    k,
				AttValue:  v,
				OrgID:     orgID,
			})
		}
		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return consentStore.CreateAttributes(tx, attrs)
		})
	}

	// Status audit
	reason := "Initial consent creation"
	actionBy := input.GroupID
	audit := &model.ConsentStatusAudit{
		StatusAuditID:  utils.GenerateUUID(),
		ConsentID:      consentID,
		CurrentStatus:  consentStatus,
		ActionTime:     currentTime,
		Reason:         &reason,
		ActionBy:       &actionBy,
		PreviousStatus: nil,
		OrgID:          orgID,
	}
	queries = append(queries, func(tx dbmodel.TxInterface) error {
		return consentStore.CreateStatusAudit(tx, audit)
	})

	// Authorization resources
	defaultAuthStatus := string(config.Get().Consent.GetApprovedAuthStatus())
	for _, authInput := range input.Authorizations {
		ar := buildAuthResource(authInput, consentID, orgID, currentTime, defaultAuthStatus)
		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return authResourceStore.Create(tx, ar)
		})
	}

	// Purpose version links and element approvals
	for _, link := range resolvedLinks {
		pvID := link.purposeVersionID
		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return consentStore.LinkPurposeVersionToConsent(tx, consentID, pvID, orgID)
		})
		for _, approval := range link.approvals {
			a := approval
			a.ConsentID = consentID
			queries = append(queries, func(tx dbmodel.TxInterface) error {
				return consentStore.CreateElementApproval(tx, &a)
			})
		}
	}

	if err := s.stores.ExecuteTransaction(queries); err != nil {
		logger.Error("Create consent transaction failed",
			log.Error(err),
			log.String("consent_id", consentID))
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError,
			fmt.Sprintf("failed to create consent: %v", err))
	}

	logger.Info("Consent created successfully", log.String("consent_id", consentID))

	// Check if the consent was created already-expired and update status accordingly
	expiredStatus := string(config.Get().Consent.GetExpiredConsentStatus())
	if consent.ExpirationTime != nil && validator.IsConsentExpired(*consent.ExpirationTime) {
		if consent.CurrentStatus != expiredStatus {
			if err := s.ExpireConsent(ctx, consent, orgID); err != nil {
				logger.Error("Failed to expire consent after creation", log.Error(err))
			} else if refreshed, err := consentStore.GetByID(ctx, consentID, orgID); err == nil && refreshed != nil {
				consent = refreshed
			}
		}
	}

	out, err := s.loadConsentOutput(ctx, consent, orgID)
	if err != nil {
		logger.Error("Failed to load consent output after creation", log.Error(err))
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, err.Error())
	}
	return out, nil
}

// =============================================================================
// GetConsent
// =============================================================================

// GetConsent retrieves a consent by ID with all related data.
func (s *consentService) GetConsent(ctx context.Context, consentID, orgID string) (*model.ConsentOutput, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Debug("Retrieving consent",
		log.String("consent_id", consentID),
		log.String("org_id", orgID))

	consentStore := s.stores.Consent

	consent, err := consentStore.GetByID(ctx, consentID, orgID)
	if err != nil {
		logger.Error("Failed to retrieve consent", log.Error(err), log.String("consent_id", consentID))
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, err.Error())
	}
	if consent == nil {
		return nil, serviceerror.CustomServiceError(ErrorConsentNotFound,
			fmt.Sprintf("consent with ID '%s' not found", consentID))
	}

	// Auto-expire if the consent has passed its expiration time
	expiredStatus := string(config.Get().Consent.GetExpiredConsentStatus())
	if consent.ExpirationTime != nil && validator.IsConsentExpired(*consent.ExpirationTime) {
		if consent.CurrentStatus != expiredStatus {
			if err := s.ExpireConsent(ctx, consent, orgID); err != nil {
				logger.Error("Failed to expire consent on get", log.Error(err))
			} else if refreshed, err := consentStore.GetByID(ctx, consentID, orgID); err == nil && refreshed != nil {
				consent = refreshed
			}
		}
	}

	out, err := s.loadConsentOutput(ctx, consent, orgID)
	if err != nil {
		logger.Error("Failed to load consent output", log.Error(err))
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, err.Error())
	}

	logger.Debug("Consent retrieved",
		log.String("consent_id", consentID),
		log.String("status", consent.CurrentStatus))
	return out, nil
}

// =============================================================================
// SearchConsents
// =============================================================================

// SearchConsents retrieves consents matching the filters with full detail.
func (s *consentService) SearchConsents(ctx context.Context, filters model.ConsentSearchFilter) (*model.ConsentListOutput, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Info("Searching consents",
		log.String("org_id", filters.OrgID),
		log.Int("limit", filters.Limit))

	if filters.Limit <= 0 {
		filters.Limit = 10
	}
	if filters.Offset < 0 {
		filters.Offset = 0
	}

	consentStore := s.stores.Consent
	authResourceStore := s.stores.AuthResource

	consents, total, err := consentStore.Search(ctx, filters)
	if err != nil {
		logger.Error("Failed to search consents", log.Error(err))
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, err.Error())
	}

	if len(consents) == 0 {
		return &model.ConsentListOutput{
			Data:   []model.ConsentOutput{},
			Total:  0,
			Limit:  filters.Limit,
			Offset: filters.Offset,
			Count:  0,
		}, nil
	}

	// Batch-fetch attributes and auth resources across all result consents
	consentIDs := make([]string, len(consents))
	for i, c := range consents {
		consentIDs[i] = c.ConsentID
	}

	attrsByConsent, err := consentStore.GetAttributesByConsentIDs(ctx, consentIDs, filters.OrgID)
	if err != nil {
		logger.Error("Failed to batch-fetch attributes", log.Error(err))
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, err.Error())
	}

	authResources, err := authResourceStore.GetByConsentIDs(ctx, consentIDs, filters.OrgID)
	if err != nil {
		logger.Error("Failed to batch-fetch auth resources", log.Error(err))
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, err.Error())
	}

	// Group auth resources by consent ID
	authsByConsent := make(map[string][]authmodel.AuthResource, len(consents))
	for _, ar := range authResources {
		authsByConsent[ar.ConsentID] = append(authsByConsent[ar.ConsentID], ar)
	}

	// Build output per consent (purposes/approvals are fetched individually — acceptable N+1)
	outputs := make([]model.ConsentOutput, 0, len(consents))
	for _, c := range consents {
		purposeRows, err := consentStore.GetPurposesByConsentID(ctx, c.ConsentID, filters.OrgID)
		if err != nil {
			logger.Warn("Failed to load purposes for consent",
				log.String("consent_id", c.ConsentID),
				log.Error(err))
			purposeRows = nil
		}
		approvalRows, err := consentStore.GetElementApprovalsByConsentID(ctx, c.ConsentID, filters.OrgID)
		if err != nil {
			logger.Warn("Failed to load approvals for consent",
				log.String("consent_id", c.ConsentID),
				log.Error(err))
			approvalRows = nil
		}

		attrMap := attrsByConsent[c.ConsentID]
		if attrMap == nil {
			attrMap = make(map[string]string)
		}

		consent := c // avoid loop-variable capture for pointer fields
		out := buildConsentOutput(&consent, purposeRows, approvalRows, attrMap, authsByConsent[c.ConsentID], nil, nil)
		outputs = append(outputs, *out)
	}

	logger.Info("Consents searched successfully",
		log.Int("count", len(outputs)),
		log.Int("total", total))

	return &model.ConsentListOutput{
		Data:   outputs,
		Total:  total,
		Limit:  filters.Limit,
		Offset: filters.Offset,
		Count:  len(outputs),
	}, nil
}

// =============================================================================
// UpdateConsent
// =============================================================================

// UpdateConsent updates an existing consent.
func (s *consentService) UpdateConsent(ctx context.Context, consentID, groupID, orgID string, input model.UpdateConsentInput) (*model.ConsentOutput, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Info("Updating consent",
		log.String("consent_id", consentID),
		log.String("group_id", groupID),
		log.String("org_id", orgID))

	consentStore := s.stores.Consent
	authResourceStore := s.stores.AuthResource

	// Fetch existing consent
	existing, err := consentStore.GetByID(ctx, consentID, orgID)
	if err != nil {
		logger.Error("Failed to retrieve consent", log.Error(err))
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, err.Error())
	}
	if existing == nil {
		return nil, serviceerror.CustomServiceError(ErrorConsentNotFound,
			fmt.Sprintf("consent with ID '%s' not found", consentID))
	}

	// Only the owning group may update the consent
	if existing.GroupID != groupID {
		logger.Warn("Group mismatch on update attempt",
			log.String("consent_group", existing.GroupID),
			log.String("request_group", groupID))
		return nil, serviceerror.CustomServiceError(ErrorValidationFailed,
			fmt.Sprintf("group '%s' is not authorized to update consent '%s'", groupID, consentID))
	}

	currentTime := utils.GetCurrentTimeMillis()
	previousStatus := existing.CurrentStatus
	expiredStatus := string(config.Get().Consent.GetExpiredConsentStatus())

	// Compute the effective expiration time after this update (input overrides existing)
	newExpirationTime := existing.ExpirationTime
	if input.ExpirationTime != nil {
		newExpirationTime = input.ExpirationTime
	}

	// Derive new consent status
	var newStatus string
	if input.Authorizations != nil {
		// Authorizations are being replaced — derive status from the incoming auth statuses
		authStatuses := make([]string, 0, len(input.Authorizations))
		for _, ar := range input.Authorizations {
			authStatuses = append(authStatuses, ar.AuthStatus)
		}
		newStatus = validator.EvaluateConsentStatusFromAuthStatuses(authStatuses)
	} else if existing.CurrentStatus == expiredStatus &&
		(newExpirationTime == nil || !validator.IsConsentExpired(*newExpirationTime)) {
		// Consent was expired but the new expiration time is in the future (or removed) —
		// re-derive status from the existing auth resources so the consent is reactivated
		allAuthResources, err := authResourceStore.GetByConsentID(ctx, consentID, orgID)
		if err != nil {
			logger.Error("Failed to fetch auth resources for reactivation", log.Error(err))
			return nil, serviceerror.CustomServiceError(ErrorInternalServerError, err.Error())
		}
		authStatuses := make([]string, 0, len(allAuthResources))
		for _, ar := range allAuthResources {
			authStatuses = append(authStatuses, ar.AuthStatus)
		}
		newStatus = validator.EvaluateConsentStatusFromAuthStatuses(authStatuses)
	} else {
		newStatus = existing.CurrentStatus
	}
	statusChanged := newStatus != previousStatus

	// Merge input fields — preserve existing values for unset fields
	updatedConsent := &model.Consent{
		ConsentID:                  consentID,
		GroupID:                    existing.GroupID,
		OrgID:                      orgID,
		UpdatedTime:                currentTime,
		CurrentStatus:              newStatus,
		ConsentType:                existing.ConsentType,
		ConsentFrequency:           existing.ConsentFrequency,
		ExpirationTime:             existing.ExpirationTime,
		RecurringIndicator:         existing.RecurringIndicator,
		DataAccessValidityDuration: existing.DataAccessValidityDuration,
	}
	if input.ConsentType != "" {
		updatedConsent.ConsentType = input.ConsentType
	}
	if input.ExpirationTime != nil {
		updatedConsent.ExpirationTime = input.ExpirationTime
	}
	if input.ConsentFrequency != nil {
		updatedConsent.ConsentFrequency = input.ConsentFrequency
	}
	if input.RecurringIndicator != nil {
		updatedConsent.RecurringIndicator = input.RecurringIndicator
	}
	if input.DataAccessValidityDuration != nil {
		updatedConsent.DataAccessValidityDuration = input.DataAccessValidityDuration
	}

	queries := []func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error { return consentStore.Update(tx, updatedConsent) },
	}

	// Status audit when status changes
	if statusChanged {
		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return consentStore.UpdateStatus(tx, consentID, orgID, newStatus, currentTime)
		})
		reason := "Consent status updated based on authorization states"
		actionBy := existing.GroupID
		audit := &model.ConsentStatusAudit{
			StatusAuditID:  utils.GenerateUUID(),
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

	// Replace attributes if provided (nil = don't touch; non-nil including empty = replace)
	if input.Attributes != nil {
		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return consentStore.DeleteAttributesByConsentID(tx, consentID, orgID)
		})
		if len(input.Attributes) > 0 {
			attrs := make([]model.ConsentAttribute, 0, len(input.Attributes))
			for k, v := range input.Attributes {
				attrs = append(attrs, model.ConsentAttribute{
					ConsentID: consentID,
					AttKey:    k,
					AttValue:  v,
					OrgID:     orgID,
				})
			}
			queries = append(queries, func(tx dbmodel.TxInterface) error {
				return consentStore.CreateAttributes(tx, attrs)
			})
		}
	}

	// Replace auth resources if provided
	if input.Authorizations != nil {
		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return authResourceStore.DeleteByConsentID(tx, consentID, orgID)
		})
		defaultAuthStatus := string(config.Get().Consent.GetApprovedAuthStatus())
		for _, authInput := range input.Authorizations {
			ar := buildAuthResource(authInput, consentID, orgID, currentTime, defaultAuthStatus)
			queries = append(queries, func(tx dbmodel.TxInterface) error {
				return authResourceStore.Create(tx, ar)
			})
		}
	}

	// Replace purpose links and approvals if purposes provided
	if input.Purposes != nil {
		var resolvedLinks []resolvedPurposeLink
		if len(input.Purposes) > 0 {
			resolvedLinks, err = s.validatePurposesAndResolve(ctx, input.Purposes, existing.GroupID, orgID)
			if err != nil {
				logger.Error("Purpose resolution failed on update", log.Error(err))
				return nil, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error())
			}
		}
		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return consentStore.DeletePurposesByConsentID(tx, consentID, orgID)
		})
		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return consentStore.DeleteElementApprovalsByConsentID(tx, consentID, orgID)
		})
		for _, link := range resolvedLinks {
			pvID := link.purposeVersionID
			queries = append(queries, func(tx dbmodel.TxInterface) error {
				return consentStore.LinkPurposeVersionToConsent(tx, consentID, pvID, orgID)
			})
			for _, approval := range link.approvals {
				a := approval
				a.ConsentID = consentID
				queries = append(queries, func(tx dbmodel.TxInterface) error {
					return consentStore.CreateElementApproval(tx, &a)
				})
			}
		}
	}

	if err := s.stores.ExecuteTransaction(queries); err != nil {
		logger.Error("Update consent transaction failed", log.Error(err), log.String("consent_id", consentID))
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError,
			fmt.Sprintf("failed to update consent: %v", err))
	}

	// Re-fetch to get the DB-authoritative state
	updated, err := consentStore.GetByID(ctx, consentID, orgID)
	if err != nil || updated == nil {
		logger.Error("Failed to re-fetch consent after update", log.Error(err))
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, "failed to retrieve updated consent")
	}

	if updated.ExpirationTime != nil && validator.IsConsentExpired(*updated.ExpirationTime) {
		// Consent has passed its expiration time — expire it
		if updated.CurrentStatus != expiredStatus {
			if err := s.ExpireConsent(ctx, updated, orgID); err != nil {
				logger.Error("Failed to expire consent after update", log.Error(err))
			} else if refreshed, err := consentStore.GetByID(ctx, consentID, orgID); err == nil && refreshed != nil {
				updated = refreshed
			}
		}
	}

	out, err := s.loadConsentOutput(ctx, updated, orgID)
	if err != nil {
		logger.Error("Failed to load consent output after update", log.Error(err))
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, err.Error())
	}

	logger.Info("Consent updated successfully",
		log.String("consent_id", consentID),
		log.String("status", updated.CurrentStatus))
	return out, nil
}

// =============================================================================
// RevokeConsent
// =============================================================================

// RevokeConsent sets the consent status to revoked and cascades to all auth resources.
func (s *consentService) RevokeConsent(ctx context.Context, consentID, orgID string, input model.ConsentRevokeInput) (*model.ConsentRevokeOutput, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Info("Revoking consent",
		log.String("consent_id", consentID),
		log.String("org_id", orgID))

	if input.ActionBy == "" {
		return nil, serviceerror.CustomServiceError(ErrorValidationFailed, "actionBy is required")
	}

	consentStore := s.stores.Consent
	authResourceStore := s.stores.AuthResource

	existing, err := consentStore.GetByID(ctx, consentID, orgID)
	if err != nil {
		logger.Error("Failed to retrieve consent", log.Error(err))
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, err.Error())
	}
	if existing == nil {
		return nil, serviceerror.CustomServiceError(ErrorConsentNotFound,
			fmt.Sprintf("consent with ID '%s' not found", consentID))
	}

	revokedStatus := string(config.Get().Consent.GetRevokedConsentStatus())
	if existing.CurrentStatus == revokedStatus {
		return nil, serviceerror.CustomServiceError(ErrorConsentAlreadyRevoked,
			fmt.Sprintf("consent '%s' is already revoked", consentID))
	}

	currentTime := utils.GetCurrentTimeMillis()
	reason := input.Reason
	actionBy := input.ActionBy
	prevStatus := existing.CurrentStatus
	audit := &model.ConsentStatusAudit{
		StatusAuditID:  utils.GenerateUUID(),
		ConsentID:      consentID,
		CurrentStatus:  revokedStatus,
		ActionTime:     currentTime,
		Reason:         &reason,
		ActionBy:       &actionBy,
		PreviousStatus: &prevStatus,
		OrgID:          orgID,
	}

	sysRevokedStatus := string(config.Get().Consent.GetSystemRevokedAuthStatus())
	err = s.stores.ExecuteTransaction([]func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return consentStore.UpdateStatus(tx, consentID, orgID, revokedStatus, currentTime)
		},
		func(tx dbmodel.TxInterface) error {
			return authResourceStore.UpdateAllStatusByConsentID(tx, consentID, orgID, sysRevokedStatus, currentTime)
		},
		func(tx dbmodel.TxInterface) error {
			return consentStore.CreateStatusAudit(tx, audit)
		},
	})
	if err != nil {
		logger.Error("Revoke consent transaction failed", log.Error(err))
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError,
			fmt.Sprintf("failed to revoke consent: %v", err))
	}

	logger.Info("Consent revoked",
		log.String("consent_id", consentID),
		log.String("previous_status", prevStatus),
		log.String("new_status", revokedStatus))

	return &model.ConsentRevokeOutput{
		ActionTime: currentTime,
		ActionBy:   input.ActionBy,
		Reason:     input.Reason,
	}, nil
}

// =============================================================================
// ValidateConsent
// =============================================================================

// ValidateConsent checks whether a consent is currently valid for data access.
func (s *consentService) ValidateConsent(ctx context.Context, input model.ConsentValidateInput, orgID string) (*model.ConsentValidateOutput, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Info("Validating consent",
		log.String("consent_id", input.ConsentID),
		log.String("org_id", orgID))

	if input.ConsentID == "" {
		return nil, serviceerror.CustomServiceError(ErrorValidationFailed, "consentId is required")
	}

	output := &model.ConsentValidateOutput{IsValid: false}

	consentStore := s.stores.Consent
	consent, err := consentStore.GetByID(ctx, input.ConsentID, orgID)
	if err != nil {
		logger.Error("Failed to retrieve consent", log.Error(err))
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, err.Error())
	}
	if consent == nil {
		return nil, serviceerror.CustomServiceError(ErrorConsentNotFound,
			fmt.Sprintf("consent with ID '%s' not found", input.ConsentID))
	}

	// Auto-expire if past expiration time
	expiredStatus := string(config.Get().Consent.GetExpiredConsentStatus())
	if consent.ExpirationTime != nil && validator.IsConsentExpired(*consent.ExpirationTime) {
		if consent.CurrentStatus != expiredStatus {
			if err := s.ExpireConsent(ctx, consent, orgID); err != nil {
				logger.Warn("Failed to expire consent during validation", log.Error(err))
			} else if refreshed, err := consentStore.GetByID(ctx, input.ConsentID, orgID); err == nil && refreshed != nil {
				consent = refreshed
			}
		}
	}

	// Check consent is in the active status
	activeStatus := string(config.Get().Consent.GetActiveConsentStatus())
	if consent.CurrentStatus != activeStatus {
		output.ErrorCode = 401
		output.ErrorMessage = "invalid_consent_status"
		output.ErrorDescription = fmt.Sprintf("consent status is '%s', expected '%s'", consent.CurrentStatus, activeStatus)
	}

	// Load full consent output; the handler formats the enriched validate response from it
	out, err := s.loadConsentOutput(ctx, consent, orgID)
	if err != nil {
		logger.Error("Failed to load consent output for validation", log.Error(err))
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, err.Error())
	}

	output.ConsentInfo = out

	// Check mandatory elements are approved
	if output.ErrorCode == 0 {
		unapproved := make([]string, 0)
		for _, purpose := range out.Purposes {
			for _, elem := range purpose.Elements {
				if elem.Mandatory && !elem.Approved {
					unapproved = append(unapproved, elem.Name)
				}
			}
		}
		if len(unapproved) > 0 {
			output.ErrorCode = 403
			output.ErrorMessage = "mandatory_elements_not_approved"
			output.ErrorDescription = fmt.Sprintf("the following mandatory elements are not approved: %v", unapproved)
			logger.Warn("Mandatory elements not approved",
				log.String("consent_id", input.ConsentID),
				log.Any("unapproved", unapproved))
		}
	}

	if output.ErrorCode == 0 {
		output.IsValid = true
		logger.Info("Consent validation passed", log.String("consent_id", input.ConsentID))
	} else {
		logger.Warn("Consent validation failed",
			log.String("consent_id", input.ConsentID),
			log.Int("error_code", output.ErrorCode),
			log.String("error_message", output.ErrorMessage))
	}

	return output, nil
}

// =============================================================================
// SearchConsentsByAttribute
// =============================================================================

// SearchConsentsByAttribute searches for consent IDs by attribute key and/or value.
// If value is empty the search is by key only.
func (s *consentService) SearchConsentsByAttribute(ctx context.Context, key, value, orgID string) (*model.ConsentAttributeSearchOutput, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Info("Searching consents by attribute",
		log.String("key", key),
		log.String("value", value),
		log.String("org_id", orgID))

	consentStore := s.stores.Consent

	var consentIDs []string
	var err error
	if value != "" {
		consentIDs, err = consentStore.GetConsentIDsByAttribute(ctx, key, value, orgID)
	} else {
		consentIDs, err = consentStore.GetConsentIDsByAttributeKey(ctx, key, orgID)
	}
	if err != nil {
		logger.Error("Failed to search by attribute", log.Error(err))
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, err.Error())
	}

	logger.Info("Attribute search completed", log.Int("count", len(consentIDs)))
	return &model.ConsentAttributeSearchOutput{
		ConsentIDs: consentIDs,
		Count:      len(consentIDs),
	}, nil
}

// GetExpiredConsents retrieves all consents whose validity time has passed
// and whose status is in the expirable list.
func (s *consentService) GetExpiredConsents(ctx context.Context, currentTimeMs int64, expirableStatuses []string) ([]model.Consent, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)

	consents, err := s.stores.Consent.GetExpiredConsents(ctx, currentTimeMs, expirableStatuses)
	if err != nil {
		logger.Error("Failed to fetch expired consents", log.Error(err))
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, err.Error())
	}

	return consents, nil
}

// ExpireConsent updates consent and all related auth resources to expired status.
func (s *consentService) ExpireConsent(ctx context.Context, consent *model.Consent, orgID string) *serviceerror.ServiceError {
	logger := log.GetLogger().WithContext(ctx)
	expiredStatus := string(config.Get().Consent.GetExpiredConsentStatus())
	sysExpiredAuthStatus := string(config.Get().Consent.GetSystemExpiredAuthStatus())
	currentTime := utils.GetCurrentTimeMillis()

	reason := "Consent expired based on expirationTime"
	actionBy := "SYSTEM"
	prevStatus := consent.CurrentStatus
	audit := &model.ConsentStatusAudit{
		StatusAuditID:  utils.GenerateUUID(),
		ConsentID:      consent.ConsentID,
		CurrentStatus:  expiredStatus,
		ActionTime:     currentTime,
		Reason:         &reason,
		ActionBy:       &actionBy,
		PreviousStatus: &prevStatus,
		OrgID:          orgID,
	}

	consentStore := s.stores.Consent
	authResourceStore := s.stores.AuthResource

	err := s.stores.ExecuteTransaction([]func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return consentStore.UpdateStatus(tx, consent.ConsentID, orgID, expiredStatus, currentTime)
		},
		func(tx dbmodel.TxInterface) error {
			return authResourceStore.UpdateAllStatusByConsentID(tx, consent.ConsentID, orgID, sysExpiredAuthStatus, currentTime)
		},
		func(tx dbmodel.TxInterface) error {
			return consentStore.CreateStatusAudit(tx, audit)
		},
	})
	if err != nil {
		logger.Error("Expire consent transaction failed",
			log.Error(err),
			log.String("consent_id", consent.ConsentID))
		return serviceerror.CustomServiceError(ErrorInternalServerError, err.Error())
	}

	consent.CurrentStatus = expiredStatus
	consent.UpdatedTime = currentTime
	return nil
}

// =============================================================================
// Private helpers
// =============================================================================

// validatePurposesAndResolve validates the purposes in a consent create/update request
// and returns per-purpose version IDs and element approval records ready for insertion.
//
// For each purpose in the request:
//  1. Looks up the logical purpose by name+groupID.
//  2. Resolves the target version (specific or latest).
//  3. Validates that any element (name, namespace) pairs provided in the request belong to that version.
//  4. Builds ConsentElementApproval rows for every element in the version,
//     using request values where provided and defaulting to approved=false otherwise.
//     The same element may appear in multiple purposes; each purpose stores its own approval row.
func (s *consentService) validatePurposesAndResolve(
	ctx context.Context,
	purposes []model.ConsentPurposeInput,
	groupID, orgID string,
) ([]resolvedPurposeLink, error) {
	logger := log.GetLogger().WithContext(ctx)
	purposeStore := s.stores.ConsentPurpose

	links := make([]resolvedPurposeLink, 0, len(purposes))

	for _, pi := range purposes {
		purposeName := pi.PurposeRef.PurposeName

		// 1. Find the purpose: try the consent's group first, then fall back to org-level (GROUP_ID = orgID).
		pv, err := purposeStore.GetByNameAndGroupID(ctx, purposeName, groupID, orgID)
		if err != nil {
			return nil, fmt.Errorf("failed to look up purpose %q: %w", purposeName, err)
		}
		if pv == nil {
			// Org-level purposes have GROUP_ID = orgID and are available to all consents in the org.
			pv, err = purposeStore.GetByNameAndGroupID(ctx, purposeName, orgID, orgID)
			if err != nil {
				return nil, fmt.Errorf("failed to look up purpose %q: %w", purposeName, err)
			}
		}
		if pv == nil {
			return nil, fmt.Errorf("purpose '%s' is not accessible: not found in group '%s' or as an org-level purpose", purposeName, groupID)
		}

		// 2. Resolve the target version (specific version or latest)
		var resolvedPV *purposemodel.PurposeVersion
		if pi.PurposeRef.Version != nil {
			resolvedPV, err = purposeStore.GetVersion(ctx, pv.ID, *pi.PurposeRef.Version, orgID)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch version %d of purpose %q: %w",
					*pi.PurposeRef.Version, purposeName, err)
			}
			if resolvedPV == nil {
				return nil, fmt.Errorf("version %d of purpose '%s' not found",
					*pi.PurposeRef.Version, purposeName)
			}
		} else {
			resolvedPV, err = purposeStore.GetLatestVersion(ctx, pv.ID, orgID)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch latest version of purpose %q: %w", purposeName, err)
			}
			if resolvedPV == nil {
				return nil, fmt.Errorf("purpose '%s' has no versions", purposeName)
			}
		}

		logger.Debug("Resolved purpose version",
			log.String("purpose", purposeName),
			log.String("version_id", resolvedPV.VersionID),
			log.Int("version_num", resolvedPV.VersionNum))

		// 3. Build element (name, namespace) lookup from the request.
		// The handler guarantees Namespace is always non-empty (defaults to "default").
		type elemKey struct{ name, namespace string }
		requestedElements := make(map[elemKey]model.ElementApprovalInput, len(pi.Elements))
		for _, e := range pi.Elements {
			requestedElements[elemKey{e.Name, e.Namespace}] = e
		}

		// 4. Validate that every requested (name, namespace) pair belongs to this purpose version.
		validElements := make(map[elemKey]bool, len(resolvedPV.Elements))
		for _, elem := range resolvedPV.Elements {
			validElements[elemKey{elem.Name, elem.Namespace}] = true
		}
		for k := range requestedElements {
			if !validElements[k] {
				return nil, fmt.Errorf("element '%s' in namespace '%s' does not belong to purpose '%s'",
					k.name, k.namespace, purposeName)
			}
		}

		// 5. Build approval rows for every element in the resolved version
		approvals := make([]model.ConsentElementApproval, 0, len(resolvedPV.Elements))
		for _, elem := range resolvedPV.Elements {
			approved := false
			var value *string
			if reqElem, found := requestedElements[elemKey{elem.Name, elem.Namespace}]; found {
				approved = reqElem.Approved
				if reqElem.Value != nil {
					strVal := valueToString(reqElem.Value)
					if err := validateElementValue(ctx, elem.ElementType, elem.Schema, strVal); err != nil {
						return nil, fmt.Errorf("element '%s' in purpose '%s': %w", elem.Name, purposeName, err)
					}
					value = &strVal
				}
			}

			approvals = append(approvals, model.ConsentElementApproval{
				// ConsentID set by the caller after this function returns
				PurposeVersionID: resolvedPV.VersionID,
				ElementVersionID: elem.ElementVersionID,
				Approved:         approved,
				Value:            value,
				OrgID:            orgID,
			})
		}

		links = append(links, resolvedPurposeLink{
			purposeVersionID: resolvedPV.VersionID,
			approvals:        approvals,
		})
	}

	logger.Info("Purposes validated and resolved", log.Int("purpose_count", len(links)))
	return links, nil
}

// loadConsentOutput fetches all related data for a consent and assembles a ConsentOutput.
func (s *consentService) loadConsentOutput(ctx context.Context, consent *model.Consent, orgID string) (*model.ConsentOutput, error) {
	consentStore := s.stores.Consent
	authResourceStore := s.stores.AuthResource

	attrs, err := consentStore.GetAttributesByConsentID(ctx, consent.ConsentID, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to load attributes: %w", err)
	}
	authResources, err := authResourceStore.GetByConsentID(ctx, consent.ConsentID, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to load auth resources: %w", err)
	}

	purposeRows, err := consentStore.GetPurposesByConsentID(ctx, consent.ConsentID, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to load purpose rows: %w", err)
	}
	approvalRows, err := consentStore.GetElementApprovalsByConsentID(ctx, consent.ConsentID, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to load approval rows: %w", err)
	}
	elementProps, err := consentStore.GetElementPropertiesByConsentID(ctx, consent.ConsentID, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to load element properties: %w", err)
	}
	purposeProps, err := consentStore.GetPurposePropertiesByConsentID(ctx, consent.ConsentID, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to load purpose properties: %w", err)
	}

	attrMap := make(map[string]string, len(attrs))
	for _, a := range attrs {
		attrMap[a.AttKey] = a.AttValue
	}

	return buildConsentOutput(consent, purposeRows, approvalRows, attrMap, authResources, elementProps, purposeProps), nil
}

// buildConsentOutput is a pure, side-effect-free function that assembles a ConsentOutput
// from pre-fetched data. No DB calls are made here.
// elementProps maps elementVersionID → {attKey → attValue}; may be nil (no properties loaded).
// purposeProps maps purposeVersionID → {attKey → attValue}; may be nil (no properties loaded).
func buildConsentOutput(
	consent *model.Consent,
	purposeRows []model.ConsentPurposeRow,
	approvalRows []model.ConsentApprovalRow,
	attrMap map[string]string,
	authResources []authmodel.AuthResource,
	elementProps map[string]map[string]string,
	purposeProps map[string]map[string]string,
) *model.ConsentOutput {
	// Index approvals by purposeVersionID → elementVersionID for O(1) lookup
	approvalIdx := make(map[string]map[string]model.ConsentApprovalRow, len(purposeRows))
	for _, row := range approvalRows {
		if approvalIdx[row.PurposeVersionID] == nil {
			approvalIdx[row.PurposeVersionID] = make(map[string]model.ConsentApprovalRow)
		}
		approvalIdx[row.PurposeVersionID][row.ElementVersionID] = row
	}

	purposes := make([]model.ConsentPurposeOutput, 0, len(purposeRows))
	for _, pr := range purposeRows {
		elemApprovals := approvalIdx[pr.PurposeVersionID]
		elements := make([]model.ConsentElementApprovalOutput, 0, len(elemApprovals))
		for _, ar := range elemApprovals {
			elements = append(elements, model.ConsentElementApprovalOutput{
				ElementVersionID: ar.ElementVersionID,
				ElementID:        ar.ElementID,
				Name:             ar.ElementName,
				Namespace:        ar.ElementNamespace,
				VersionNum:       ar.ElementVersionNum,
				ElementType:      ar.ElementType,
				Mandatory:        ar.Mandatory,
				Approved:         ar.Approved,
				Value:            ar.Value,
				DisplayName:      ar.ElementDisplayName,
				Description:      ar.ElementDescription,
				Properties:       elementProps[ar.ElementVersionID],
			})
		}
		purposes = append(purposes, model.ConsentPurposeOutput{
			PurposeVersionID: pr.PurposeVersionID,
			PurposeID:        pr.PurposeID,
			Name:             pr.PurposeName,
			GroupID:          pr.PurposeGroupID,
			VersionNum:       pr.PurposeVersion,
			DisplayName:      pr.DisplayName,
			Description:      pr.Description,
			Properties:       purposeProps[pr.PurposeVersionID],
			Elements:         elements,
		})
	}

	authOutputs := make([]authmodel.AuthResourceOutput, 0, len(authResources))
	for _, ar := range authResources {
		authOutputs = append(authOutputs, authResourceToOutput(ar))
	}

	return &model.ConsentOutput{
		ConsentID:                  consent.ConsentID,
		GroupID:                    consent.GroupID,
		ConsentType:                consent.ConsentType,
		CurrentStatus:              consent.CurrentStatus,
		ConsentFrequency:           consent.ConsentFrequency,
		ExpirationTime:             consent.ExpirationTime,
		RecurringIndicator:         consent.RecurringIndicator,
		DataAccessValidityDuration: consent.DataAccessValidityDuration,
		CreatedTime:                consent.CreatedTime,
		UpdatedTime:                consent.UpdatedTime,
		OrgID:                      consent.OrgID,
		Attributes:                 attrMap,
		Purposes:                   purposes,
		Authorizations:             authOutputs,
	}
}

// buildAuthResource builds an AuthResource DB model from a CreateAuthResourceInput,
// applying DefaultAuthType and the configured default auth status when the caller omits them.
func buildAuthResource(
	input authmodel.CreateAuthResourceInput,
	consentID, orgID string,
	updatedTime int64,
	defaultAuthStatus string,
) *authmodel.AuthResource {
	authType := input.AuthType
	if authType == "" {
		authType = authmodel.DefaultAuthType
	}
	status := input.AuthStatus
	if status == "" {
		status = defaultAuthStatus
	}

	var resourcesJSON *string
	if input.Resources != nil {
		b, err := json.Marshal(input.Resources)
		if err == nil {
			s := string(b)
			resourcesJSON = &s
		}
	}

	return &authmodel.AuthResource{
		AuthID:      utils.GenerateUUID(),
		ConsentID:   consentID,
		AuthType:    authType,
		UserID:      input.UserID,
		AuthStatus:  status,
		UpdatedTime: updatedTime,
		Resources:   resourcesJSON,
		OrgID:       orgID,
	}
}

// authResourceToOutput converts an AuthResource DB model to the service output type,
// parsing the JSON resources blob into an interface{}.
func authResourceToOutput(ar authmodel.AuthResource) authmodel.AuthResourceOutput {
	var resources interface{}
	if ar.Resources != nil && *ar.Resources != "" {
		_ = json.Unmarshal([]byte(*ar.Resources), &resources)
	}
	return authmodel.AuthResourceOutput{
		AuthID:      ar.AuthID,
		ConsentID:   ar.ConsentID,
		AuthType:    ar.AuthType,
		UserID:      ar.UserID,
		AuthStatus:  ar.AuthStatus,
		UpdatedTime: ar.UpdatedTime,
		Resources:   resources,
		OrgID:       ar.OrgID,
	}
}

// parseVersionString parses a version string in the "v<N>" format (e.g. "v1", "v2")
// and returns the integer version number.
func parseVersionString(s string) (int, error) {
	if len(s) < 2 || s[0] != 'v' {
		return 0, fmt.Errorf("invalid version format '%s': expected 'v<N>'", s)
	}
	n, err := strconv.Atoi(s[1:])
	if err != nil || n < 1 {
		return 0, fmt.Errorf("invalid version '%s': version number must be a positive integer", s)
	}
	return n, nil
}

// formatVersion converts an integer version number to the "v<N>" string format.
func formatVersion(n int) string {
	return "v" + strconv.Itoa(n)
}

// valueToString converts an arbitrary value to its string representation for storage.
// Plain strings are stored as-is; all other types are JSON-marshalled.
func valueToString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

// validateElementValue validates a consent element value against its element type and schema.
// basic elements are always skipped. json and xml elements are validated for syntax, and
// additionally validated against their schema when one is defined.
func validateElementValue(ctx context.Context, elemType string, schema *string, value string) error {
	switch elemType {
	case "json":
		return validateJSONElementValue(schema, value)
	case "xml":
		return validateXMLElementValue(ctx, schema, value)
	default:
		return nil
	}
}

// validateJSONElementValue checks that value is valid JSON and, when a schema is present,
// validates it against the JSON Schema.
func validateJSONElementValue(schema *string, value string) error {
	var instance interface{}
	if err := json.Unmarshal([]byte(value), &instance); err != nil {
		return fmt.Errorf("value is not valid JSON: %w", err)
	}
	if schema == nil || *schema == "" {
		return nil
	}
	var s jsonschema.Schema
	if err := json.Unmarshal([]byte(*schema), &s); err != nil {
		return fmt.Errorf("element has an invalid JSON schema definition: %w", err)
	}
	resolved, err := s.Resolve(nil)
	if err != nil {
		return fmt.Errorf("failed to resolve JSON schema: %w", err)
	}
	if err := resolved.Validate(instance); err != nil {
		return fmt.Errorf("value does not match element JSON schema: %w", err)
	}
	return nil
}

// validateXMLElementValue checks that value is well-formed XML and, when a schema is present,
// validates it against the XSD schema using helium.
func validateXMLElementValue(ctx context.Context, schema *string, value string) error {
	if schema == nil || *schema == "" {
		// No schema: well-formedness check only via standard library
		decoder := xml.NewDecoder(strings.NewReader(value))
		for {
			if _, err := decoder.Token(); err != nil {
				if err == io.EOF {
					break
				}
				return fmt.Errorf("value is not well-formed XML: %w", err)
			}
		}
		return nil
	}
	// Full XSD validation via helium
	p := helium.NewParser()
	schemaDoc, err := p.Parse(ctx, []byte(*schema))
	if err != nil {
		return fmt.Errorf("element has an invalid XSD schema definition: %w", err)
	}
	compiled, err := xsd.NewCompiler().Compile(ctx, schemaDoc)
	if err != nil {
		return fmt.Errorf("failed to compile XSD schema: %w", err)
	}
	doc, err := p.Parse(ctx, []byte(value))
	if err != nil {
		return fmt.Errorf("value is not valid XML: %w", err)
	}
	if err := xsd.NewValidator(compiled).Validate(ctx, doc); err != nil {
		return fmt.Errorf("value does not match element XSD schema: %w", err)
	}
	return nil
}
