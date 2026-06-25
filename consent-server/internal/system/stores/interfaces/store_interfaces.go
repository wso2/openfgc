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

	authResourceModel "github.com/wso2/openfgc/internal/authresource/model"
	consentModel "github.com/wso2/openfgc/internal/consent/model"
	consentElementModel "github.com/wso2/openfgc/internal/consentelement/model"
	purposeModel "github.com/wso2/openfgc/internal/consentpurpose/model"
	dbmodel "github.com/wso2/openfgc/internal/system/database/model"
)

// ConsentStore defines the interface for consent data operations.
type ConsentStore interface {
	// Create inserts a new CONSENT row within a transaction.
	Create(tx dbmodel.TxInterface, consent *consentModel.Consent) error
	// Update overwrites the mutable fields of an existing consent within a transaction.
	Update(tx dbmodel.TxInterface, consent *consentModel.Consent) error
	// UpdateStatus changes CURRENT_STATUS and UPDATED_TIME for a consent within a transaction.
	// Returns an error if no row was matched (consent not found).
	UpdateStatus(tx dbmodel.TxInterface, consentID, orgID, status string, updatedTime int64) error
	// GetByID returns the CONSENT row for the given ID, or nil if not found.
	GetByID(ctx context.Context, consentID, orgID string) (*consentModel.Consent, error)
	// GetByIDForUpdate returns the CONSENT row and locks it for update within a transaction.
	GetByIDForUpdate(tx dbmodel.TxInterface, consentID, orgID string) (*consentModel.Consent, error)
	// Search returns consents matching the filters along with the total count for pagination.
	Search(ctx context.Context, filters consentModel.ConsentSearchFilter) ([]consentModel.Consent, int, error)
	// GetExpiredConsents Get expired consents based on current time and expirable statuses.
	GetExpiredConsents(ctx context.Context, currentTimeMs int64, expirableStatuses []string) ([]consentModel.Consent, error)

	// CreateAttributes inserts one or more CONSENT_ATTRIBUTE rows within a transaction.
	CreateAttributes(tx dbmodel.TxInterface, attributes []consentModel.ConsentAttribute) error
	// DeleteAttributesByConsentID removes all attributes for a consent within a transaction.
	DeleteAttributesByConsentID(tx dbmodel.TxInterface, consentID, orgID string) error
	// GetAttributesByConsentID returns all attributes for a single consent.
	GetAttributesByConsentID(ctx context.Context, consentID, orgID string) ([]consentModel.ConsentAttribute, error)
	// GetAttributesByConsentIDTx returns all attributes for a single consent within a transaction.
	GetAttributesByConsentIDTx(tx dbmodel.TxInterface, consentID, orgID string) ([]consentModel.ConsentAttribute, error)
	// GetAttributesByConsentIDs returns attributes for multiple consents in one query,
	// keyed by consent ID then attribute key.
	GetAttributesByConsentIDs(ctx context.Context, consentIDs []string, orgID string) (map[string]map[string]string, error)
	// GetConsentIDsByAttributeKey returns all consent IDs that carry the given attribute key.
	GetConsentIDsByAttributeKey(ctx context.Context, key, orgID string) ([]string, error)
	// GetConsentIDsByAttribute returns all consent IDs that carry the given key-value attribute pair.
	GetConsentIDsByAttribute(ctx context.Context, key, value, orgID string) ([]string, error)

	// CreateStatusAudit inserts a CONSENT_STATUS_AUDIT row within a transaction.
	CreateStatusAudit(tx dbmodel.TxInterface, audit *consentModel.ConsentStatusAudit) error
	// CreateHistory inserts a CONSENT_HISTORY row within a transaction.
	CreateHistory(tx dbmodel.TxInterface, history *consentModel.ConsentHistory) error
	// GetHistoryByConsentID returns consent history for a consent.
	GetHistoryByConsentID(ctx context.Context, consentID, orgID string, includeSnapshots bool) ([]consentModel.ConsentHistory, error)
	// GetStatusAuditsByConsentID returns status audit history for a consent.
	GetStatusAuditsByConsentID(ctx context.Context, consentID, orgID string) ([]consentModel.ConsentStatusAudit, error)

	// LinkPurposeVersionToConsent records that a consent was created against a specific purpose version.
	LinkPurposeVersionToConsent(tx dbmodel.TxInterface, consentID, purposeVersionID, orgID string) error
	// DeletePurposesByConsentID removes all purpose-version links for a consent within a transaction.
	DeletePurposesByConsentID(tx dbmodel.TxInterface, consentID, orgID string) error
	// GetPurposesByConsentID returns purpose rows joined with PURPOSE metadata for a consent.
	GetPurposesByConsentID(ctx context.Context, consentID, orgID string) ([]consentModel.ConsentPurposeRow, error)
	// GetPurposesByConsentIDTx returns purpose rows joined with PURPOSE metadata within a transaction.
	GetPurposesByConsentIDTx(tx dbmodel.TxInterface, consentID, orgID string) ([]consentModel.ConsentPurposeRow, error)
	// IsPurposeUsedInConsents reports whether any version of a logical purpose is referenced by any consent.
	// Returns true → caller must reject the purpose delete with 409 Conflict.
	IsPurposeUsedInConsents(ctx context.Context, purposeID, orgID string) (bool, error)

	// CreateElementApproval records a user's approval state for one element within a purpose version.
	CreateElementApproval(tx dbmodel.TxInterface, approval *consentModel.ConsentElementApproval) error
	// DeleteElementApprovalsByConsentID removes all element approvals for a consent within a transaction.
	DeleteElementApprovalsByConsentID(tx dbmodel.TxInterface, consentID, orgID string) error
	// GetElementApprovalsByConsentID returns approval rows joined with ELEMENT metadata for a consent.
	GetElementApprovalsByConsentID(ctx context.Context, consentID, orgID string) ([]consentModel.ConsentApprovalRow, error)
	// GetElementApprovalsByConsentIDTx returns approval rows joined with ELEMENT metadata within a transaction.
	GetElementApprovalsByConsentIDTx(tx dbmodel.TxInterface, consentID, orgID string) ([]consentModel.ConsentApprovalRow, error)
	// GetElementPropertiesByConsentID returns element properties for all elements in a consent,
	// keyed by element version ID then attribute key.
	GetElementPropertiesByConsentID(ctx context.Context, consentID, orgID string) (map[string]map[string]string, error)
	// GetElementPropertiesByConsentIDTx returns element properties within a transaction.
	GetElementPropertiesByConsentIDTx(tx dbmodel.TxInterface, consentID, orgID string) (map[string]map[string]string, error)
	// GetPurposePropertiesByConsentID returns purpose properties for all purposes in a consent,
	// keyed by purpose version ID then attribute key.
	GetPurposePropertiesByConsentID(ctx context.Context, consentID, orgID string) (map[string]map[string]string, error)
	// GetPurposePropertiesByConsentIDTx returns purpose properties within a transaction.
	GetPurposePropertiesByConsentIDTx(tx dbmodel.TxInterface, consentID, orgID string) (map[string]map[string]string, error)
}

// AuthResourceStore defines the interface for authorization resource data operations.
// Auth resources are stored in the CONSENT_AUTH_RESOURCE table and are always scoped to
// a consent (consentID) and an organization (orgID).
type AuthResourceStore interface {
	// Create inserts a new CONSENT_AUTH_RESOURCE row within a transaction.
	Create(tx dbmodel.TxInterface, authResource *authResourceModel.AuthResource) error

	// Update overwrites AUTH_STATUS, USER_ID, RESOURCES, and UPDATED_TIME for an existing
	// auth resource within a transaction. AUTH_ID and ORG_ID are used as the key.
	Update(tx dbmodel.TxInterface, authResource *authResourceModel.AuthResource) error

	// DeleteByConsentID removes all auth resource rows for a consent within a transaction.
	// Called when a consent is being replaced (e.g., full update that replaces authorizations).
	DeleteByConsentID(tx dbmodel.TxInterface, consentID, orgID string) error

	// UpdateAllStatusByConsentID sets AUTH_STATUS and UPDATED_TIME for every auth resource
	// belonging to a consent within a transaction.
	// Used during consent revocation and expiry to bulk-update statuses in one statement.
	UpdateAllStatusByConsentID(tx dbmodel.TxInterface, consentID, orgID, status string, updatedTime int64) error

	// GetByID returns the CONSENT_AUTH_RESOURCE row for the given AUTH_ID, or nil if not found.
	// The caller is responsible for verifying that the returned resource belongs to the
	// expected consentID — ownership is not checked inside the store.
	GetByID(ctx context.Context, authID, orgID string) (*authResourceModel.AuthResource, error)

	// GetByConsentID returns all auth resource rows for a consent.
	// Used when deriving the aggregate consent status from individual auth statuses.
	GetByConsentID(ctx context.Context, consentID, orgID string) ([]authResourceModel.AuthResource, error)
	// GetByConsentIDTx returns all auth resource rows for a consent within a transaction.
	GetByConsentIDTx(tx dbmodel.TxInterface, consentID, orgID string) ([]authResourceModel.AuthResource, error)

	// GetByConsentIDs returns auth resource rows for multiple consents in one query.
	// Used for batch-loading auth resources during consent list/search responses.
	// Returns an empty slice (not an error) when consentIDs is empty.
	GetByConsentIDs(ctx context.Context, consentIDs []string, orgID string) ([]authResourceModel.AuthResource, error)
}

// ConsentElementStore defines the interface for consent element data operations.
// Each logical element is identified by an ID and has one or more immutable versions.
// Version 1 is created when the element is first created; subsequent versions are added via CreateVersion.
type ConsentElementStore interface {
	// CreateVersion inserts a new element version (ELEMENT row + ELEMENT_PROPERTY rows).
	// Used for the initial create (version=1) and all subsequent versions.
	CreateVersion(tx dbmodel.TxInterface, version *consentElementModel.ElementVersion) error

	// GetLatestVersion returns the highest-numbered version of an element, with properties populated.
	GetLatestVersion(ctx context.Context, elementID, orgID string) (*consentElementModel.ElementVersion, error)

	// GetVersion returns a specific version by version number, with properties populated.
	GetVersion(ctx context.Context, elementID string, version int, orgID string) (*consentElementModel.ElementVersion, error)

	// ListVersions returns all versions of one element ordered by version number ascending, with properties.
	ListVersions(ctx context.Context, elementID, orgID string) ([]consentElementModel.ElementVersion, error)

	// List returns the latest version of each element matching the filters, with total count for pagination.
	// When filters.Details is false, Schema and Properties are not populated.
	List(ctx context.Context, orgID string, filters consentElementModel.ElementListFilter) ([]consentElementModel.ElementVersion, int, error)

	// GetByNameAndNamespace returns the latest version of an element matching name+namespace, or nil if not found.
	// Used for duplicate-name checks on element create.
	GetByNameAndNamespace(ctx context.Context, name, namespace, orgID string) (*consentElementModel.ElementVersion, error)

	// ElementExists reports whether any version of the element exists.
	ElementExists(ctx context.Context, elementID, orgID string) (bool, error)

	// DeleteVersion deletes a specific version row (ELEMENT_PROPERTY rows cascade).
	DeleteVersion(tx dbmodel.TxInterface, versionID, orgID string) error

	// DeleteElement deletes all versions of an element. Called when the last version is removed.
	DeleteElement(tx dbmodel.TxInterface, elementID, orgID string) error

	// IsVersionReferencedByPurpose reports whether any purpose version references this element version.
	// Returns true → caller must reject the delete with 409 Conflict.
	IsVersionReferencedByPurpose(ctx context.Context, versionID, orgID string) (bool, error)
}

// ConsentPurposeStore defines the interface for purpose data operations.
// Each logical purpose is identified by an ID and has one or more immutable versions.
// Version 1 is created when the purpose is first created; subsequent versions are added via CreateVersion.
type ConsentPurposeStore interface {
	// CreateVersion inserts a new purpose version (PURPOSE row + PURPOSE_PROPERTY rows).
	// Element mappings are linked separately via LinkElementVersion.
	CreateVersion(tx dbmodel.TxInterface, version *purposeModel.PurposeVersion) error

	// GetLatestVersion returns the highest-numbered version of a purpose, with properties and elements.
	// Returns nil if not found.
	GetLatestVersion(ctx context.Context, purposeID, orgID string) (*purposeModel.PurposeVersion, error)

	// GetVersion returns a specific version by version number, with properties and elements.
	// Returns nil if not found.
	GetVersion(ctx context.Context, purposeID string, version int, orgID string) (*purposeModel.PurposeVersion, error)

	// GetVersionByID returns a purpose version by its VERSION_ID, with properties and elements.
	// Returns nil if not found.
	GetVersionByID(ctx context.Context, purposeVersionID, orgID string) (*purposeModel.PurposeVersion, error)

	// ListVersions returns all versions of one purpose ordered by version number ascending, with properties and elements.
	ListVersions(ctx context.Context, purposeID, orgID string) ([]purposeModel.PurposeVersion, error)

	// PurposeExists reports whether any version of the purpose exists.
	PurposeExists(ctx context.Context, purposeID, orgID string) (bool, error)

	// GetByNameAndGroupID returns the latest version of a purpose with the given name and groupID,
	// or nil if no such purpose exists. Used to enforce name uniqueness within a group before insert.
	GetByNameAndGroupID(ctx context.Context, name, groupID, orgID string) (*purposeModel.PurposeVersion, error)

	// ExistsByNameInOrg reports whether any purpose with the given name exists in the org,
	// regardless of which group owns it. Used to enforce that no two purposes (org-level or
	// group-scoped) share the same name within an org.
	ExistsByNameInOrg(ctx context.Context, name, orgID string) (bool, error)

	// DeleteVersion deletes a specific version row.
	// PURPOSE_PROPERTY and PURPOSE_ELEMENT_MAPPING rows cascade automatically.
	DeleteVersion(tx dbmodel.TxInterface, purposeVersionID, orgID string) error

	// DeletePurpose deletes all versions of a purpose. Called when the last version is removed.
	DeletePurpose(tx dbmodel.TxInterface, purposeID, orgID string) error

	// LinkElementVersion creates a mapping from a purpose version to an element version.
	LinkElementVersion(tx dbmodel.TxInterface, purposeVersionID, elementVersionID, orgID string, mandatory bool) error

	// GetVersionElements returns all element refs for a specific purpose version.
	GetVersionElements(ctx context.Context, purposeVersionID, orgID string) ([]purposeModel.PurposeMappedElement, error)

	// IsElementVersionUsed reports whether any purpose version references this element version.
	// Returns true → caller must reject the element version delete with 409 Conflict.
	IsElementVersionUsed(ctx context.Context, elementVersionID, orgID string) (bool, error)

	// IsVersionUsedInConsents reports whether any consent references this purpose version.
	// Returns true → caller must reject the purpose version delete with 409 Conflict.
	IsVersionUsedInConsents(ctx context.Context, purposeVersionID, orgID string) (bool, error)

	// List returns the latest version of each purpose matching the filters, with total count for pagination.
	// When filters.Details is false, Properties and Elements are not populated.
	List(ctx context.Context, orgID string, filters purposeModel.PurposeListFilter) ([]purposeModel.PurposeVersion, int, error)
}
