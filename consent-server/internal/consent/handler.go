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
	"fmt"
	"net/http"
	"strconv"
	"strings"

	authmodel "github.com/wso2/openfgc/internal/authresource/model"
	"github.com/wso2/openfgc/internal/consent/model"
	"github.com/wso2/openfgc/internal/consent/validator"
	elementmodel "github.com/wso2/openfgc/internal/consentelement/model"
	"github.com/wso2/openfgc/internal/system/constants"
	"github.com/wso2/openfgc/internal/system/error/serviceerror"
	"github.com/wso2/openfgc/internal/system/utils"
)

type consentHandler struct {
	service ConsentService
}

func newConsentHandler(service ConsentService) *consentHandler {
	return &consentHandler{service: service}
}

// =============================================================================
// HTTP handlers
// =============================================================================

// createConsent handles POST /consents
func (h *consentHandler) createConsent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.Header.Get(constants.HeaderOrgID)
	groupID := r.Header.Get(constants.HeaderGroupID)

	if err := utils.ValidateOrgID(orgID); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error()))
		return
	}

	var req model.ConsentCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorInvalidRequestBody, "invalid request body"))
		return
	}

	if err := validator.ValidateConsentCreateRequest(req, groupID, orgID); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error()))
		return
	}

	input, err := requestToCreateInput(req, groupID)
	if err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error()))
		return
	}

	out, serviceErr := h.service.CreateConsent(ctx, input, orgID)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(consentOutputToResponse(out))
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

	includeStatusHistory := r.URL.Query().Get("includeStatusHistory") == "true"
	var out *model.ConsentOutput
	var serviceErr *serviceerror.ServiceError
	if includeStatusHistory {
		out, serviceErr = h.service.GetConsentWithStatusHistory(ctx, consentID, orgID)
	} else {
		out, serviceErr = h.service.GetConsent(ctx, consentID, orgID)
	}
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	json.NewEncoder(w).Encode(consentOutputToResponse(out))
}

// getConsentHistory handles GET /consents/{consentId}/history
func (h *consentHandler) getConsentHistory(w http.ResponseWriter, r *http.Request) {
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

	includeSnapshots := r.URL.Query().Get("includeSnapshots") == "true"
	out, serviceErr := h.service.GetConsentHistory(ctx, consentID, orgID, includeSnapshots)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	json.NewEncoder(w).Encode(consentHistoryOutputToResponse(out))
}

// listConsents handles GET /consents
func (h *consentHandler) listConsents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.Header.Get(constants.HeaderOrgID)

	if err := utils.ValidateOrgID(orgID); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error()))
		return
	}

	limit := 10
	offset := 0
	const maxLimit = 100

	if s := r.URL.Query().Get("limit"); s != "" {
		if l, err := strconv.Atoi(s); err == nil && l > 0 {
			if l > maxLimit {
				limit = maxLimit
			} else {
				limit = l
			}
		}
	}
	if s := r.URL.Query().Get("offset"); s != "" {
		if o, err := strconv.Atoi(s); err == nil && o >= 0 {
			offset = o
		}
	}

	filters := model.ConsentSearchFilter{
		OrgID:  orgID,
		Limit:  limit,
		Offset: offset,
	}

	// groupIds (replaces clientIds)
	if s := r.URL.Query().Get("groupIds"); s != "" {
		parts := strings.Split(s, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		filters.GroupIDs = parts
	}

	// consentTypes
	if s := r.URL.Query().Get("consentTypes"); s != "" {
		parts := strings.Split(s, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		filters.ConsentTypes = parts
	}

	// consentStatuses
	if s := r.URL.Query().Get("consentStatuses"); s != "" {
		parts := strings.Split(s, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		filters.ConsentStatuses = parts
	}

	// userIds
	if s := r.URL.Query().Get("userIds"); s != "" {
		parts := strings.Split(s, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		filters.UserIDs = parts
	}

	// delegation (boolean) — filters by delegation role when combined with userIds
	// true = user is a delegate, false = user's own self-consents
	if s := r.URL.Query().Get("delegation"); s != "" {
		switch strings.ToLower(s) {
		case "true":
			v := true
			filters.Delegation = &v
		case "false":
			v := false
			filters.Delegation = &v
		default:
			utils.SendError(w, r, serviceerror.CustomServiceError(ErrorValidationFailed,
				"delegation must be 'true' or 'false'"))
			return
		}
	}

	// delegateSubject — filter consents where this userId is the delegate subject
	filters.DelegateSubject = r.URL.Query().Get("delegateSubject")

	// authTypes — filter by specific auth type values (e.g., "agent", "carer")
	if s := r.URL.Query().Get("authTypes"); s != "" {
		parts := strings.Split(s, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		filters.AuthTypes = parts
	}

	// purposeName (single) + optional purposeVersion
	filters.PurposeName = r.URL.Query().Get("purposeName")
	if s := r.URL.Query().Get("purposeVersion"); s != "" {
		v, err := parseVersionString(s)
		if err != nil {
			utils.SendError(w, r, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error()))
			return
		}
		filters.PurposeVersion = &v
	}

	// elementName + optional elementNamespace + elementVersion
	filters.ElementName = r.URL.Query().Get("elementName")
	filters.ElementNamespace = r.URL.Query().Get("elementNamespace")
	if s := r.URL.Query().Get("elementVersion"); s != "" {
		v, err := parseVersionString(s)
		if err != nil {
			utils.SendError(w, r, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error()))
			return
		}
		filters.ElementVersion = &v
	}

	// fromTime / toTime (Unix ms)
	if s := r.URL.Query().Get("fromTime"); s != "" {
		if ft, err := strconv.ParseInt(s, 10, 64); err == nil {
			filters.FromTime = &ft
		}
	}
	if s := r.URL.Query().Get("toTime"); s != "" {
		if tt, err := strconv.ParseInt(s, 10, 64); err == nil {
			filters.ToTime = &tt
		}
	}

	// purposeVersion requires purposeName
	if filters.PurposeVersion != nil && filters.PurposeName == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorValidationFailed,
			"purposeVersion requires purposeName to be specified"))
		return
	}
	// elementVersion requires elementName or elementNamespace
	if filters.ElementVersion != nil && filters.ElementName == "" && filters.ElementNamespace == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorValidationFailed,
			"elementVersion requires elementName or elementNamespace to be specified"))
		return
	}
	// delegation and authTypes are mutually exclusive — use one or the other
	if filters.Delegation != nil && len(filters.AuthTypes) > 0 {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorValidationFailed,
			"delegation and authTypes cannot be used together; use delegation for first-class types or authTypes for custom types"))
		return
	}

	listOut, serviceErr := h.service.SearchConsents(ctx, filters)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	json.NewEncoder(w).Encode(consentListOutputToResponse(listOut))
}

// updateConsent handles PUT /consents/{consentId}
func (h *consentHandler) updateConsent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	consentID := r.PathValue("consentId")
	orgID := r.Header.Get(constants.HeaderOrgID)
	groupID := r.Header.Get(constants.HeaderGroupID)

	if err := utils.ValidateOrgID(orgID); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error()))
		return
	}
	if err := utils.ValidateConsentID(consentID); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error()))
		return
	}

	var req model.ConsentUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorInvalidRequestBody, "invalid request body"))
		return
	}

	if err := validator.ValidateConsentUpdateRequest(req); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error()))
		return
	}

	input, err := requestToUpdateInput(req)
	if err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error()))
		return
	}

	out, serviceErr := h.service.UpdateConsent(ctx, consentID, groupID, orgID, input)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(consentOutputToResponse(out))
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
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorInvalidRequestBody, "invalid request body"))
		return
	}

	revokeOut, serviceErr := h.service.RevokeConsent(ctx, consentID, orgID, model.ConsentRevokeInput{
		ActionBy: req.ActionBy,
		Reason:   req.RevocationReason,
	})
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(model.ConsentRevokeResponse{
		ActionTime:       revokeOut.ActionTime,
		ActionBy:         revokeOut.ActionBy,
		RevocationReason: revokeOut.Reason,
	})
}

// validateConsent handles POST /consents/validate
func (h *consentHandler) validateConsent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.Header.Get(constants.HeaderOrgID)

	if err := utils.ValidateOrgID(orgID); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error()))
		return
	}

	var req model.ConsentValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorInvalidRequestBody, "invalid request body"))
		return
	}

	validateOut, serviceErr := h.service.ValidateConsent(ctx, validateRequestToInput(req), orgID)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	// Always HTTP 200; validity is in the response body
	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(validateOutputToResponse(validateOut))
}

// searchConsentsByAttribute handles GET /consents/attributes
func (h *consentHandler) searchConsentsByAttribute(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.Header.Get(constants.HeaderOrgID)

	if err := utils.ValidateOrgID(orgID); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorValidationFailed, err.Error()))
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorValidationFailed, "key parameter is required"))
		return
	}
	value := r.URL.Query().Get("value")

	out, serviceErr := h.service.SearchConsentsByAttribute(ctx, key, value, orgID)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&model.ConsentAttributeSearchResponse{
		ConsentIDs: out.ConsentIDs,
		Count:      out.Count,
	})
}

// =============================================================================
// Request → service input converters
// =============================================================================

// requestToCreateInput converts a ConsentCreateRequest body + groupID header
// into the service-layer CreateConsentInput.
func requestToCreateInput(req model.ConsentCreateRequest, groupID string) (model.CreateConsentInput, error) {
	purposes, err := parsePurposeRefRequests(req.Purposes)
	if err != nil {
		return model.CreateConsentInput{}, err
	}

	auths := make([]authmodel.CreateAuthResourceInput, 0, len(req.Authorizations))
	for _, ar := range req.Authorizations {
		auths = append(auths, authorizationRequestToInput(ar))
	}

	return model.CreateConsentInput{
		GroupID:                    groupID,
		ConsentType:                req.Type,
		ExpirationTime:             toExpirationMillis(req.ExpirationTime),
		ConsentFrequency:           req.Frequency,
		RecurringIndicator:         req.RecurringIndicator,
		DataAccessValidityDuration: req.DataAccessValidityDuration,
		Attributes:                 req.Attributes,
		Purposes:                   purposes,
		Authorizations:             auths,
	}, nil
}

// requestToUpdateInput converts a ConsentUpdateRequest body into UpdateConsentInput.
// Nil slice/map fields mean "caller omitted the field" → preserve existing.
// Non-nil (including empty) means "caller explicitly set" → replace.
func requestToUpdateInput(req model.ConsentUpdateRequest) (model.UpdateConsentInput, error) {
	var purposes []model.ConsentPurposeInput
	if req.Purposes != nil {
		var err error
		purposes, err = parsePurposeRefRequests(req.Purposes)
		if err != nil {
			return model.UpdateConsentInput{}, err
		}
	}

	var auths []authmodel.CreateAuthResourceInput
	if req.Authorizations != nil {
		auths = make([]authmodel.CreateAuthResourceInput, 0, len(req.Authorizations))
		for _, ar := range req.Authorizations {
			auths = append(auths, authorizationRequestToInput(ar))
		}
	}

	return model.UpdateConsentInput{
		ConsentType:                req.Type,
		ExpirationTime:             toExpirationMillis(req.ExpirationTime),
		ConsentFrequency:           req.Frequency,
		RecurringIndicator:         req.RecurringIndicator,
		DataAccessValidityDuration: req.DataAccessValidityDuration,
		Attributes:                 req.Attributes,
		Purposes:                   purposes,
		Authorizations:             auths,
	}, nil
}

// parsePurposeRefRequests converts API purpose references to service-layer input structs.
// Version strings ("v1", "v2", …) are parsed into integer version numbers.
func parsePurposeRefRequests(reqs []model.ConsentPurposeRefRequest) ([]model.ConsentPurposeInput, error) {
	inputs := make([]model.ConsentPurposeInput, 0, len(reqs))
	for _, pr := range reqs {
		var version *int
		if pr.Version != nil {
			v, err := parseVersionString(*pr.Version)
			if err != nil {
				return nil, fmt.Errorf("purpose %q: %w", pr.Name, err)
			}
			version = &v
		}

		elements := make([]model.ElementApprovalInput, 0, len(pr.Elements))
		for _, e := range pr.Elements {
			ns := e.Namespace
			if ns == "" {
				ns = elementmodel.DefaultNamespace
			}
			elements = append(elements, model.ElementApprovalInput{
				Name:      e.Name,
				Namespace: ns,
				Approved:  e.Approved,
				Value:     e.Value,
			})
		}

		inputs = append(inputs, model.ConsentPurposeInput{
			PurposeRef: model.PurposeRef{
				PurposeName: pr.Name,
				Version:     version,
			},
			Elements: elements,
		})
	}
	return inputs, nil
}

// authorizationRequestToInput converts one AuthorizationRequest to CreateAuthResourceInput.
func authorizationRequestToInput(ar model.AuthorizationRequest) authmodel.CreateAuthResourceInput {
	var userIDPtr *string
	if ar.UserID != "" {
		userIDPtr = &ar.UserID
	}
	return authmodel.CreateAuthResourceInput{
		AuthType:   ar.Type,
		UserID:     userIDPtr,
		AuthStatus: ar.Status,
		Resources:  ar.Resources,
	}
}

// =============================================================================
// Service output → API response converters
// =============================================================================

// consentOutputToResponse converts a ConsentOutput to the JSON-ready ConsentResponse.
func consentOutputToResponse(out *model.ConsentOutput) *model.ConsentResponse {
	if out == nil {
		return nil
	}

	purposes := make([]model.ConsentPurposeResponse, 0, len(out.Purposes))
	for _, p := range out.Purposes {
		elements := make([]model.ConsentPurposeElementApprovalResponse, 0, len(p.Elements))
		for _, e := range p.Elements {
			elements = append(elements, model.ConsentPurposeElementApprovalResponse{
				ElementID: e.ElementID,
				Name:      e.Name,
				Namespace: e.Namespace,
				Version:   formatVersion(e.VersionNum),
				Mandatory: e.Mandatory,
				Approved:  e.Approved,
				Value:     valueStringToInterface(e.Value, e.ElementType),
			})
		}
		purposes = append(purposes, model.ConsentPurposeResponse{
			PurposeID: p.PurposeID,
			Name:      p.Name,
			Version:   formatVersion(p.VersionNum),
			Elements:  elements,
		})
	}

	auths := make([]model.AuthorizationResponse, 0, len(out.Authorizations))
	for _, ar := range out.Authorizations {
		auths = append(auths, model.AuthorizationResponse{
			ID:          ar.AuthID,
			UserID:      ar.UserID,
			Type:        ar.AuthType,
			Status:      ar.AuthStatus,
			UpdatedTime: ar.UpdatedTime,
			Resources:   ar.Resources,
		})
	}

	attrs := out.Attributes
	if attrs == nil {
		attrs = make(map[string]string)
	}

	statusHistory := make([]model.ConsentStatusAuditResponse, 0, len(out.StatusHistory))
	for _, audit := range out.StatusHistory {
		statusHistory = append(statusHistory, statusAuditOutputToResponse(audit))
	}

	return &model.ConsentResponse{
		ConsentID:                  out.ConsentID,
		GroupID:                    out.GroupID,
		Type:                       out.ConsentType,
		Status:                     out.CurrentStatus,
		CreatedTime:                out.CreatedTime,
		UpdatedTime:                out.UpdatedTime,
		ExpirationTime:             out.ExpirationTime,
		Frequency:                  out.ConsentFrequency,
		RecurringIndicator:         out.RecurringIndicator,
		DataAccessValidityDuration: out.DataAccessValidityDuration,
		Attributes:                 attrs,
		Purposes:                   purposes,
		Authorizations:             auths,
		StatusHistory:              statusHistory,
	}
}

// consentListOutputToResponse converts a ConsentListOutput to the JSON-ready ConsentListResponse.
func consentListOutputToResponse(out *model.ConsentListOutput) *model.ConsentListResponse {
	data := make([]model.ConsentResponse, 0, len(out.Data))
	for i := range out.Data {
		r := consentOutputToResponse(&out.Data[i])
		data = append(data, *r)
	}
	return &model.ConsentListResponse{
		Data: data,
		Metadata: model.ConsentListMetadata{
			Total:  out.Total,
			Offset: out.Offset,
			Count:  out.Count,
			Limit:  out.Limit,
		},
	}
}

// consentHistoryOutputToResponse converts a ConsentHistoryListOutput to the JSON-ready ConsentHistoryListResponse.
func consentHistoryOutputToResponse(out *model.ConsentHistoryListOutput) *model.ConsentHistoryListResponse {
	if out == nil {
		return nil
	}

	history := make([]model.ConsentHistoryResponse, 0, len(out.History))
	for _, item := range out.History {
		history = append(history, model.ConsentHistoryResponse{
			HistoryID:  item.HistoryID,
			ActionTime: item.ActionTime,
			ActionBy:   item.ActionBy,
			Reason:     item.Reason,
			Snapshot:   item.Snapshot,
		})
	}

	return &model.ConsentHistoryListResponse{
		ID:      out.ID,
		History: history,
	}
}

// statusAuditOutputToResponse converts a StatusAuditOutput to the JSON-ready ConsentStatusAuditResponse.
func statusAuditOutputToResponse(out model.StatusAuditOutput) model.ConsentStatusAuditResponse {
	return model.ConsentStatusAuditResponse{
		StatusAuditID:  out.StatusAuditID,
		PreviousStatus: out.PreviousStatus,
		CurrentStatus:  out.CurrentStatus,
		ActionTime:     out.ActionTime,
		ActionBy:       out.ActionBy,
		Reason:         out.Reason,
	}
}

// valueStringToInterface parses a stored value string for use in API responses.
// For "json" element types the string is unmarshalled back to an interface{};
// for all other types the raw string is returned as-is.
func valueStringToInterface(v *string, elemType string) interface{} {
	if v == nil {
		return nil
	}
	if elemType == "json" {
		var parsed interface{}
		if err := json.Unmarshal([]byte(*v), &parsed); err == nil {
			return parsed
		}
	}
	return *v
}

// validateRequestToInput converts a ConsentValidateRequest (API type) to ConsentValidateInput (service type).
func validateRequestToInput(req model.ConsentValidateRequest) model.ConsentValidateInput {
	input := model.ConsentValidateInput{
		ConsentID:       req.ConsentID,
		GroupID:         req.GroupID,
		UserID:          req.UserID,
		Headers:         req.Headers,
		Payload:         req.Payload,
		ElectedResource: req.ElectedResource,
	}
	if req.ResourceParams != nil {
		input.ResourceParams = &model.ResourceParamsInput{
			Resource:   req.ResourceParams.Resource,
			HTTPMethod: req.ResourceParams.HTTPMethod,
			Context:    req.ResourceParams.Context,
		}
	}
	return input
}

// validateOutputToResponse converts a ConsentValidateOutput (service type) to ConsentValidateResponse (API type).
func validateOutputToResponse(out *model.ConsentValidateOutput) *model.ConsentValidateResponse {
	return &model.ConsentValidateResponse{
		IsValid:          out.IsValid,
		ErrorCode:        out.ErrorCode,
		ErrorMessage:     out.ErrorMessage,
		ErrorDescription: out.ErrorDescription,
		ConsentInfo:      consentOutputToValidateInfo(out.ConsentInfo),
	}
}

// consentOutputToValidateInfo converts a ConsentOutput to the enriched ConsentValidateInfo
// used in the validate endpoint response. Element type/description/displayName are included
// from the approval rows (populated via JOIN in GetElementApprovalsByConsentID).
func consentOutputToValidateInfo(out *model.ConsentOutput) *model.ConsentValidateInfo {
	if out == nil {
		return nil
	}

	purposes := make([]model.ConsentValidatePurposeResponse, 0, len(out.Purposes))
	for _, p := range out.Purposes {
		elements := make([]model.ConsentValidatePurposeElementResponse, 0, len(p.Elements))
		for _, e := range p.Elements {
			elements = append(elements, model.ConsentValidatePurposeElementResponse{
				ElementID:   e.ElementID,
				Name:        e.Name,
				Namespace:   e.Namespace,
				Version:     formatVersion(e.VersionNum),
				Mandatory:   e.Mandatory,
				Approved:    e.Approved,
				Value:       valueStringToInterface(e.Value, e.ElementType),
				DisplayName: e.DisplayName,
				Type:        e.ElementType,
				Description: e.Description,
				Properties:  e.Properties,
			})
		}
		purposes = append(purposes, model.ConsentValidatePurposeResponse{
			PurposeID:   p.PurposeID,
			Name:        p.Name,
			Version:     formatVersion(p.VersionNum),
			DisplayName: p.DisplayName,
			Description: p.Description,
			Properties:  p.Properties,
			Elements:    elements,
		})
	}

	auths := make([]model.AuthorizationResponse, 0, len(out.Authorizations))
	for _, ar := range out.Authorizations {
		auths = append(auths, model.AuthorizationResponse{
			ID:          ar.AuthID,
			UserID:      ar.UserID,
			Type:        ar.AuthType,
			Status:      ar.AuthStatus,
			UpdatedTime: ar.UpdatedTime,
			Resources:   ar.Resources,
		})
	}

	attrs := out.Attributes
	if attrs == nil {
		attrs = make(map[string]string)
	}

	return &model.ConsentValidateInfo{
		ConsentID:                  out.ConsentID,
		GroupID:                    out.GroupID,
		Type:                       out.ConsentType,
		Status:                     out.CurrentStatus,
		CreatedTime:                out.CreatedTime,
		UpdatedTime:                out.UpdatedTime,
		ExpirationTime:             out.ExpirationTime,
		Frequency:                  out.ConsentFrequency,
		RecurringIndicator:         out.RecurringIndicator,
		DataAccessValidityDuration: out.DataAccessValidityDuration,
		Attributes:                 attrs,
		Purposes:                   purposes,
		Authorizations:             auths,
	}
}

// toExpirationMillis normalises a client-supplied expiration timestamp to Unix milliseconds.
// Values >= 10^11 are already in milliseconds and are returned as-is.
// Values < 10^11 are treated as Unix seconds and multiplied by 1000.
// nil is returned unchanged.
func toExpirationMillis(t *int64) *int64 {
	if t == nil {
		return nil
	}
	const secondsCutoff = int64(100_000_000_000)
	ms := *t
	if ms < secondsCutoff {
		ms *= 1000
	}
	return &ms
}
