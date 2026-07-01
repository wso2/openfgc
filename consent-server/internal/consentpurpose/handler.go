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
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/wso2/openfgc/consent-server/internal/consentpurpose/model"
	"github.com/wso2/openfgc/consent-server/internal/system/constants"
	"github.com/wso2/openfgc/consent-server/internal/system/error/serviceerror"
	"github.com/wso2/openfgc/consent-server/internal/system/utils"
)

// consentPurposeHandler handles HTTP requests for consent purposes.
type consentPurposeHandler struct {
	service ConsentPurposeService
}

// newConsentPurposeHandler creates a new consent purpose handler.
func newConsentPurposeHandler(service ConsentPurposeService) *consentPurposeHandler {
	return &consentPurposeHandler{service: service}
}

// =============================================================================
// Handlers
// =============================================================================

// createPurpose handles POST /consent-purposes.
// The group-id header is optional; when absent the purpose is org-level (groupId = orgId).
func (h *consentPurposeHandler) createPurpose(w http.ResponseWriter, r *http.Request) {
	orgID := r.Header.Get(constants.HeaderOrgID)
	if orgID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorOrgIDRequired, "org-id header is required"))
		return
	}

	groupID := r.Header.Get(constants.HeaderGroupID)
	if groupID == "" {
		groupID = orgID // org-level purpose
	}

	var req model.CreatePurposeRequest
	if err := utils.DecodeJSONBody(r, &req); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorInvalidRequestBody, "invalid request body"))
		return
	}

	elementRefs, err := toElementRefs(req.Elements)
	if err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorInvalidRequestBody, err.Error()))
		return
	}

	input := model.CreatePurposeInput{
		Name:        req.Name,
		GroupID:     groupID,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Properties:  req.Properties,
		Elements:    elementRefs,
	}

	pv, svcErr := h.service.CreatePurpose(r.Context(), input, orgID)
	if svcErr != nil {
		utils.SendError(w, r, svcErr)
		return
	}

	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(purposeToResponse(pv))
}

// getPurpose handles GET /consent-purposes/{purposeId} — returns the latest version.
func (h *consentPurposeHandler) getPurpose(w http.ResponseWriter, r *http.Request) {
	orgID := r.Header.Get(constants.HeaderOrgID)
	if orgID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorOrgIDRequired, "org-id header is required"))
		return
	}

	purposeID := r.PathValue("purposeId")

	pv, svcErr := h.service.GetPurpose(r.Context(), purposeID, orgID)
	if svcErr != nil {
		utils.SendError(w, r, svcErr)
		return
	}

	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(purposeToResponse(pv))
}

// listPurposes handles GET /consent-purposes.
func (h *consentPurposeHandler) listPurposes(w http.ResponseWriter, r *http.Request) {
	orgID := r.Header.Get(constants.HeaderOrgID)
	if orgID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorOrgIDRequired, "org-id header is required"))
		return
	}

	q := r.URL.Query()

	var groupIDs []string
	if raw := q.Get("groupIds"); raw != "" {
		for _, g := range strings.Split(raw, ",") {
			if t := strings.TrimSpace(g); t != "" {
				groupIDs = append(groupIDs, t)
			}
		}
	}

	var purposeVersion *int
	if raw := q.Get("purposeVersion"); raw != "" {
		v, err := strconv.Atoi(strings.TrimPrefix(raw, "v"))
		if err != nil || v <= 0 {
			utils.SendError(w, r, serviceerror.CustomServiceError(ErrorInvalidQueryParams,
				"purposeVersion must be a positive integer in vN format (e.g. v1)"))
			return
		}
		purposeVersion = &v
	}

	var elementVersion *int
	if raw := q.Get("elementVersion"); raw != "" {
		v, err := strconv.Atoi(strings.TrimPrefix(raw, "v"))
		if err != nil || v <= 0 {
			utils.SendError(w, r, serviceerror.CustomServiceError(ErrorInvalidQueryParams,
				"elementVersion must be a positive integer in vN format (e.g. v1)"))
			return
		}
		elementVersion = &v
	}

	limit := 100
	if raw := q.Get("limit"); raw != "" {
		if l, err := strconv.Atoi(raw); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	offset := 0
	if raw := q.Get("offset"); raw != "" {
		if o, err := strconv.Atoi(raw); err == nil && o >= 0 {
			offset = o
		}
	}

	purposeName := strings.TrimSpace(q.Get("purposeName"))
	elementName := strings.TrimSpace(q.Get("elementName"))
	elementNamespace := strings.TrimSpace(q.Get("elementNamespace"))

	if purposeVersion != nil && purposeName == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorInvalidQueryParams,
			"purposeVersion requires purposeName to also be specified"))
		return
	}
	if elementVersion != nil && elementName == "" && elementNamespace == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorInvalidQueryParams,
			"elementVersion requires at least one of elementName or elementNamespace to also be specified"))
		return
	}

	filters := model.PurposeListFilter{
		GroupIDs:         groupIDs,
		PurposeName:      purposeName,
		PurposeVersion:   purposeVersion,
		ElementName:      elementName,
		ElementNamespace: elementNamespace,
		ElementVersion:   elementVersion,
		Details:          q.Get("details") == "true",
		Limit:            limit,
		Offset:           offset,
	}

	out, svcErr := h.service.ListPurposes(r.Context(), orgID, filters)
	if svcErr != nil {
		utils.SendError(w, r, svcErr)
		return
	}

	items := make([]model.PurposeResponse, 0, len(out.Data))
	for i := range out.Data {
		items = append(items, purposeToResponse(&out.Data[i]))
	}

	resp := model.PurposeListResponse{
		Data: items,
		Metadata: model.PageMetadata{
			Total:  out.Total,
			Offset: out.Offset,
			Count:  out.Count,
			Limit:  out.Limit,
		},
	}

	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// listPurposeVersions handles GET /consent-purposes/{purposeId}/versions.
func (h *consentPurposeHandler) listPurposeVersions(w http.ResponseWriter, r *http.Request) {
	orgID := r.Header.Get(constants.HeaderOrgID)
	if orgID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorOrgIDRequired, "org-id header is required"))
		return
	}

	purposeID := r.PathValue("purposeId")

	out, svcErr := h.service.GetPurposeVersions(r.Context(), purposeID, orgID)
	if svcErr != nil {
		utils.SendError(w, r, svcErr)
		return
	}

	versionItems := make([]model.PurposeVersionItem, 0, len(out.Versions))
	for i := range out.Versions {
		versionItems = append(versionItems, purposeToItem(&out.Versions[i]))
	}

	resp := model.PurposeVersionListResponse{
		PurposeID: out.PurposeID,
		Name:      out.Name,
		GroupID:   out.GroupID,
		Versions:  versionItems,
	}

	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// createPurposeVersion handles POST /consent-purposes/{purposeId}/versions.
func (h *consentPurposeHandler) createPurposeVersion(w http.ResponseWriter, r *http.Request) {
	orgID := r.Header.Get(constants.HeaderOrgID)
	if orgID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorOrgIDRequired, "org-id header is required"))
		return
	}

	purposeID := r.PathValue("purposeId")

	var req model.CreatePurposeVersionRequest
	if err := utils.DecodeJSONBody(r, &req); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorInvalidRequestBody, "invalid request body"))
		return
	}

	elementRefs, err := toElementRefs(req.Elements)
	if err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorInvalidRequestBody, err.Error()))
		return
	}

	input := model.CreatePurposeVersionInput{
		DisplayName: req.DisplayName,
		Description: req.Description,
		Properties:  req.Properties,
		Elements:    elementRefs,
	}

	pv, svcErr := h.service.CreatePurposeVersion(r.Context(), purposeID, input, orgID)
	if svcErr != nil {
		utils.SendError(w, r, svcErr)
		return
	}

	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(purposeToResponse(pv))
}

// getPurposeVersion handles GET /consent-purposes/{purposeId}/versions/{version}.
func (h *consentPurposeHandler) getPurposeVersion(w http.ResponseWriter, r *http.Request) {
	orgID := r.Header.Get(constants.HeaderOrgID)
	if orgID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorOrgIDRequired, "org-id header is required"))
		return
	}

	purposeID := r.PathValue("purposeId")
	versionNum, err := parseVersionParam(r.PathValue("version"))
	if err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorInvalidVersionFormat, err.Error()))
		return
	}

	pv, svcErr := h.service.GetPurposeVersion(r.Context(), purposeID, versionNum, orgID)
	if svcErr != nil {
		utils.SendError(w, r, svcErr)
		return
	}

	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(purposeToResponse(pv))
}

// deletePurposeVersion handles DELETE /consent-purposes/{purposeId}/versions/{version}.
func (h *consentPurposeHandler) deletePurposeVersion(w http.ResponseWriter, r *http.Request) {
	orgID := r.Header.Get(constants.HeaderOrgID)
	if orgID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorOrgIDRequired, "org-id header is required"))
		return
	}

	purposeID := r.PathValue("purposeId")
	versionNum, err := parseVersionParam(r.PathValue("version"))
	if err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorInvalidVersionFormat, err.Error()))
		return
	}

	if svcErr := h.service.DeletePurposeVersion(r.Context(), purposeID, versionNum, orgID); svcErr != nil {
		utils.SendError(w, r, svcErr)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// deletePurpose handles DELETE /consent-purposes/{purposeId}.
func (h *consentPurposeHandler) deletePurpose(w http.ResponseWriter, r *http.Request) {
	orgID := r.Header.Get(constants.HeaderOrgID)
	if orgID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(ErrorOrgIDRequired, "org-id header is required"))
		return
	}

	purposeID := r.PathValue("purposeId")

	if svcErr := h.service.DeletePurpose(r.Context(), purposeID, orgID); svcErr != nil {
		utils.SendError(w, r, svcErr)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// =============================================================================
// Mapping helpers
// =============================================================================

// purposeToResponse maps a PurposeOutput to its API response shape.
func purposeToResponse(p *model.PurposeOutput) model.PurposeResponse {
	resp := model.PurposeResponse{
		PurposeID:   p.ID,
		Name:        p.Name,
		GroupID:     p.GroupID,
		Version:     fmt.Sprintf("v%d", p.VersionNum),
		DisplayName: p.DisplayName,
		Description: p.Description,
		Properties:  p.Properties,
		CreatedTime: p.CreatedTime,
	}
	if len(p.Elements) > 0 {
		elems := make([]model.PurposeElementResponse, 0, len(p.Elements))
		for _, e := range p.Elements {
			elems = append(elems, model.PurposeElementResponse{
				ElementID: e.ElementID,
				Name:      e.Name,
				Namespace: e.Namespace,
				Version:   fmt.Sprintf("v%d", e.VersionNum),
				Mandatory: e.Mandatory,
			})
		}
		resp.Elements = elems
	}
	return resp
}

// purposeToItem maps a PurposeOutput to the compact version-list item shape.
func purposeToItem(p *model.PurposeOutput) model.PurposeVersionItem {
	item := model.PurposeVersionItem{
		Version:     fmt.Sprintf("v%d", p.VersionNum),
		DisplayName: p.DisplayName,
		Description: p.Description,
		Properties:  p.Properties,
		CreatedTime: p.CreatedTime,
	}
	if len(p.Elements) > 0 {
		elems := make([]model.PurposeElementResponse, 0, len(p.Elements))
		for _, e := range p.Elements {
			elems = append(elems, model.PurposeElementResponse{
				ElementID: e.ElementID,
				Name:      e.Name,
				Namespace: e.Namespace,
				Version:   fmt.Sprintf("v%d", e.VersionNum),
				Mandatory: e.Mandatory,
			})
		}
		item.Elements = elems
	}
	return item
}

// toElementRefs converts API request element refs to service input element refs.
// Returns an error if any version string is not in the expected "vN" format.
func toElementRefs(reqs []model.ElementRefRequest) ([]model.ElementRef, error) {
	refs := make([]model.ElementRef, 0, len(reqs))
	for _, r := range reqs {
		v, err := parseVersionString(r.Version)
		if err != nil {
			return nil, err
		}
		refs = append(refs, model.ElementRef{
			Name:      r.Name,
			Namespace: r.Namespace,
			Version:   v,
			Mandatory: r.Mandatory,
		})
	}
	return refs, nil
}

// parseVersionParam parses a version path segment (e.g. "v2") and returns the integer value.
// Returns an error if the format is invalid.
func parseVersionParam(v string) (int, error) {
	if len(v) < 2 || v[0] != 'v' {
		return 0, fmt.Errorf("version must be in the format vN (e.g. v1, v2), got '%s'", v)
	}
	n, err := strconv.Atoi(v[1:])
	if err != nil || n < 1 {
		return 0, fmt.Errorf("version must be in the format vN (e.g. v1, v2), got '%s'", v)
	}
	return n, nil
}

// parseVersionString converts a version string in "vN" format (e.g. "v1") to its integer value.
// Returns nil if s is nil, and an error if the format is unrecognised.
func parseVersionString(s *string) (*int, error) {
	if s == nil {
		return nil, nil
	}
	v := *s
	if len(v) < 2 || v[0] != 'v' {
		return nil, fmt.Errorf("version must be in the format vN (e.g. v1, v2), got '%s'", v)
	}
	n, err := strconv.Atoi(v[1:])
	if err != nil || n < 1 {
		return nil, fmt.Errorf("version must be in the format vN (e.g. v1, v2), got '%s'", v)
	}
	return &n, nil
}
