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
	"context"
	"encoding/json"
	"fmt"

	"github.com/wso2/openfgc/internal/consent/model"
	"github.com/wso2/openfgc/internal/system/config"
	dbmodel "github.com/wso2/openfgc/internal/system/database/model"
	"github.com/wso2/openfgc/internal/system/stores"
	"github.com/wso2/openfgc/internal/system/utils"
)

// HistoryReason is the server-generated reason attached to consent amendment history.
type HistoryReason string

const (
	HistoryReasonConsentAmended                        HistoryReason = "Consent amended"
	HistoryReasonConsentDetailsAmended                 HistoryReason = "Consent details amended"
	HistoryReasonConsentAttributesAmended              HistoryReason = "Consent attributes amended"
	HistoryReasonConsentAuthorizationsAmended          HistoryReason = "Consent authorizations amended"
	HistoryReasonConsentPurposesAmended                HistoryReason = "Consent purposes amended"
	HistoryReasonConsentRevoked                        HistoryReason = "Consent revoked"
	HistoryReasonConsentExpired                        HistoryReason = "Consent expired"
	HistoryReasonConsentDetailsAmendedAndReactivated   HistoryReason = "Consent details amended and reactivated"
	HistoryReasonConsentAuthorizationsAmendedAndStatus HistoryReason = "Consent authorizations amended and status updated"
)

// RecordConsentHistory records a pre-mutation consent snapshot using a shared store registry.
func RecordConsentHistory(
	ctx context.Context,
	registry *stores.StoreRegistry,
	tx dbmodel.TxInterface,
	consentID, orgID string,
	actionBy *string,
	reason HistoryReason,
) error {
	return (&consentService{stores: registry}).recordConsentHistory(
		ctx,
		tx,
		consentID,
		orgID,
		actionBy,
		reason,
	)
}

func (consentService *consentService) recordConsentHistory(
	ctx context.Context,
	tx dbmodel.TxInterface,
	consentID, orgID string,
	actionBy *string,
	reason HistoryReason,
) error {
	cfg := config.Get()
	if cfg == nil || !cfg.Consent.History.Enabled {
		return nil
	}

	consent, err := consentService.stores.Consent.GetByIDForUpdate(tx, consentID, orgID)
	if err != nil {
		return fmt.Errorf("failed to lock consent for history: %w", err)
	}
	if consent == nil {
		return fmt.Errorf("consent with ID '%s' not found", consentID)
	}

	snapshot, err := consentService.buildConsentHistorySnapshot(ctx, consent, orgID)
	if err != nil {
		return err
	}

	reasonText := string(reason)
	history := &model.ConsentHistory{
		HistoryID:  utils.GenerateUUID(),
		ConsentID:  consentID,
		OrgID:      orgID,
		ActionTime: utils.GetCurrentTimeMillis(),
		ActionBy:   actionBy,
		Reason:     &reasonText,
		Snapshot:   snapshot,
	}
	return consentService.stores.Consent.CreateHistory(tx, history)
}

func (consentService *consentService) buildConsentHistorySnapshot(
	ctx context.Context,
	consent *model.Consent,
	orgID string,
) ([]byte, error) {
	output, err := consentService.loadConsentOutput(ctx, consent, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to load consent output for history: %w", err)
	}

	snapshot, err := json.Marshal(consentOutputToResponse(output))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal consent history snapshot: %w", err)
	}
	return snapshot, nil
}
