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
)

// TestCreatePurpose covers POST /consent-purposes.
//
// Element verification contract:
//   - When no version is specified in the request, the response must return the
//     resolved version (always "v1" for a freshly created element).
//   - When a version is pinned (e.g. "v1"), the response must echo that exact version.
//   - Every element in the response must have a non-empty elementId, the correct name,
//     namespace ("default" when not specified), version string, and mandatory flag.
func (ts *PurposeAPITestSuite) TestCreatePurpose() {
	type testCase struct {
		name string

		// buildBody creates any required elements and returns the full request body.
		// Used for happy-path cases that need real elements in scope.
		buildBody func(orgID string) any

		// rawBody is used for validation/error cases that fail before any element
		// look-up (malformed JSON, version parse errors, org-id missing, etc.).
		rawBody string

		// setup runs before buildBody; use for pre-conditions beyond the body itself
		// (e.g. a purpose that already exists to test duplicate detection).
		setup func(orgID string)

		groupID       string // group-id header value; "" = omit (server defaults to orgId)
		omitOrgID     bool
		wantStatus    int
		wantErrorCode string
		wantErrorDesc string // optional: assert Description contains this substring
		checkResult   func(orgID string, resp *PurposeResponse)
	}

	cases := []testCase{
		// -----------------------------------------------------------------------
		// group-id behaviour
		// -----------------------------------------------------------------------
		{
			name: "missing group-id → groupId stored as orgId (org-level purpose)",
			buildBody: func(orgID string) any {
				ts.mustCreateElement(orgID, "cp-org-elem", "basic")
				return CreatePurposeRequest{
					Name:     "cp-org-level",
					Elements: []ElementRefRequest{{Name: "cp-org-elem"}},
				}
			},
			wantStatus: http.StatusCreated,
			checkResult: func(orgID string, resp *PurposeResponse) {
				ts.assertPurposeResponse(resp, "cp-org-level")
				ts.Equal("v1", resp.Version)
				ts.Equal(orgID, resp.GroupID,
					"when group-id header is absent, groupId must equal orgId")
				ts.Require().Len(resp.Elements, 1)
				ts.assertPurposeElement(resp.Elements[0], "cp-org-elem", "default", "v1", false)
			},
		},
		{
			name:    "explicit group-id header — stored and returned as groupId",
			groupID: "my-group",
			buildBody: func(orgID string) any {
				ts.mustCreateElement(orgID, "cp-grp-elem", "basic")
				return CreatePurposeRequest{
					Name:     "cp-grouped",
					Elements: []ElementRefRequest{{Name: "cp-grp-elem", Mandatory: true}},
				}
			},
			wantStatus: http.StatusCreated,
			checkResult: func(_ string, resp *PurposeResponse) {
				ts.assertPurposeResponse(resp, "cp-grouped")
				ts.Equal("my-group", resp.GroupID)
				ts.Require().Len(resp.Elements, 1)
				ts.assertPurposeElement(resp.Elements[0], "cp-grp-elem", "default", "v1", true)
			},
		},

		// -----------------------------------------------------------------------
		// Element version resolution
		// -----------------------------------------------------------------------
		{
			name: "element version not specified → server resolves to latest (v1)",
			buildBody: func(orgID string) any {
				ts.mustCreateElement(orgID, "cp-auto-ver-elem", "basic")
				return CreatePurposeRequest{
					Name: "cp-auto-ver",
					Elements: []ElementRefRequest{
						{Name: "cp-auto-ver-elem"}, // no Version field → use latest
					},
				}
			},
			wantStatus: http.StatusCreated,
			checkResult: func(_ string, resp *PurposeResponse) {
				ts.Require().Len(resp.Elements, 1)
				ts.assertPurposeElement(resp.Elements[0], "cp-auto-ver-elem", "default", "v1", false)
			},
		},
		{
			name: "element pinned to v1 when v2 also exists — response confirms v1",
			buildBody: func(orgID string) any {
				elemID := ts.mustCreateElement(orgID, "cp-pin-elem", "basic")
				// Create v2 so "latest" would be v2 — the pin must override it.
				ts.doRequest(http.MethodPost,
					"/api/v1/consent-elements/"+elemID+"/versions",
					orgID, map[string]string{})
				return CreatePurposeRequest{
					Name: "cp-pinned",
					Elements: []ElementRefRequest{
						{Name: "cp-pin-elem", Version: ptr("v1"), Mandatory: true},
					},
				}
			},
			wantStatus: http.StatusCreated,
			checkResult: func(_ string, resp *PurposeResponse) {
				ts.Require().Len(resp.Elements, 1)
				ts.assertPurposeElement(resp.Elements[0], "cp-pin-elem", "default", "v1", true)
			},
		},
		{
			name: "no version specified — always resolves to current latest (v2 after element is upgraded)",
			buildBody: func(orgID string) any {
				// Start with v1 only; bind a first purpose to establish v1 usage.
				elemID := ts.mustCreateElement(orgID, "cp-up-elem", "basic")
				ts.mustCreatePurposeWith(orgID, "", CreatePurposeRequest{
					Name:     "cp-up-setup",
					Elements: []ElementRefRequest{{Name: "cp-up-elem"}}, // resolves to v1
				})
				// Upgrade the element to v2.
				ts.doRequest(http.MethodPost,
					"/api/v1/consent-elements/"+elemID+"/versions",
					orgID, map[string]string{})
				// New purpose created without version — must bind to v2, not v1.
				return CreatePurposeRequest{
					Name:     "cp-up-after",
					Elements: []ElementRefRequest{{Name: "cp-up-elem"}},
				}
			},
			wantStatus: http.StatusCreated,
			checkResult: func(_ string, resp *PurposeResponse) {
				ts.Require().Len(resp.Elements, 1)
				ts.assertPurposeElement(resp.Elements[0], "cp-up-elem", "default", "v2", false)
			},
		},

		// -----------------------------------------------------------------------
		// Multiple elements
		// -----------------------------------------------------------------------
		{
			name: "multiple elements — all returned with correct name, namespace, version, mandatory",
			buildBody: func(orgID string) any {
				ts.mustCreateElement(orgID, "cp-multi-email", "basic")
				ts.mustCreateElement(orgID, "cp-multi-phone", "basic")
				ts.mustCreateElement(orgID, "cp-multi-addr", "basic")
				return CreatePurposeRequest{
					Name: "cp-multi-elem",
					Elements: []ElementRefRequest{
						{Name: "cp-multi-email", Mandatory: true},
						{Name: "cp-multi-phone", Mandatory: false},
						{Name: "cp-multi-addr", Mandatory: true},
					},
				}
			},
			wantStatus: http.StatusCreated,
			checkResult: func(_ string, resp *PurposeResponse) {
				ts.assertPurposeResponse(resp, "cp-multi-elem")
				ts.Require().Len(resp.Elements, 3, "all three elements must be returned")
				// Index by name for order-independent assertions.
				byName := make(map[string]PurposeElementResponse, len(resp.Elements))
				for _, e := range resp.Elements {
					byName[e.Name] = e
				}
				ts.assertPurposeElement(byName["cp-multi-email"], "cp-multi-email", "default", "v1", true)
				ts.assertPurposeElement(byName["cp-multi-phone"], "cp-multi-phone", "default", "v1", false)
				ts.assertPurposeElement(byName["cp-multi-addr"], "cp-multi-addr", "default", "v1", true)
			},
		},

		// -----------------------------------------------------------------------
		// Optional metadata fields
		// -----------------------------------------------------------------------
		{
			name: "displayName, description, properties — all round-trip correctly",
			buildBody: func(orgID string) any {
				ts.mustCreateElement(orgID, "cp-rich-elem", "basic")
				return CreatePurposeRequest{
					Name:        "cp-rich",
					DisplayName: ptr("Rich Purpose"),
					Description: ptr("A fully populated purpose"),
					Properties:  map[string]string{"env": "test", "tier": "gold"},
					Elements:    []ElementRefRequest{{Name: "cp-rich-elem"}},
				}
			},
			wantStatus: http.StatusCreated,
			checkResult: func(_ string, resp *PurposeResponse) {
				ts.assertPurposeResponse(resp, "cp-rich")
				ts.Require().NotNil(resp.DisplayName)
				ts.Equal("Rich Purpose", *resp.DisplayName)
				ts.Require().NotNil(resp.Description)
				ts.Equal("A fully populated purpose", *resp.Description)
				ts.Equal("test", resp.Properties["env"])
				ts.Equal("gold", resp.Properties["tier"])
				ts.Require().Len(resp.Elements, 1)
				ts.assertPurposeElement(resp.Elements[0], "cp-rich-elem", "default", "v1", false)
			},
		},

		// -----------------------------------------------------------------------
		// Duplicate / conflict
		// -----------------------------------------------------------------------
		{
			name: "same name + same groupId → 409 CP-4041",
			setup: func(orgID string) {
				ts.mustCreatePurpose(orgID, "cp-dup")
			},
			// Name check fires before element validation — any valid element name is fine.
			rawBody:       `{"name":"cp-dup","elements":[{"name":"any"}]}`,
			wantStatus:    http.StatusConflict,
			wantErrorCode: "CP-4041",
		},
		{
			name: "org-level duplicate (group-id omitted) → description says 'already exists in this org'",
			setup: func(orgID string) {
				ts.mustCreatePurpose(orgID, "cp-dup-org-msg")
			},
			rawBody:       `{"name":"cp-dup-org-msg","elements":[{"name":"any"}]}`,
			wantStatus:    http.StatusConflict,
			wantErrorCode: "CP-4041",
			wantErrorDesc: "already exists in this org",
		},
		{
			name: "group-scoped duplicate (explicit group-id) → description says 'already exists in this group'",
			setup: func(orgID string) {
				ts.mustCreatePurposeWith(orgID, "grp-dup", CreatePurposeRequest{Name: "cp-dup-grp-msg"})
			},
			groupID:       "grp-dup",
			rawBody:       `{"name":"cp-dup-grp-msg","elements":[{"name":"any"}]}`,
			wantStatus:    http.StatusConflict,
			wantErrorCode: "CP-4041",
			wantErrorDesc: "already exists in this group",
		},
		{
			name: "same name + different groupId → 201 allowed",
			setup: func(orgID string) {
				ts.mustCreatePurposeWith(orgID, "grp-a", CreatePurposeRequest{Name: "cp-shared-name"})
			},
			groupID: "grp-b",
			buildBody: func(orgID string) any {
				ts.mustCreateElement(orgID, "cp-shared-elem2", "basic")
				return CreatePurposeRequest{
					Name:     "cp-shared-name",
					Elements: []ElementRefRequest{{Name: "cp-shared-elem2"}},
				}
			},
			wantStatus: http.StatusCreated,
			checkResult: func(_ string, resp *PurposeResponse) {
				ts.assertPurposeResponse(resp, "cp-shared-name")
				ts.Equal("grp-b", resp.GroupID)
				ts.Require().Len(resp.Elements, 1)
			},
		},

		// -----------------------------------------------------------------------
		// Element validation errors
		// -----------------------------------------------------------------------
		{
			name:          "no elements field at all → 400 CP-4003 (at least one required)",
			rawBody:       `{"name":"cp-no-elems"}`,
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "CP-4003",
		},
		{
			name:          "empty elements array → 400 CP-4003 (at least one required)",
			rawBody:       `{"name":"cp-empty-elems","elements":[]}`,
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "CP-4003",
		},
		{
			name:          "element name that does not exist → 400 CP-4003",
			rawBody:       `{"name":"cp-missing-elem","elements":[{"name":"no-such-element-xyz"}]}`,
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "CP-4003",
		},
		{
			// Element was created in "finance" namespace.
			// A purpose reference without an explicit namespace defaults to "default".
			// Since no element named "cp-ns-elem" exists in the "default" namespace,
			// the server must return CP-4003.
			name: "element exists in non-default namespace — omitting namespace defaults to 'default', not found → 400 CP-4003",
			buildBody: func(orgID string) any {
				ts.mustCreateElementWith(orgID, map[string]any{
					"name":      "cp-ns-elem",
					"type":      "basic",
					"namespace": "finance",
				})
				return CreatePurposeRequest{
					Name: "cp-ns-default-fail",
					Elements: []ElementRefRequest{
						{Name: "cp-ns-elem"}, // no Namespace → server defaults to "default"
					},
				}
			},
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "CP-4003",
		},
		{
			// Same element in "finance" namespace, but this time the reference includes
			// namespace: "finance" explicitly — the server should accept it.
			name: "element exists in non-default namespace — reference includes correct namespace → 201",
			buildBody: func(orgID string) any {
				ts.mustCreateElementWith(orgID, map[string]any{
					"name":      "cp-ns-explicit-elem",
					"type":      "basic",
					"namespace": "finance",
				})
				return CreatePurposeRequest{
					Name: "cp-ns-explicit",
					Elements: []ElementRefRequest{
						{Name: "cp-ns-explicit-elem", Namespace: "finance"},
					},
				}
			},
			wantStatus: http.StatusCreated,
			checkResult: func(_ string, resp *PurposeResponse) {
				ts.assertPurposeResponse(resp, "cp-ns-explicit")
				ts.Require().Len(resp.Elements, 1)
				ts.assertPurposeElement(resp.Elements[0], "cp-ns-explicit-elem", "finance", "v1", false)
			},
		},
		{
			name:          "element version in invalid format → 400 CP-4001",
			rawBody:       `{"name":"cp-bad-ver","elements":[{"name":"e","version":"abc"}]}`,
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "CP-4001",
		},
		{
			name:          "element version v0 (must be ≥ v1) → 400 CP-4001",
			rawBody:       `{"name":"cp-ver-zero","elements":[{"name":"e","version":"v0"}]}`,
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "CP-4001",
		},

		// -----------------------------------------------------------------------
		// Header validation errors
		// -----------------------------------------------------------------------
		{
			name:          "missing org-id header → 400 CP-4004",
			omitOrgID:     true,
			rawBody:       `{"name":"cp-no-org","elements":[{"name":"e"}]}`,
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "CP-4004",
		},
		{
			name:          "malformed JSON body → 400 CP-4001",
			rawBody:       `{bad json`,
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "CP-4001",
		},
	}

	for _, tc := range cases {
		tc := tc
		ts.Run(tc.name, func() {
			orgID := freshOrgID()

			if tc.setup != nil {
				tc.setup(orgID)
			}

			requestOrgID := orgID
			if tc.omitOrgID {
				requestOrgID = ""
			}

			// Dynamic builder takes priority; rawBody is the fallback for static bodies.
			var reqBody any
			if tc.buildBody != nil {
				reqBody = tc.buildBody(orgID)
			} else {
				reqBody = tc.rawBody
			}

			status, body := ts.doCreatePurposeFull(requestOrgID, tc.groupID, reqBody)
			ts.Require().Equal(tc.wantStatus, status)

			if tc.wantErrorCode != "" {
				errResp := ts.assertAPIError(body, tc.wantErrorCode)
				if tc.wantErrorDesc != "" {
					ts.Contains(errResp.Description, tc.wantErrorDesc, "error description mismatch")
				}
				return
			}

			var resp PurposeResponse
			ts.Require().NoError(json.Unmarshal(body, &resp), "unmarshal PurposeResponse: %s", body)
			if tc.checkResult != nil {
				tc.checkResult(orgID, &resp)
			}
		})
	}
}
