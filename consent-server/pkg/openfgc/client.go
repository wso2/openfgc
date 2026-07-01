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

// Package openfgc embeds the openfgc consent service in-process.
//
// The library owns its DB pool, logger, and transactions; it does not share
// these with the host. Schema migrations are the caller's responsibility.
package openfgc

import (
	"context"
	"fmt"

	"github.com/wso2/openfgc/consent-server/internal/authresource"
	authmodel "github.com/wso2/openfgc/consent-server/internal/authresource/model"
	"github.com/wso2/openfgc/consent-server/internal/consent"
	consentmodel "github.com/wso2/openfgc/consent-server/internal/consent/model"
	"github.com/wso2/openfgc/consent-server/internal/consentelement"
	elementmodel "github.com/wso2/openfgc/consent-server/internal/consentelement/model"
	"github.com/wso2/openfgc/consent-server/internal/consentpurpose"
	purposemodel "github.com/wso2/openfgc/consent-server/internal/consentpurpose/model"
	"github.com/wso2/openfgc/consent-server/internal/system/config"
	"github.com/wso2/openfgc/consent-server/internal/system/database/provider"
	"github.com/wso2/openfgc/consent-server/internal/system/error/serviceerror"
	"github.com/wso2/openfgc/consent-server/internal/system/stores"
)

type (
	// ConsentService manages consent lifecycle.
	ConsentService = consent.ConsentService
	// ConsentPurposeService manages consent purposes and their versions.
	ConsentPurposeService = consentpurpose.ConsentPurposeService
	// ConsentElementService manages consent elements and their versions.
	ConsentElementService = consentelement.ConsentElementService
	// AuthResourceService manages authorization resources attached to consents.
	AuthResourceService = authresource.AuthResourceServiceInterface
	// ServiceError is the error returned by every service method.
	ServiceError = serviceerror.ServiceError
)

type (
	// Consent is a CONSENT row.
	Consent = consentmodel.Consent
	// ConsentAttribute is a key/value pair attached to a consent.
	ConsentAttribute = consentmodel.ConsentAttribute
	// ConsentPurposeMapping links a consent to a specific purpose version.
	ConsentPurposeMapping = consentmodel.ConsentPurposeMapping
	// ConsentElementApproval records the user's decision on one element within a consent purpose.
	ConsentElementApproval = consentmodel.ConsentElementApproval
	// ConsentPurposeRow is a join of consent purpose mappings with purpose metadata.
	ConsentPurposeRow = consentmodel.ConsentPurposeRow
	// ConsentApprovalRow is a join of element approvals with element metadata.
	ConsentApprovalRow = consentmodel.ConsentApprovalRow
	// PurposeRef identifies a purpose by name with an optional version.
	PurposeRef = consentmodel.PurposeRef
	// ElementApprovalInput is one element's approval data in a create/update.
	ElementApprovalInput = consentmodel.ElementApprovalInput
	// ConsentPurposeInput is one purpose entry in a create/update.
	ConsentPurposeInput = consentmodel.ConsentPurposeInput
	// CreateConsentInput is the input to CreateConsent.
	CreateConsentInput = consentmodel.CreateConsentInput
	// UpdateConsentInput is the input to UpdateConsent.
	UpdateConsentInput = consentmodel.UpdateConsentInput
	// ConsentSearchFilter is the query filter for SearchConsents.
	ConsentSearchFilter = consentmodel.ConsentSearchFilter
	// ConsentElementApprovalOutput is the service-layer view of one element in a consent purpose.
	ConsentElementApprovalOutput = consentmodel.ConsentElementApprovalOutput
	// ConsentPurposeOutput is the service-layer view of one purpose in a consent.
	ConsentPurposeOutput = consentmodel.ConsentPurposeOutput
	// ConsentOutput is the service-layer view of a consent.
	ConsentOutput = consentmodel.ConsentOutput
	// ConsentListOutput is the result of SearchConsents.
	ConsentListOutput = consentmodel.ConsentListOutput
	// ConsentAttributeSearchOutput is the result of SearchConsentsByAttribute.
	ConsentAttributeSearchOutput = consentmodel.ConsentAttributeSearchOutput
	// ConsentRevokeInput is the input to RevokeConsent.
	ConsentRevokeInput = consentmodel.ConsentRevokeInput
	// ConsentRevokeOutput is the result of RevokeConsent.
	ConsentRevokeOutput = consentmodel.ConsentRevokeOutput
	// ResourceParamsInput carries optional resource context for ValidateConsent.
	ResourceParamsInput = consentmodel.ResourceParamsInput
	// ConsentValidateInput is the input to ValidateConsent.
	ConsentValidateInput = consentmodel.ConsentValidateInput
	// ConsentValidateOutput is the result of ValidateConsent.
	ConsentValidateOutput = consentmodel.ConsentValidateOutput
	// ConsentHistory is one pre-mutation snapshot of a consent.
	ConsentHistory = consentmodel.ConsentHistory
	// ConsentHistoryOutput is the service-layer view of one history entry.
	ConsentHistoryOutput = consentmodel.ConsentHistoryOutput
	// ConsentHistoryListOutput is the result of GetConsentHistory.
	ConsentHistoryListOutput = consentmodel.ConsentHistoryListOutput
	// StatusAuditOutput is one entry in a consent's status transition audit trail.
	StatusAuditOutput = consentmodel.StatusAuditOutput
)

type (
	// PurposeVersion is a row from PURPOSE_VERSION.
	PurposeVersion = purposemodel.PurposeVersion
	// PurposeVersionProperty is a property attached to a purpose version.
	PurposeVersionProperty = purposemodel.PurposeVersionProperty
	// PurposeMappedElement is an element mapped into a purpose version.
	PurposeMappedElement = purposemodel.PurposeMappedElement
	// ElementRef identifies an element by name and namespace with an optional version.
	ElementRef = purposemodel.ElementRef
	// CreatePurposeInput is the input to CreatePurpose.
	CreatePurposeInput = purposemodel.CreatePurposeInput
	// CreatePurposeVersionInput is the input to CreatePurposeVersion.
	CreatePurposeVersionInput = purposemodel.CreatePurposeVersionInput
	// PurposeListFilter is the query filter for ListPurposes.
	PurposeListFilter = purposemodel.PurposeListFilter
	// PurposeElementOutput is the service-layer view of one element in a purpose.
	PurposeElementOutput = purposemodel.PurposeElementOutput
	// PurposeOutput is the service-layer view of a purpose.
	PurposeOutput = purposemodel.PurposeOutput
	// PurposeListOutput is the result of ListPurposes.
	PurposeListOutput = purposemodel.PurposeListOutput
	// PurposeVersionListOutput is the result of GetPurposeVersions.
	PurposeVersionListOutput = purposemodel.PurposeVersionListOutput
)

type (
	// ElementVersion is a versioned element definition.
	ElementVersion = elementmodel.ElementVersion
	// ElementVersionProperty is a property attached to an element version.
	ElementVersionProperty = elementmodel.ElementVersionProperty
	// CreateElementInput is one element entry for CreateElementsInBatch.
	CreateElementInput = elementmodel.CreateElementInput
	// CreateElementVersionInput is the input to CreateElementVersion.
	CreateElementVersionInput = elementmodel.CreateElementVersionInput
	// ElementListFilter is the query filter for ListElements.
	ElementListFilter = elementmodel.ElementListFilter
	// CreateElementOutput is one per-item result in CreateElementsInBatch.
	CreateElementOutput = elementmodel.CreateElementOutput
	// BatchCreateOutput is the result of CreateElementsInBatch.
	BatchCreateOutput = elementmodel.BatchCreateOutput
	// ElementListOutput is the result of ListElements.
	ElementListOutput = elementmodel.ElementListOutput
	// ElementVersionListOutput is the result of ListElementVersions.
	ElementVersionListOutput = elementmodel.ElementVersionListOutput
)

type (
	// AuthResource is a row from CONSENT_AUTH_RESOURCE.
	AuthResource = authmodel.AuthResource
	// CreateAuthResourceInput is the input to CreateAuthResource.
	CreateAuthResourceInput = authmodel.CreateAuthResourceInput
	// UpdateAuthResourceInput is the input to UpdateAuthResource.
	UpdateAuthResourceInput = authmodel.UpdateAuthResourceInput
	// AuthResourceOutput is the service-layer view of an auth resource.
	AuthResourceOutput = authmodel.AuthResourceOutput
	// AuthResourceListOutput is the result of GetAuthResourcesByConsentID.
	AuthResourceListOutput = authmodel.AuthResourceListOutput
)

// Client is an embedded openfgc instance.
type Client struct {
	consents      ConsentService
	purposes      ConsentPurposeService
	elements      ConsentElementService
	authResources AuthResourceService
}

// New constructs a Client from cfg and opens the database connection.
func New(cfg Config) (*Client, error) {
	if err := config.SetFromStruct(cfg.toInternal()); err != nil {
		return nil, fmt.Errorf("openfgc: %w", err)
	}

	storeRegistry := stores.NewStoreRegistry(
		consent.NewConsentStore(),
		authresource.NewAuthResourceStore(),
		consentelement.NewConsentElementStore(),
		consentpurpose.NewPurposeStore(),
	)

	// Surface DB errors at New time, not at first call.
	if _, err := provider.GetDBProvider().GetConsentDBClient(); err != nil {
		return nil, fmt.Errorf("openfgc: failed to initialize database: %w", err)
	}

	return &Client{
		consents:      consent.NewConsentService(storeRegistry),
		purposes:      consentpurpose.NewConsentPurposeService(storeRegistry),
		elements:      consentelement.NewConsentElementService(storeRegistry),
		authResources: authresource.NewAuthResourceService(storeRegistry),
	}, nil
}

// Consents returns the consent service.
func (c *Client) Consents() ConsentService { return c.consents }

// Purposes returns the consent purpose service.
func (c *Client) Purposes() ConsentPurposeService { return c.purposes }

// Elements returns the consent element service.
func (c *Client) Elements() ConsentElementService { return c.elements }

// AuthResources returns the auth resource service.
func (c *Client) AuthResources() AuthResourceService { return c.authResources }

// Shutdown closes the database connection pool.
func (c *Client) Shutdown(_ context.Context) error {
	if err := provider.GetDBProviderCloser().Close(); err != nil {
		return fmt.Errorf("openfgc: shutdown: %w", err)
	}
	return nil
}
