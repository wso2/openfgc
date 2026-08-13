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

package authresource

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/wso2/openfgc/internal/authresource/model"
	authvalidator "github.com/wso2/openfgc/internal/authresource/validator"
	consenthistory "github.com/wso2/openfgc/internal/consent"
	consentModel "github.com/wso2/openfgc/internal/consent/model"
	"github.com/wso2/openfgc/internal/consent/validator"
	"github.com/wso2/openfgc/internal/system/config"
	dbmodel "github.com/wso2/openfgc/internal/system/database/model"
	"github.com/wso2/openfgc/internal/system/error/serviceerror"
	"github.com/wso2/openfgc/internal/system/log"
	"github.com/wso2/openfgc/internal/system/stores"
	"github.com/wso2/openfgc/internal/system/utils"
)

// AuthResourceServiceInterface defines the contract for auth resource business operations.
type AuthResourceServiceInterface interface {
	CreateAuthResource(ctx context.Context, consentID, orgID string, input model.CreateAuthResourceInput) (*model.AuthResourceOutput, *serviceerror.ServiceError)
	GetAuthResource(ctx context.Context, authID, consentID, orgID string) (*model.AuthResourceOutput, *serviceerror.ServiceError)
	GetAuthResourcesByConsentID(ctx context.Context, consentID, orgID string) (*model.AuthResourceListOutput, *serviceerror.ServiceError)
	UpdateAuthResource(ctx context.Context, authID, consentID, orgID string, input model.UpdateAuthResourceInput) (*model.AuthResourceOutput, *serviceerror.ServiceError)
	UpdateAllStatusByConsentID(ctx context.Context, consentID, orgID string, status string) *serviceerror.ServiceError
}

// authResourceService implements AuthResourceServiceInterface.
type authResourceService struct {
	stores *stores.StoreRegistry
}

// newAuthResourceService creates a new auth resource service.
func newAuthResourceService(registry *stores.StoreRegistry) AuthResourceServiceInterface {
	return &authResourceService{stores: registry}
}

// =============================================================================
// CreateAuthResource
// =============================================================================

// CreateAuthResource creates a new authorization resource for a consent.
// AuthType defaults to "primary" and AuthStatus defaults to the configured approved state
// when not provided by the caller.
func (s *authResourceService) CreateAuthResource(
	ctx context.Context,
	consentID, orgID string,
	input model.CreateAuthResourceInput,
) (*model.AuthResourceOutput, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)

	cfg := config.Get()
	if cfg == nil {
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, "configuration not initialized")
	}

	// Apply defaults for optional fields
	if input.AuthType == "" {
		input.AuthType = model.AuthTypePrimary
	}
	// Canonicalise reserved types so the persisted value matches what validation and search expect.
	input.AuthType = model.NormalizeAuthType(input.AuthType)
	if input.AuthStatus == "" {
		input.AuthStatus = string(cfg.Consent.GetApprovedAuthStatus())
	}

	logger.Info("Creating auth resource",
		log.String("consent_id", consentID),
		log.String("org_id", orgID),
		log.String("auth_type", input.AuthType),
		log.String("auth_status", input.AuthStatus))

	if err := s.validateConsentIDAndOrgID(consentID, orgID); err != nil {
		return nil, err
	}

	// Validate that auth status is not a system-reserved status
	if err := authvalidator.ValidateAuthStatus(input.AuthStatus, cfg.Consent.AuthStatusMappings); err != nil {
		return nil, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error())
	}

	authID := utils.GenerateUUID()
	updatedTime := utils.GetCurrentTimeMillis()

	var resourcesJSON *string
	if input.Resources != nil {
		b, err := json.Marshal(input.Resources)
		if err != nil {
			return nil, serviceerror.CustomServiceError(ErrorValidationFailed,
				fmt.Sprintf("failed to marshal resources: %v", err))
		}
		s := string(b)
		resourcesJSON = &s
	}

	authResource := &model.AuthResource{
		AuthID:      authID,
		ConsentID:   consentID,
		AuthType:    input.AuthType,
		UserID:      input.UserID,
		AuthStatus:  input.AuthStatus,
		UpdatedTime: updatedTime,
		Resources:   resourcesJSON,
		OrgID:       orgID,
	}

	store := s.stores.AuthResource

	// The consent state and its existing authorizations are read under a row lock inside the
	// transaction, so validation and status derivation act on a snapshot that a concurrent
	// authorization write or revocation cannot invalidate between the read and the write.
	var svcErr *serviceerror.ServiceError
	var derivedConsentStatus string
	var previousConsentStatus string

	txErr := s.stores.ExecuteTransaction([]func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			currentConsent, err := s.stores.Consent.GetByIDForUpdate(tx, consentID, orgID)
			if err != nil {
				return err
			}
			if currentConsent == nil {
				svcErr = serviceerror.CustomServiceError(ErrorConsentNotFound,
					fmt.Sprintf("consent %s does not exist in org %s", consentID, orgID))
				return errAuthWriteAborted
			}
			if e := ensureConsentAcceptsAuthorizationWrites(currentConsent); e != nil {
				svcErr = e
				return errAuthWriteAborted
			}

			allAuthResources, err := store.GetByConsentIDTx(tx, consentID, orgID)
			if err != nil {
				return err
			}

			resultingAuthResources := make([]model.AuthResource, 0, len(allAuthResources)+1)
			resultingAuthResources = append(resultingAuthResources, *authResource)
			resultingAuthResources = append(resultingAuthResources, allAuthResources...)

			// The rules describe a consent as a whole, so the resulting set is validated, not the input.
			if err := validator.ValidateAuthResourceTypeConstraints(resultingAuthResources); err != nil {
				svcErr = serviceerror.CustomServiceError(ErrorValidationFailed, err.Error())
				return errAuthWriteAborted
			}

			authStatuses := make([]string, 0, len(resultingAuthResources))
			for _, ar := range resultingAuthResources {
				authStatuses = append(authStatuses, ar.AuthStatus)
			}
			derivedConsentStatus = validator.EvaluateConsentStatusFromAuthStatuses(authStatuses)
			previousConsentStatus = currentConsent.CurrentStatus
			return nil
		},
		func(tx dbmodel.TxInterface) error {
			return consenthistory.RecordConsentHistory(ctx, s.stores, tx, consentID, orgID, nil, consenthistory.HistoryReasonConsentAuthorizationsAdded)
		},
		func(tx dbmodel.TxInterface) error {
			return store.Create(tx, authResource)
		},
		func(tx dbmodel.TxInterface) error {
			if previousConsentStatus == derivedConsentStatus {
				return nil
			}
			t := utils.GetCurrentTimeMillis()
			if err := s.stores.Consent.UpdateStatus(tx, consentID, orgID, derivedConsentStatus, t); err != nil {
				return err
			}
			reason := fmt.Sprintf("auth resource %s created with status %s", authID, input.AuthStatus)
			audit := &consentModel.ConsentStatusAudit{
				StatusAuditID:  utils.GenerateUUID(),
				ConsentID:      consentID,
				CurrentStatus:  derivedConsentStatus,
				ActionTime:     t,
				Reason:         &reason,
				PreviousStatus: &previousConsentStatus,
				OrgID:          orgID,
			}
			return s.stores.Consent.CreateStatusAudit(tx, audit)
		},
	})
	if txErr != nil {
		if svcErr != nil {
			return nil, svcErr
		}
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError,
			fmt.Sprintf("failed to create auth resource: %v", txErr))
	}

	logger.Info("Auth resource created", log.String("auth_id", authID))
	return buildAuthResourceOutput(authResource), nil
}

// =============================================================================
// GetAuthResource
// =============================================================================

// GetAuthResource retrieves an authorization resource by ID.
func (s *authResourceService) GetAuthResource(
	ctx context.Context,
	authID, consentID, orgID string,
) (*model.AuthResourceOutput, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Debug("Retrieving auth resource",
		log.String("auth_id", authID),
		log.String("consent_id", consentID))

	if err := s.validateAuthIDAndOrgID(authID, orgID); err != nil {
		return nil, err
	}
	if err := s.validateConsentIDAndOrgID(consentID, orgID); err != nil {
		return nil, err
	}

	ar, err := s.stores.AuthResource.GetByID(ctx, authID, orgID)
	if err != nil {
		logger.Error("Failed to retrieve auth resource", log.Error(err), log.String("auth_id", authID))
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError,
			fmt.Sprintf("failed to retrieve auth resource: %v", err))
	}
	if ar == nil || ar.ConsentID != consentID {
		return nil, serviceerror.CustomServiceError(ErrorAuthResourceNotFound,
			"the authorization resource does not exist, does not belong to the specified consent, or is not accessible in this organization")
	}

	return buildAuthResourceOutput(ar), nil
}

// =============================================================================
// GetAuthResourcesByConsentID
// =============================================================================

// GetAuthResourcesByConsentID retrieves all authorization resources for a consent.
func (s *authResourceService) GetAuthResourcesByConsentID(
	ctx context.Context,
	consentID, orgID string,
) (*model.AuthResourceListOutput, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Debug("Retrieving auth resources by consent",
		log.String("consent_id", consentID))

	if err := s.validateConsentIDAndOrgID(consentID, orgID); err != nil {
		return nil, err
	}

	authResources, err := s.stores.AuthResource.GetByConsentID(ctx, consentID, orgID)
	if err != nil {
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError,
			fmt.Sprintf("failed to fetch auth resources: %v", err))
	}

	data := make([]model.AuthResourceOutput, 0, len(authResources))
	for _, ar := range authResources {
		data = append(data, *buildAuthResourceOutput(&ar))
	}

	return &model.AuthResourceListOutput{Data: data}, nil
}

// =============================================================================
// UpdateAuthResource
// =============================================================================

// UpdateAuthResource updates an existing authorization resource.
func (s *authResourceService) UpdateAuthResource(
	ctx context.Context,
	authID, consentID, orgID string,
	input model.UpdateAuthResourceInput,
) (*model.AuthResourceOutput, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Info("Updating auth resource",
		log.String("auth_id", authID),
		log.String("consent_id", consentID))

	if err := s.validateAuthIDAndOrgID(authID, orgID); err != nil {
		return nil, err
	}
	if err := s.validateConsentIDAndOrgID(consentID, orgID); err != nil {
		return nil, err
	}

	store := s.stores.AuthResource
	existing, err := store.GetByID(ctx, authID, orgID)
	if err != nil {
		logger.Error("Failed to retrieve auth resource", log.Error(err), log.String("auth_id", authID))
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError,
			fmt.Sprintf("failed to retrieve auth resource: %v", err))
	}
	if existing == nil || existing.ConsentID != consentID {
		return nil, serviceerror.CustomServiceError(ErrorAuthResourceNotFound,
			"the authorization resource does not exist, does not belong to the specified consent, or is not accessible in this organization")
	}
	updated := *existing
	updated.UpdatedTime = utils.GetCurrentTimeMillis()

	statusChanged := false
	if input.AuthStatus != "" {
		cfg := config.Get()
		if cfg == nil {
			return nil, serviceerror.CustomServiceError(ErrorInternalServerError, "configuration not initialized")
		}
		if err := authvalidator.ValidateAuthStatus(input.AuthStatus, cfg.Consent.AuthStatusMappings); err != nil {
			return nil, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error())
		}
		statusChanged = existing.AuthStatus != input.AuthStatus
		updated.AuthStatus = input.AuthStatus
	}
	if input.AuthType != "" {
		// Canonicalise reserved types so the persisted value matches what validation and search expect.
		updated.AuthType = model.NormalizeAuthType(input.AuthType)
	}
	if input.UserID != nil {
		updated.UserID = input.UserID
	}
	if input.Resources != nil {
		b, err := json.Marshal(input.Resources)
		if err != nil {
			return nil, serviceerror.CustomServiceError(ErrorValidationFailed,
				fmt.Sprintf("failed to marshal resources: %v", err))
		}
		rs := string(b)
		updated.Resources = &rs
	}

	typeChanged := input.AuthType != "" && existing.AuthType != updated.AuthType

	// The consent state and its existing authorizations are read under a row lock inside the
	// transaction, so validation and status derivation act on a snapshot that a concurrent
	// authorization write or revocation cannot invalidate between the read and the write. The
	// type is needed to validate the consent's resulting authorizations; the status is
	// additionally needed to derive the consent status. Updates to other fields need neither.
	var svcErr *serviceerror.ServiceError
	var derivedConsentStatus string
	var previousConsentStatus string
	var statusNeedsUpdate bool

	txSteps := []func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			currentConsent, err := s.stores.Consent.GetByIDForUpdate(tx, consentID, orgID)
			if err != nil {
				return err
			}
			if currentConsent == nil {
				svcErr = serviceerror.CustomServiceError(ErrorConsentNotFound,
					fmt.Sprintf("consent %s does not exist in org %s", consentID, orgID))
				return errAuthWriteAborted
			}
			if e := ensureConsentAcceptsAuthorizationWrites(currentConsent); e != nil {
				svcErr = e
				return errAuthWriteAborted
			}
			previousConsentStatus = currentConsent.CurrentStatus

			if statusChanged || typeChanged {
				allAuthResources, err := store.GetByConsentIDTx(tx, consentID, orgID)
				if err != nil {
					return err
				}

				resultingAuthResources := make([]model.AuthResource, 0, len(allAuthResources))
				for _, ar := range allAuthResources {
					if ar.AuthID == authID {
						resultingAuthResources = append(resultingAuthResources, updated)
						continue
					}
					resultingAuthResources = append(resultingAuthResources, ar)
				}

				// The rules describe a consent as a whole, so the resulting set is validated, not the input.
				if err := validator.ValidateAuthResourceTypeConstraints(resultingAuthResources); err != nil {
					svcErr = serviceerror.CustomServiceError(ErrorValidationFailed, err.Error())
					return errAuthWriteAborted
				}

				if statusChanged {
					authStatuses := make([]string, 0, len(resultingAuthResources))
					for _, ar := range resultingAuthResources {
						authStatuses = append(authStatuses, ar.AuthStatus)
					}
					derivedConsentStatus = validator.EvaluateConsentStatusFromAuthStatuses(authStatuses)
					statusNeedsUpdate = true
				}
			}
			return nil
		},
		func(tx dbmodel.TxInterface) error {
			return consenthistory.RecordConsentHistory(ctx, s.stores, tx, consentID, orgID, nil, consenthistory.HistoryReasonConsentAuthorizationsUpdated)
		},
		func(tx dbmodel.TxInterface) error { return store.Update(tx, &updated) },
		func(tx dbmodel.TxInterface) error {
			if !statusNeedsUpdate || previousConsentStatus == derivedConsentStatus {
				return nil
			}
			t := utils.GetCurrentTimeMillis()
			if err := s.stores.Consent.UpdateStatus(tx, consentID, orgID, derivedConsentStatus, t); err != nil {
				return err
			}
			reason := fmt.Sprintf("auth resource %s status changed from %s to %s",
				authID, existing.AuthStatus, updated.AuthStatus)
			audit := &consentModel.ConsentStatusAudit{
				StatusAuditID:  utils.GenerateUUID(),
				ConsentID:      consentID,
				CurrentStatus:  derivedConsentStatus,
				ActionTime:     t,
				Reason:         &reason,
				PreviousStatus: &previousConsentStatus,
				OrgID:          orgID,
			}
			return s.stores.Consent.CreateStatusAudit(tx, audit)
		},
	}

	if txErr := s.stores.ExecuteTransaction(txSteps); txErr != nil {
		if svcErr != nil {
			return nil, svcErr
		}
		return nil, serviceerror.CustomServiceError(ErrorUpdateAuthResource,
			fmt.Sprintf("failed to update auth resource: %v", txErr))
	}

	logger.Info("Auth resource updated", log.String("auth_id", authID))
	return buildAuthResourceOutput(&updated), nil
}

// =============================================================================
// UpdateAllStatusByConsentID
// =============================================================================

// UpdateAllStatusByConsentID sets the status of all auth resources for a consent.
func (s *authResourceService) UpdateAllStatusByConsentID(
	ctx context.Context,
	consentID, orgID string,
	status string,
) *serviceerror.ServiceError {
	logger := log.GetLogger().WithContext(ctx)

	if err := s.validateConsentIDAndOrgID(consentID, orgID); err != nil {
		return err
	}
	if status == "" {
		return serviceerror.CustomServiceError(ErrorInvalidRequestBody, "status is required")
	}

	t := utils.GetCurrentTimeMillis()
	err := s.stores.ExecuteTransaction([]func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return s.stores.AuthResource.UpdateAllStatusByConsentID(tx, consentID, orgID, status, t)
		},
	})
	if err != nil {
		return serviceerror.CustomServiceError(ErrorUpdateAuthResource,
			fmt.Sprintf("failed to update auth resource statuses: %v", err))
	}

	logger.Info("All auth resource statuses updated",
		log.String("consent_id", consentID),
		log.String("status", status))
	return nil
}

// =============================================================================
// Private helpers
// =============================================================================

// buildAuthResourceOutput converts an AuthResource DB model to AuthResourceOutput.
// The Resources JSON blob is unmarshalled into an interface{}.
func buildAuthResourceOutput(ar *model.AuthResource) *model.AuthResourceOutput {
	var resources interface{}
	if ar.Resources != nil && *ar.Resources != "" {
		if err := json.Unmarshal([]byte(*ar.Resources), &resources); err != nil {
			log.GetLogger().Error("Failed to unmarshal auth resource resources",
				log.String("auth_id", ar.AuthID),
				log.Error(err))
		}
	}
	return &model.AuthResourceOutput{
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

func (s *authResourceService) validateAuthIDAndOrgID(authID, orgID string) *serviceerror.ServiceError {
	if authID == "" {
		return serviceerror.CustomServiceError(ErrorAuthResourceIDRequired, "auth ID is required")
	}
	if len(authID) > 255 {
		return serviceerror.CustomServiceError(ErrorValidationFailed, "auth ID too long (max 255 characters)")
	}
	return s.validateOrgID(orgID)
}

// errAuthWriteAborted rolls back an authorization-write transaction when a precondition or
// validation check inside it fails. The failing step records the specific *serviceerror to
// return to the caller and returns this sentinel so ExecuteTransaction rolls the write back.
var errAuthWriteAborted = errors.New("authorization write aborted")

// ensureConsentAcceptsAuthorizationWrites rejects authorization writes to a consent that has
// reached a terminal state — revoked or expired. Both states mark every existing authorization
// with a status that is excluded from consent status derivation (system-revoked or
// system-expired). A later write would therefore derive a status from the incoming
// authorization alone and could return the consent to an active state, contradicting the
// withdrawal a revocation recorded or the lapse an expiration recorded. An expired consent must
// be renewed through UpdateConsent before its authorizations can change.
func ensureConsentAcceptsAuthorizationWrites(consent *consentModel.Consent) *serviceerror.ServiceError {
	cfg := config.Get()
	if cfg == nil {
		return serviceerror.CustomServiceError(ErrorInternalServerError, "configuration not initialized")
	}

	if strings.EqualFold(consent.CurrentStatus, string(cfg.Consent.GetRevokedConsentStatus())) {
		return serviceerror.CustomServiceError(ErrorValidationFailed,
			fmt.Sprintf("consent %s is revoked; its authorizations can no longer be added to or modified",
				consent.ConsentID))
	}

	if strings.EqualFold(consent.CurrentStatus, string(cfg.Consent.GetExpiredConsentStatus())) {
		return serviceerror.CustomServiceError(ErrorValidationFailed,
			fmt.Sprintf("consent %s is expired; renew it before adding to or modifying its authorizations",
				consent.ConsentID))
	}

	return nil
}

func (s *authResourceService) validateConsentIDAndOrgID(consentID, orgID string) *serviceerror.ServiceError {
	if consentID == "" {
		return serviceerror.CustomServiceError(ErrorConsentIDRequired, "consent ID is required")
	}
	if len(consentID) > 255 {
		return serviceerror.CustomServiceError(ErrorValidationFailed, "consent ID too long (max 255 characters)")
	}
	return s.validateOrgID(orgID)
}

func (s *authResourceService) validateOrgID(orgID string) *serviceerror.ServiceError {
	if orgID == "" {
		return serviceerror.CustomServiceError(ErrorOrgIDRequired, "organization ID is required")
	}
	if len(orgID) > 255 {
		return serviceerror.CustomServiceError(ErrorValidationFailed, "organization ID too long (max 255 characters)")
	}
	return nil
}
