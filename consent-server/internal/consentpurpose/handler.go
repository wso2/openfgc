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

package consentpurpose

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/wso2/openfgc/internal/consentpurpose/model"
	"github.com/wso2/openfgc/internal/system/constants"
	"github.com/wso2/openfgc/internal/system/error/serviceerror"
	"github.com/wso2/openfgc/internal/system/utils"
)

// consentPurposeHandler handles HTTP requests for consent purposes
type consentPurposeHandler struct {
	service ConsentPurposeService
}

// newConsentPurposeHandler creates a new consent purpose handler
func newConsentPurposeHandler(service ConsentPurposeService) *consentPurposeHandler {
	return &consentPurposeHandler{
		service: service,
	}
}

// createPurpose handles POST /consent-purposes
func (h *consentPurposeHandler) createPurpose(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.Header.Get(constants.HeaderOrgID)
	clientID := r.Header.Get(constants.HeaderTPPClientID)

	// Validate required headers
	if orgID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorOrgIDRequired, "org-id header is required"))
		return
	}
	if clientID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorClientIDRequired, "TPP-client-id header is required"))
		return
	}

	// Decode request
	var req model.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorInvalidRequestBody, "invalid request body"))
		return
	}

	// Create consent purpose
	purpose, serviceErr := h.service.CreatePurpose(ctx, req, orgID, clientID)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	// Return response
	response := purpose.ToResponse()
	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// getPurpose handles GET /consent-purposes/{purposeId}
func (h *consentPurposeHandler) getPurpose(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.Header.Get(constants.HeaderOrgID)
	purposeID := r.PathValue("purposeId")

	// Validate required headers
	if orgID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorOrgIDRequired, "org-id header is required"))
		return
	}

	if purposeID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorPurposeIDRequired, "purposeId is required"))
		return
	}

	// Get consent purpose
	purpose, serviceErr := h.service.GetPurpose(ctx, purposeID, orgID)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	// Return response
	response := purpose.ToResponse()
	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// listPurposes handles GET /consent-purposes
func (h *consentPurposeHandler) listPurposes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.Header.Get(constants.HeaderOrgID)

	// Validate required headers
	if orgID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorOrgIDRequired, "org-id header is required"))
		return
	}

	// Parse query parameters
	queryParams := r.URL.Query()

	// Parse name filter
	name := strings.TrimSpace(queryParams.Get("name"))

	// Parse clientIds (comma-separated)
	var clientIDs []string
	if clientIDsParam := queryParams.Get("clientIds"); clientIDsParam != "" {
		split := strings.Split(clientIDsParam, ",")
		// Trim spaces and filter out empty strings
		for i := range split {
			trimmed := strings.TrimSpace(split[i])
			if trimmed != "" {
				clientIDs = append(clientIDs, trimmed)
			}
		}
	}

	// Parse elementNames (comma-separated) - filter purposes by element names they contain
	var elementNames []string
	if elementNamesParam := queryParams.Get("elementNames"); elementNamesParam != "" {
		split := strings.Split(elementNamesParam, ",")
		// Trim spaces and filter out empty strings
		for i := range split {
			trimmed := strings.TrimSpace(split[i])
			if trimmed != "" {
				elementNames = append(elementNames, trimmed)
			}
		}
	}

	// Parse pagination parameters
	limit := 100
	if limitParam := queryParams.Get("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := 0
	if offsetParam := queryParams.Get("offset"); offsetParam != "" {
		if o, err := strconv.Atoi(offsetParam); err == nil && o >= 0 {
			offset = o
		}
	}

	// List consent purposes
	purposes, total, serviceErr := h.service.ListPurposes(ctx, orgID, name, clientIDs, elementNames, offset, limit)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	// Convert to response format
	responses := make([]model.Response, 0, len(purposes))
	for _, purpose := range purposes {
		responses = append(responses, purpose.ToResponse())
	}

	// Build response with pagination metadata
	response := model.ListResponse{
		Data: responses,
		Metadata: model.PaginationMetadata{
			Total:  total,
			Offset: offset,
			Count:  len(responses),
			Limit:  limit,
		},
	}

	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// updatePurpose handles PUT /consent-purposes/{purposeId}
func (h *consentPurposeHandler) updatePurpose(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.Header.Get(constants.HeaderOrgID)
	clientID := r.Header.Get(constants.HeaderTPPClientID)
	purposeID := r.PathValue("purposeId")

	// Validate required headers and parameters
	if orgID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorOrgIDRequired, "org-id header is required"))
		return
	}
	if clientID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorClientIDRequired, "TPP-client-id header is required"))
		return
	}
	if purposeID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorPurposeIDRequired, "purposeId is required"))
		return
	}

	// Decode request
	var req model.UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorInvalidRequestBody, "invalid request body"))
		return
	}

	// Update consent purpose
	purpose, serviceErr := h.service.UpdatePurpose(ctx, purposeID, req, orgID, clientID)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	// Return response
	response := purpose.ToResponse()
	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// deletePurpose handles DELETE /consent-purposes/{purposeId}
func (h *consentPurposeHandler) deletePurpose(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.Header.Get(constants.HeaderOrgID)
	purposeID := r.PathValue("purposeId")

	// Validate required headers and parameters
	if orgID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorOrgIDRequired, "org-id header is required"))
		return
	}
	if purposeID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorPurposeIDRequired, "purposeId is required"))
		return
	}

	// Delete consent purpose
	if serviceErr := h.service.DeletePurpose(ctx, purposeID, orgID); serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	// Return no content
	w.WriteHeader(http.StatusNoContent)
}
