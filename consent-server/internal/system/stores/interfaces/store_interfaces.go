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

// Package interfaces defines the store interfaces for data operations.
package interfaces

import (
	"context"

	authResourceModel "github.com/wso2/consent-management-api/internal/authresource/model"
	consentModel "github.com/wso2/consent-management-api/internal/consent/model"
	consentElementModel "github.com/wso2/consent-management-api/internal/consentelement/model"
	consentConsentPurposeModel "github.com/wso2/consent-management-api/internal/consentpurpose/model"
	dbmodel "github.com/wso2/consent-management-api/internal/system/database/model"
)

// ConsentStore defines the interface for consent data operations
type ConsentStore interface {
	GetByID(ctx context.Context, consentID, orgID string) (*consentModel.Consent, error)
	List(ctx context.Context, orgID string, limit, offset int) ([]consentModel.Consent, int, error)
	Search(ctx context.Context, filters consentModel.ConsentSearchFilters) ([]consentModel.Consent, int, error)
	GetByClientID(ctx context.Context, clientID, orgID string) ([]consentModel.Consent, error)
	GetAttributesByConsentID(ctx context.Context, consentID, orgID string) ([]consentModel.ConsentAttribute, error)
	GetAttributesByConsentIDs(ctx context.Context, consentIDs []string, orgID string) (map[string]map[string]string, error)
	GetStatusAuditByConsentID(ctx context.Context, consentID, orgID string) ([]consentModel.ConsentStatusAudit, error)
	FindConsentIDsByAttributeKey(ctx context.Context, key, orgID string) ([]string, error)
	FindConsentIDsByAttribute(ctx context.Context, key, value, orgID string) ([]string, error)
	Create(tx dbmodel.TxInterface, consent *consentModel.Consent) error
	Update(tx dbmodel.TxInterface, consent *consentModel.Consent) error
	UpdateStatus(tx dbmodel.TxInterface, consentID, orgID, status string, updatedTime int64) error
	Delete(tx dbmodel.TxInterface, consentID, orgID string) error
	CreateAttributes(tx dbmodel.TxInterface, attributes []consentModel.ConsentAttribute) error
	DeleteAttributesByConsentID(tx dbmodel.TxInterface, consentID, orgID string) error
	CreateStatusAudit(tx dbmodel.TxInterface, audit *consentModel.ConsentStatusAudit) error
	// Purpose Consent mapping methods
	CreateConsentPurposeConsent(tx dbmodel.TxInterface, consentID, purposeID, orgID string) error
	CreatePurposeApproval(tx dbmodel.TxInterface, approval *consentModel.ConsentPurposeApprovalRecord) error
	GetConsentPurposesByConsentID(ctx context.Context, consentID, orgID string) ([]consentModel.ConsentPurposeMapping, error)
	GetPurposeApprovalsByConsentID(ctx context.Context, consentID, orgID string) ([]consentModel.ConsentPurposeApprovalRecord, error)
	DeleteConsentPurposesByConsentID(tx dbmodel.TxInterface, consentID, orgID string) error
	DeletePurposeApprovalsByConsentID(tx dbmodel.TxInterface, consentID, orgID string) error
	CheckPurposeUsedInConsents(ctx context.Context, purposeID, orgID string) (bool, error)
}

// AuthResourceStore defines the interface for authorization resource data operations
type AuthResourceStore interface {
	GetByID(ctx context.Context, authID, orgID string) (*authResourceModel.AuthResource, error)
	GetByConsentID(ctx context.Context, consentID, orgID string) ([]authResourceModel.AuthResource, error)
	GetByConsentIDs(ctx context.Context, consentIDs []string, orgID string) ([]authResourceModel.AuthResource, error)
	Exists(ctx context.Context, authID, orgID string) (bool, error)
	GetByUserID(ctx context.Context, userID, orgID string) ([]authResourceModel.AuthResource, error)
	Create(tx dbmodel.TxInterface, authResource *authResourceModel.AuthResource) error
	Update(tx dbmodel.TxInterface, authResource *authResourceModel.AuthResource) error
	UpdateStatus(tx dbmodel.TxInterface, authID, orgID, status string, updatedTime int64) error
	Delete(tx dbmodel.TxInterface, authID, orgID string) error
	DeleteByConsentID(tx dbmodel.TxInterface, consentID, orgID string) error
	UpdateAllStatusByConsentID(tx dbmodel.TxInterface, consentID, orgID, status string, updatedTime int64) error
}

// ConsentElementStore defines the interface for consent purpose data operations
type ConsentElementStore interface {
	GetByID(ctx context.Context, purposeID, orgID string) (*consentElementModel.ConsentElement, error)
	GetByName(ctx context.Context, name, orgID string) (*consentElementModel.ConsentElement, error)
	List(ctx context.Context, orgID string, limit, offset int, name string) ([]consentElementModel.ConsentElement, int, error)
	CheckNameExists(ctx context.Context, name, orgID string) (bool, error)
	GetPropertiesByElementID(ctx context.Context, elementID, orgID string) ([]consentElementModel.ConsentElementProperty, error)
	GetMappingsByConsentID(ctx context.Context, consentID, orgID string) ([]consentElementModel.ConsentElementMapping, error)
	GetMappingsByConsentIDs(ctx context.Context, consentIDs []string, orgID string) ([]consentElementModel.ConsentElementMapping, error)
	GetIDsByNames(ctx context.Context, names []string, orgID string) (map[string]string, error)
	Create(tx dbmodel.TxInterface, element *consentElementModel.ConsentElement) error
	Update(tx dbmodel.TxInterface, element *consentElementModel.ConsentElement) error
	Delete(tx dbmodel.TxInterface, elementID, orgID string) error
	CreateProperties(tx dbmodel.TxInterface, properties []consentElementModel.ConsentElementProperty) error
	DeletePropertiesByElementID(tx dbmodel.TxInterface, elementID, orgID string) error
	LinkElementToConsent(tx dbmodel.TxInterface, consentID, elementID, orgID string, value *string, isUserApproved, isMandatory bool) error
	DeleteMappingsByConsentID(tx dbmodel.TxInterface, consentID, orgID string) error
}

// ConsentPurposeStore defines the interface for purpose data operations
type ConsentPurposeStore interface {
	CreatePurpose(tx dbmodel.TxInterface, purpose *consentConsentPurposeModel.ConsentPurpose) error
	GetPurposeByID(ctx context.Context, purposeID, orgID string) (*consentConsentPurposeModel.ConsentPurpose, error)
	GetPurposeByName(ctx context.Context, name, orgID string) (*consentConsentPurposeModel.ConsentPurpose, error)
	ListPurposes(ctx context.Context, orgID, name string, clientIDs []string, purposeNames []string, offset, limit int) ([]consentConsentPurposeModel.ConsentPurpose, int, error)
	UpdatePurpose(tx dbmodel.TxInterface, purpose *consentConsentPurposeModel.ConsentPurpose) error
	DeletePurpose(tx dbmodel.TxInterface, purposeID, orgID string) error
	CheckPurposeNameExists(ctx context.Context, name, clientID, orgID string, excludePurposeID *string) (bool, error)
	LinkElementToPurpose(tx dbmodel.TxInterface, purposeID, elementID, orgID string, isMandatory bool) error
	GetPurposeElements(ctx context.Context, purposeID, orgID string) ([]consentConsentPurposeModel.PurposeElement, error)
	DeletePurposeElements(tx dbmodel.TxInterface, purposeID, orgID string) error
	GetPurposeIDByName(ctx context.Context, purposeName, orgID string) (string, error)
	ValidatePurposeNames(ctx context.Context, purposeNames []string, orgID string) (map[string]string, error)
	IsElementUsedInPurposes(ctx context.Context, elementID, orgID string) (bool, error)
}
