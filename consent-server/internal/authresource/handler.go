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
	"net/http"

	"github.com/wso2/openfgc/consent-server/internal/authresource/model"
	authvalidator "github.com/wso2/openfgc/consent-server/internal/authresource/validator"
	"github.com/wso2/openfgc/consent-server/internal/system/constants"
	"github.com/wso2/openfgc/consent-server/internal/system/error/serviceerror"
	"github.com/wso2/openfgc/consent-server/internal/system/utils"
)

// authResourceHandler handles HTTP requests for auth resources.
type authResourceHandler struct {
	service AuthResourceServiceInterface
}

// newAuthResourceHandler creates a new auth resource handler.
func newAuthResourceHandler(service AuthResourceServiceInterface) *authResourceHandler {
	return &authResourceHandler{service: service}
}

// handleCreate handles POST /consents/{consentId}/authorizations
func (h *authResourceHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	consentID := r.PathValue("consentId")
	orgID := r.Header.Get(constants.HeaderOrgID)

	if consentID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorConsentIDRequired, "consent ID is required"))
		return
	}
	if orgID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorOrgIDRequired, "organization ID header is required"))
		return
	}

	var req model.AuthResourceCreateRequest
	if err := utils.DecodeJSONBody(r, &req); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorInvalidRequestBody, "invalid request body"))
		return
	}

	if err := authvalidator.ValidateAuthResourceCreateRequest(req, consentID, orgID); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error()))
		return
	}

	out, serviceErr := h.service.CreateAuthResource(ctx, consentID, orgID, model.CreateAuthResourceInput{
		AuthType:   req.Type,
		UserID:     req.UserID,
		AuthStatus: req.Status,
		Resources:  req.Resources,
	})
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(authResourceOutputToResponse(out))
}

// handleGet handles GET /consents/{consentId}/authorizations/{authorizationId}
func (h *authResourceHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	consentID := r.PathValue("consentId")
	authID := r.PathValue("authorizationId")
	orgID := r.Header.Get(constants.HeaderOrgID)

	if consentID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorConsentIDRequired, "consent ID is required"))
		return
	}
	if authID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorAuthResourceIDRequired, "authorization ID is required"))
		return
	}
	if orgID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorOrgIDRequired, "organization ID header is required"))
		return
	}

	out, serviceErr := h.service.GetAuthResource(ctx, authID, consentID, orgID)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(authResourceOutputToResponse(out))
}

// handleListByConsent handles GET /consents/{consentId}/authorizations
func (h *authResourceHandler) handleListByConsent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	consentID := r.PathValue("consentId")
	orgID := r.Header.Get(constants.HeaderOrgID)

	if consentID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorConsentIDRequired, "consent ID is required"))
		return
	}
	if orgID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorOrgIDRequired, "organization ID header is required"))
		return
	}

	listOut, serviceErr := h.service.GetAuthResourcesByConsentID(ctx, consentID, orgID)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	data := make([]model.AuthResourceResponse, 0, len(listOut.Data))
	for _, o := range listOut.Data {
		data = append(data, authResourceOutputToResponse(&o))
	}

	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}

// handleUpdate handles PUT /consents/{consentId}/authorizations/{authorizationId}
func (h *authResourceHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	consentID := r.PathValue("consentId")
	authID := r.PathValue("authorizationId")
	orgID := r.Header.Get(constants.HeaderOrgID)

	if consentID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorConsentIDRequired, "consent ID is required"))
		return
	}
	if authID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorAuthResourceIDRequired, "authorization ID is required"))
		return
	}
	if orgID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorOrgIDRequired, "organization ID header is required"))
		return
	}

	var req model.AuthResourceUpdateRequest
	if err := utils.DecodeJSONBody(r, &req); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorInvalidRequestBody, "invalid request body"))
		return
	}

	if err := authvalidator.ValidateAuthResourceUpdateRequest(req); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error()))
		return
	}

	out, serviceErr := h.service.UpdateAuthResource(ctx, authID, consentID, orgID, model.UpdateAuthResourceInput{
		AuthType:   req.Type,
		UserID:     req.UserID,
		AuthStatus: req.Status,
		Resources:  req.Resources,
	})
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(authResourceOutputToResponse(out))
}

// authResourceOutputToResponse converts a service-layer AuthResourceOutput to the
// JSON-ready AuthResourceResponse for API responses.
func authResourceOutputToResponse(out *model.AuthResourceOutput) model.AuthResourceResponse {
	return model.AuthResourceResponse{
		ID:          out.AuthID,
		UserID:      out.UserID,
		Type:        out.AuthType,
		Status:      out.AuthStatus,
		UpdatedTime: out.UpdatedTime,
		Resources:   out.Resources,
	}
}
