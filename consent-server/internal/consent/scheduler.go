/*
 * Copyright (c) 2026, WSO2 LLC. (https://www.wso2.com).
 *
 * WSO2 LLC. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package consent

import (
	"context"
	"time"

	"github.com/wso2/openfgc/internal/system/log"
)

// ExpirationStatuses groups all status strings needed by the expiration job.
type ExpirationStatuses struct {
	ExpirableConsentStatuses []string
}

// StartScheduler starts the consent expiration scheduler at the given interval.
// It runs until the context is cancelled.
func StartScheduler(ctx context.Context, expirationService ExpirationService, interval time.Duration, statuses ExpirationStatuses) {
	logger := log.GetLogger().With(log.String(log.LoggerKeyComponentName, "ConsentScheduler"))
	if interval <= 0 {
		logger.Error("Invalid scheduler interval — must be greater than zero", log.String("interval", interval.String()))
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.Debug("Scheduler stopped due to context cancellation")
			return
		case <-ticker.C:
			logger.Debug("Scheduler tick — running expiration job")
			// RunExpirationJob is called synchronously (not in a goroutine) to ensure only one
			// expiration job runs at a time, preventing concurrent writes on the same consents.
			RunExpirationJob(ctx, expirationService, statuses)
			logger.Debug("Expiration job completed")
		}
	}
}
