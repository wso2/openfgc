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
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/wso2/openfgc/internal/authresource/model"
	"github.com/wso2/openfgc/internal/system/constants"
	"github.com/wso2/openfgc/internal/system/error/serviceerror"
	"github.com/wso2/openfgc/internal/system/utils"
)

// authResourceHandler handles HTTP requests for auth resources
type authResourceHandler struct {
	service AuthResourceServiceInterface
}

// newAuthResourceHandler creates a new auth resource handler
func newAuthResourceHandler(service AuthResourceServiceInterface) *authResourceHandler {
	return &authResourceHandler{
		service: service,
	}
}

// handleCreate handles POST /consents/{consentId}/authorizations
func (h *authResourceHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract path parameters
	consentID := r.PathValue("consentId")
	if consentID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(
			ErrorConsentIDRequired,
			"consent ID is required",
		))
		return
	}

	// Extract organization ID from header
	orgID := r.Header.Get(constants.HeaderOrgID)
	if orgID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(
			ErrorOrgIDRequired,
			"organization ID header is required",
		))
		return
	}

	// Parse request body
	var request model.CreateRequest
	if err := utils.DecodeJSONBody(r, &request); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(
			ErrorInvalidRequestBody,
			fmt.Sprintf("invalid request body: %v", err),
		))
		return
	}

	// Call service
	response, serviceErr := h.service.CreateAuthResource(ctx, consentID, orgID, &request)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	// Send response
	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handleGet handles GET /consents/{consentId}/authorizations/{authorizationId}
func (h *authResourceHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract path parameters
	consentID := r.PathValue("consentId")
	authID := r.PathValue("authorizationId")

	if consentID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(
			ErrorConsentIDRequired,
			"consent ID is required",
		))
		return
	}

	if authID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(
			ErrorAuthResourceIDRequired,
			"authorization ID is required",
		))
		return
	}

	// Extract organization ID from header
	orgID := r.Header.Get(constants.HeaderOrgID)
	if orgID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(
			ErrorOrgIDRequired,
			"organization ID header is required",
		))
		return
	}

	// Call service
	response, serviceErr := h.service.GetAuthResource(ctx, authID, consentID, orgID)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	// Send response
	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleListByConsent handles GET /consents/{consentId}/authorizations
func (h *authResourceHandler) handleListByConsent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract path parameters
	consentID := r.PathValue("consentId")
	if consentID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(
			ErrorConsentIDRequired,
			"consent ID is required",
		))
		return
	}

	// Extract organization ID from header
	orgID := r.Header.Get(constants.HeaderOrgID)
	if orgID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(
			ErrorOrgIDRequired,
			"organization ID header is required",
		))
		return
	}

	// Call service
	response, serviceErr := h.service.GetAuthResourcesByConsentID(ctx, consentID, orgID)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	// Send response
	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleUpdate handles PUT /consents/{consentId}/authorizations/{authorizationId}
func (h *authResourceHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract path parameters
	consentID := r.PathValue("consentId")
	authID := r.PathValue("authorizationId")
	if consentID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(
			ErrorConsentIDRequired,
			"consent ID is required",
		))
		return
	}

	if authID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(
			ErrorAuthResourceIDRequired,
			"authorization ID is required",
		))
		return
	}

	// Extract organization ID from header
	orgID := r.Header.Get(constants.HeaderOrgID)
	if orgID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(
			ErrorOrgIDRequired,
			"organization ID header is required",
		))
		return
	}

	// Parse request body
	var request model.UpdateRequest
	if err := utils.DecodeJSONBody(r, &request); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(
			ErrorInvalidRequestBody,
			fmt.Sprintf("invalid request body: %v", err),
		))
		return
	}

	// Call service
	response, serviceErr := h.service.UpdateAuthResource(ctx, authID, consentID, orgID, &request)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	// Send response
	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
