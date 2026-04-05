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
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/wso2/openfgc/internal/consent/model"
	"github.com/wso2/openfgc/internal/consent/validator"
	"github.com/wso2/openfgc/internal/system/constants"
	"github.com/wso2/openfgc/internal/system/error/serviceerror"
	"github.com/wso2/openfgc/internal/system/utils"
)

type consentHandler struct {
	service ConsentService
}

func newConsentHandler(service ConsentService) *consentHandler {
	return &consentHandler{
		service: service,
	}
}

// createConsent handles POST /consents
func (h *consentHandler) createConsent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.Header.Get(constants.HeaderOrgID)
	clientID := r.Header.Get(constants.HeaderTPPClientID)

	if err := utils.ValidateOrgIdAndClientIdIsPresent(r); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error()))
		return
	}

	var req model.ConsentAPIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorInvalidRequestBody, "Invalid request body"))
		return
	}

	// Set CallerID from X-User-ID header so delegation validation in the service
	// can enforce that a person cannot create a delegated consent for themselves.
	// This field is NOT read from JSON (tagged json:"-")
	req.CallerID = r.Header.Get("X-User-ID")

	// Validate delegation attributes here in the handler so the check runs even
	// when the service is mocked in tests. ValidateDelegationAttributes is a no-op
	// when delegation.type is not present in Attributes.
	if err := validator.ValidateDelegationAttributes(req.Attributes, req.Authorizations, req.CallerID); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorInvalidDelegation, err.Error()))
		return
	}

	consent, serviceErr := h.service.CreateConsent(ctx, req, clientID, orgID)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	apiResponse := consent.ToAPIResponse()
	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(apiResponse)
}

// getConsent handles GET /consents/{consentId}
func (h *consentHandler) getConsent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	consentID := r.PathValue("consentId")
	orgID := r.Header.Get(constants.HeaderOrgID)

	if err := utils.ValidateOrgID(orgID); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error()))
		return
	}

	if err := utils.ValidateConsentID(consentID); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error()))
		return
	}

	consent, serviceErr := h.service.GetConsent(ctx, consentID, orgID)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	apiResponse := consent.ToAPIResponse()
	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	json.NewEncoder(w).Encode(apiResponse)
}

// listConsents handles GET /consents
func (h *consentHandler) listConsents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.Header.Get(constants.HeaderOrgID)

	if orgID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorValidationFailed, "Organization ID is required"))
		return
	}

	// Parse pagination parameters
	limit := 10
	offset := 0
	const maxLimit = 100

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			if l > maxLimit {
				limit = maxLimit
			} else {
				limit = l
			}
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Build search filters
	filters := model.ConsentSearchFilters{
		OrgID:  orgID,
		Limit:  limit,
		Offset: offset,
	}

	// CallerID from X-User-ID header — used to verify the caller is an authorised
	// delegate when dataPrincipalId is also provided
	filters.CallerID = r.Header.Get("X-User-ID")

	// dataPrincipalId filters consents by data subject (the person whose data was
	// consented to), not by who gave the consent.
	// Example: parent logs in → GET /consents?dataPrincipalId=child-user-id
	// The service checks the caller is a registered delegate before returning results.

	if dpID := r.URL.Query().Get("dataPrincipalId"); dpID != "" {
		filters.DataPrincipalID = strings.TrimSpace(dpID)
	}

	// Parse consentTypes (comma-separated)
	if consentTypesStr := r.URL.Query().Get("consentTypes"); consentTypesStr != "" {
		filters.ConsentTypes = strings.Split(consentTypesStr, ",")
		// Trim whitespace
		for i := range filters.ConsentTypes {
			filters.ConsentTypes[i] = strings.TrimSpace(filters.ConsentTypes[i])
		}
	}

	// Parse consentStatuses (comma-separated)
	if statusesStr := r.URL.Query().Get("consentStatuses"); statusesStr != "" {
		filters.ConsentStatuses = strings.Split(statusesStr, ",")
		for i := range filters.ConsentStatuses {
			filters.ConsentStatuses[i] = strings.TrimSpace(filters.ConsentStatuses[i])
		}
	}

	// Parse clientIds (comma-separated)
	if clientIDsStr := r.URL.Query().Get("clientIds"); clientIDsStr != "" {
		filters.ClientIDs = strings.Split(clientIDsStr, ",")
		for i := range filters.ClientIDs {
			filters.ClientIDs[i] = strings.TrimSpace(filters.ClientIDs[i])
		}
	}

	// Parse userIds (comma-separated)
	if userIDsStr := r.URL.Query().Get("userIds"); userIDsStr != "" {
		filters.UserIDs = strings.Split(userIDsStr, ",")
		for i := range filters.UserIDs {
			filters.UserIDs[i] = strings.TrimSpace(filters.UserIDs[i])
		}
	}

	// Parse purposeNames (comma-separated)
	if purposeNamesStr := r.URL.Query().Get("purposeNames"); purposeNamesStr != "" {
		filters.PurposeNames = strings.Split(purposeNamesStr, ",")
		for i := range filters.PurposeNames {
			filters.PurposeNames[i] = strings.TrimSpace(filters.PurposeNames[i])
		}
	}

	// Parse fromTime (Unix timestamp in milliseconds)
	if fromTimeStr := r.URL.Query().Get("fromTime"); fromTimeStr != "" {
		if ft, err := strconv.ParseInt(fromTimeStr, 10, 64); err == nil {
			filters.FromTime = &ft
		}
	}

	// Parse toTime (Unix timestamp in milliseconds)
	if toTimeStr := r.URL.Query().Get("toTime"); toTimeStr != "" {
		if tt, err := strconv.ParseInt(toTimeStr, 10, 64); err == nil {
			filters.ToTime = &tt
		}
	}

	// Use detailed search to include nested data
	response, serviceErr := h.service.SearchConsentsDetailed(ctx, filters)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	json.NewEncoder(w).Encode(response)
}

// updateConsent handles PUT /consents/{consentId}
func (h *consentHandler) updateConsent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	consentID := r.PathValue("consentId")
	orgID := r.Header.Get(constants.HeaderOrgID)
	clientID := r.Header.Get(constants.HeaderTPPClientID)

	if err := utils.ValidateOrgIdAndClientIdIsPresent(r); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error()))
		return
	}

	if err := utils.ValidateConsentID(consentID); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error()))
		return
	}

	var req model.ConsentAPIUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorInvalidRequestBody, "Invalid request body"))
		return
	}

	consent, serviceErr := h.service.UpdateConsent(ctx, req, clientID, orgID, consentID)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	apiResponse := consent.ToAPIResponse()
	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(apiResponse)
}

// revokeConsent handles POST /consents/{consentId}/revoke
func (h *consentHandler) revokeConsent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	consentID := r.PathValue("consentId")
	orgID := r.Header.Get(constants.HeaderOrgID)

	if err := utils.ValidateOrgID(orgID); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error()))
		return
	}

	if err := utils.ValidateConsentID(consentID); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error()))
		return
	}

	var req model.ConsentRevokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorInvalidRequestBody, "Invalid request body"))
		return
	}

	// The header is injected by the gateway/IdP and cannot be spoofed by the
	// client. Falling back to the JSON body value when the header is absent
	// would let any caller put an arbitrary userId in the body and bypass the
	// delegation revocation checks in the service layer.
	//
	// If X-User-ID is absent the request has no trusted caller identity.
	// Clear ActionBy so the service treats this as an anonymous/unauthenticated
	// call and applies the appropriate default behaviour (non-delegated path).
	userID := r.Header.Get("X-User-ID")
	req.ActionBy = userID

	revokeResponse, serviceErr := h.service.RevokeConsent(ctx, consentID, orgID, req)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(revokeResponse)
}

// validateConsent handles POST /consents/validate
func (h *consentHandler) validateConsent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.Header.Get(constants.HeaderOrgID)

	if err := utils.ValidateOrgID(orgID); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error()))
		return
	}

	var req model.ValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorInvalidRequestBody, "Invalid request body"))
		return
	}

	// Call service to validate consent
	response, serviceErr := h.service.ValidateConsent(ctx, req, orgID)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	// Always return HTTP 200, check isValid field in response
	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// searchConsentsByAttribute handles GET /consents/attributes
func (h *consentHandler) searchConsentsByAttribute(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.Header.Get(constants.HeaderOrgID)

	if err := utils.ValidateOrgID(orgID); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error()))
		return
	}

	// Get query parameters
	key := r.URL.Query().Get("key")
	value := r.URL.Query().Get("value")

	// Validate that key parameter is present
	if key == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorValidationFailed, "key parameter is required"))
		return
	}

	// Call service to search consents by attribute
	response, serviceErr := h.service.SearchConsentsByAttribute(ctx, key, value, orgID)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// getDelegates handles GET /consents/{consentId}/delegates
// Returns all registered delegates for the given consent, along with delegation
// metadata (principal_id, revocation policy, expiry).
func (h *consentHandler) getDelegates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	consentID := r.PathValue("consentId")
	orgID := r.Header.Get(constants.HeaderOrgID)

	if err := utils.ValidateOrgID(orgID); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error()))
		return
	}

	if err := utils.ValidateConsentID(consentID); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error()))
		return
	}

	response, serviceErr := h.service.GetConsentDelegates(ctx, consentID, orgID)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
