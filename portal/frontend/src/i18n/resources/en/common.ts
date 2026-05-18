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

const commonEn = {
  app: {
    title: 'Consent Portal',
  },
  sidebar: {
    ariaLabel: 'Primary navigation',
    dashboard: 'Dashboard',
    consent: 'Consent',
    allConsents: 'All Consents',
    pendingConsents: 'Pending Consents',
  },
  layout: {
    home: 'Home',
    breadcrumbAriaLabel: 'Breadcrumb',
    userAvatarAriaLabel: 'Signed-in user avatar',
  },
  dashboard: {
    title: 'Dashboard',
  },
  consentRegistry: {
    title: 'All Consents',
    details: {
      title: 'Consent Details',
      clientName: 'Client Name',
      consentId: 'Consent ID',
      status: 'Status',
      type: 'Consent Type',
      frequency: 'Access Limit',
      frequencyHelp: 'This indicates how many times this consent can be accessed per day.',
      frequencyUnitSingular: 'time per day',
      frequencyUnitPlural: 'times per day',
      purposes: 'Purposes',
      duration: 'Lookback Period',
      durationHelp:
        'This defines how far back data can be accessed. For example, if set to 6 months, data from up to 6 months ago is accessible.',
      durationUnitHourSingular: 'hour',
      durationUnitHourPlural: 'hours',
      durationUnitDaySingular: 'day',
      durationUnitDayPlural: 'days',
      durationUnitYearSingular: 'year',
      durationUnitYearPlural: 'years',
      created: 'Created',
      updated: 'Updated',
      validUntil: 'Valid Until',
      clientId: 'Client ID',
      recurring: 'Recurring',
      back: 'Back to Registry',
      notFound: 'Consent record not found',
      approved: 'Approved',
      notApproved: 'Not approved',
      approvedCount: '{{approved}}/{{total}} approved',
      section: {
        purposes: 'Consent Purposes',
        authorizations: 'Authorizations',
        lifecycle: 'Consent Lifecycle',
      },
      table: {
        element: 'Element',
        approved: 'Approved',
        required: 'Required',
        description: 'Description',
        user: 'User',
        status: 'Status',
        updated: 'Updated',
        resources: 'Resources',
        eventType: 'Event Type',
        date: 'Date',
        time: 'Time',
      },
      actions: {
        viewResources: 'View Resources',
        noResourcesTooltip: 'No resources available',
      },
      resourcesModal: {
        title: 'Authorization Resources',
        authRef: 'Auth',
        close: 'Close',
      },
      values: {
        yes: 'Yes',
        no: 'No',
        required: 'Required',
        optional: 'Optional',
      },
    },
    actions: {
      view: 'View',
      revoke: 'Revoke',
      approve: 'Approve',
    },
    modals: {
      consentId: 'Consent ID',
      actions: {
        cancel: 'Cancel',
        processing: 'Processing...',
      },
      approval: {
        title: 'Review & Approve Consent',
        subtitle: 'Please review the consent elements before approval.',
        mandatory: 'Mandatory Elements (Required)',
        optional: 'Optional Elements',
        required: 'Required',
        toggle: 'Toggle permission',
        toggleWithDetails: 'Toggle permission for {{elementName}} in {{purposeName}}',
        noMandatory: 'No mandatory requirements for this consent.',
        confirm: 'Approve & Continue',
      },
      revocation: {
        title: 'Confirm Revocation',
        message: 'Are you sure you want to revoke consent?',
        note: 'This action revokes both mandatory and optional consents granted for all associated purposes.',
        confirm: 'Revoke Consents',
        cancel: 'Cancel',
      },
    },
    status: {
      all: 'All',
      active: 'Active',
      pending: 'Pending',
      created: 'Created',
      approved: 'Approved',
      rejected: 'Rejected',
      revoked: 'Revoked',
      expired: 'Expired',
      systemExpired: 'System Expired',
      systemRevoked: 'System Revoked',
    },
    filters: {
      sectionAriaLabel: 'Consent filters',
      status: 'Status',
      startDate: 'Start date',
      endDate: 'End date',
      consentType: 'Consent type',
      clear: 'Clear',
    },
    messages: {
      loading: 'Loading consents...',
      loadFailed: 'Unable to load consents right now.',
      empty: 'No consents found for the selected filters.',
    },
    table: {
      tableAriaLabel: 'Consent registry table',
      clientLabel: 'Client: {{client}}',
      notApplicable: 'Not applicable',
      purposes: {
        more: '+{{count}} more',
        title: 'Consent purposes',
        hint: 'Showing all purposes of the consent',
      },
      headers: {
        consentId: 'Consent ID',
        type: 'Type',
        status: 'Status',
        purposes: 'Purposes',
        updated: 'Updated',
        expiration: 'Expiration',
        actions: 'Actions',
      },
    },
  },
} as const

export default commonEn
