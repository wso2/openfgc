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

import {
  Box,
  Button,
  Card,
  CardContent,
  CardHeader,
  Divider,
  Skeleton,
  Stack,
  Typography,
} from '@wso2/oxygen-ui'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useParams } from 'react-router-dom'
import HeaderBreadcrumbs from '../../components/layout/main-layout/HeaderBreadcrumbs'
import ConsentApprovalDialog from './components/ConsentApprovalDialog'
import ConsentRevocationDialog from './components/ConsentRevocationDialog'
import ConsentAuthorizationsSection from './components/details/ConsentAuthorizationsSection'
import ConsentMetadataCard from './components/details/ConsentMetadataCard'
import ConsentPurposesSection from './components/details/ConsentPurposesSection'
import ConsentResourcesModal from './components/details/ConsentResourcesModal'
import {
  useApproveConsentMutation,
  useConsentDetailQuery,
  useRevokeConsentMutation,
} from './hooks/useConsentQueries'
import { isConsentApprovableStatus, isConsentRevokableStatus } from './utils/statusChip'

function formatResourcesForModal(resources: unknown): string {
  if (!resources) {
    return '-'
  }

  if (typeof resources === 'string') {
    try {
      return JSON.stringify(JSON.parse(resources), null, 2)
    } catch {
      return resources
    }
  }

  try {
    return JSON.stringify(resources, null, 2)
  } catch {
    return String(resources)
  }
}

function ConsentDetailsLoading(): React.JSX.Element {
  return (
    <Box
      component="main"
      sx={{ p: { xs: 2, md: 4 }, display: 'flex', flexDirection: 'column', gap: 3 }}
    >
      <Stack spacing={1}>
        <HeaderBreadcrumbs />
        <Skeleton variant="text" width={220} height={48} />
      </Stack>

      <Card sx={{ boxShadow: 1 }}>
        <CardHeader
          title={<Skeleton variant="text" width={280} />}
          action={
            <Stack direction="row" spacing={1}>
              <Skeleton variant="rounded" width={84} height={24} />
              <Skeleton variant="rounded" width={84} height={24} />
            </Stack>
          }
          sx={{ pb: 2 }}
        />
        <Divider />
        <CardContent sx={{ pt: 3 }}>
          <Box
            sx={{
              display: 'grid',
              gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr', lg: 'repeat(3, 1fr)' },
              gap: { xs: 3, md: 4 },
            }}
          >
            {Array.from({ length: 6 }).map((_, index) => (
              <Box key={`detail-skeleton-${String(index)}`}>
                <Skeleton variant="text" width="45%" />
                <Skeleton variant="text" width="85%" />
              </Box>
            ))}
          </Box>
        </CardContent>
      </Card>

      {['purpose', 'table', 'lifecycle'].map((section) => (
        <Card key={`details-${section}-skeleton`} sx={{ boxShadow: 1 }}>
          <CardHeader title={<Skeleton variant="text" width={180} />} sx={{ pb: 0 }} />
          <Divider />
          <CardContent sx={{ p: 2 }}>
            {Array.from({ length: section === 'purpose' ? 2 : 4 }).map((_, index) => (
              <Box
                key={`${section}-row-skeleton-${String(index)}`}
                sx={{
                  display: section === 'purpose' ? 'block' : 'grid',
                  gridTemplateColumns:
                    section === 'table' ? '1.2fr 1fr 1fr 0.8fr' : '1fr 1fr 1fr 2fr',
                  gap: 2,
                  mb: index === (section === 'purpose' ? 1 : 3) ? 0 : 1,
                }}
              >
                {section === 'purpose' ? (
                  <Skeleton variant="rounded" height={42} />
                ) : (
                  <>
                    <Skeleton variant="text" width="70%" />
                    <Skeleton variant="text" width="80%" />
                    <Skeleton variant="text" width="80%" />
                    <Skeleton variant={section === 'table' ? 'rounded' : 'text'} width="95%" />
                  </>
                )}
              </Box>
            ))}
          </CardContent>
        </Card>
      ))}
    </Box>
  )
}

function ConsentDetailsPage(): React.JSX.Element {
  const { t } = useTranslation('common')
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const consentDetailQuery = useConsentDetailQuery(id)
  const approveMutation = useApproveConsentMutation()
  const revokeMutation = useRevokeConsentMutation()
  const [approvalDialogOpen, setApprovalDialogOpen] = useState<boolean>(false)
  const [revocationDialogOpen, setRevocationDialogOpen] = useState<boolean>(false)
  const [resourcesModalOpen, setResourcesModalOpen] = useState<boolean>(false)
  const [selectedResourcesJson, setSelectedResourcesJson] = useState<string>('')

  if (!id) {
    return (
      <Box
        component="main"
        sx={{ p: { xs: 2, md: 4 }, display: 'flex', flexDirection: 'column', gap: 2 }}
      >
        <Typography variant="h5">{t('consentRegistry.details.notFound')}</Typography>
        <Box>
          <Button variant="outlined" onClick={() => navigate('/consents')}>
            {t('consentRegistry.details.back')}
          </Button>
        </Box>
      </Box>
    )
  }

  const detail = consentDetailQuery.data
  const canApprove = detail ? isConsentApprovableStatus(detail.status) : false
  const canRevoke = detail ? isConsentRevokableStatus(detail.status) : false

  if (consentDetailQuery.isLoading) {
    return <ConsentDetailsLoading />
  }

  if (consentDetailQuery.isError || !detail) {
    return (
      <Box
        component="main"
        sx={{ p: { xs: 2, md: 4 }, display: 'flex', flexDirection: 'column', gap: 2 }}
      >
        <Typography color="error.main">{t('consentRegistry.messages.loadFailed')}</Typography>
        <Box>
          <Button variant="outlined" onClick={() => navigate('/consents')}>
            {t('consentRegistry.details.back')}
          </Button>
        </Box>
      </Box>
    )
  }

  return (
    <Box
      component="main"
      sx={{ p: { xs: 2, md: 4 }, display: 'flex', flexDirection: 'column', gap: 3 }}
    >
      <Box sx={{ position: 'relative' }}>
        <Stack spacing={1}>
          <HeaderBreadcrumbs />
          <Typography variant="h4" sx={{ fontWeight: 700 }}>
            {t('consentRegistry.details.title', 'Consent Details')}
          </Typography>
        </Stack>
        <Stack
          direction="row"
          spacing={1.5}
          sx={{
            mt: { xs: 2, md: 0 },
            position: { xs: 'static', md: 'absolute' },
            right: { md: 0 },
            bottom: { md: 0 },
          }}
        >
          {canApprove ? (
            <Button
              variant="contained"
              color="warning"
              size="small"
              disabled={approveMutation.isPending}
              onClick={() => {
                setApprovalDialogOpen(true)
              }}
            >
              {t('consentRegistry.actions.approve')}
            </Button>
          ) : null}
          <Button
            variant="contained"
            color="error"
            size="small"
            disabled={revokeMutation.isPending || !canRevoke}
            onClick={() => {
              setRevocationDialogOpen(true)
            }}
          >
            {t('consentRegistry.actions.revoke')}
          </Button>
        </Stack>
      </Box>

      <ConsentMetadataCard consentId={id} detail={detail} />
      <ConsentPurposesSection purposes={detail.purposes} />
      <ConsentAuthorizationsSection
        authorizations={detail.authorizations ?? []}
        onViewResources={(resources) => {
          setSelectedResourcesJson(formatResourcesForModal(resources))
          setResourcesModalOpen(true)
        }}
      />

      <ConsentResourcesModal
        open={resourcesModalOpen}
        resourcesJson={selectedResourcesJson}
        onClose={() => {
          setResourcesModalOpen(false)
        }}
      />

      <ConsentApprovalDialog
        key={`approval-${id}-${String(approvalDialogOpen)}`}
        open={approvalDialogOpen}
        consentId={id}
        purposes={detail.purposes}
        loading={approveMutation.isPending}
        onClose={() => {
          setApprovalDialogOpen(false)
        }}
        onConfirm={(selectedOptionalElements) => {
          approveMutation.mutate(
            { consentID: id, selectedOptionalElements },
            {
              onSuccess: () => {
                setApprovalDialogOpen(false)
              },
            },
          )
        }}
      />

      <ConsentRevocationDialog
        key={`revocation-${id}-${String(revocationDialogOpen)}`}
        open={revocationDialogOpen}
        consentId={id}
        loading={revokeMutation.isPending}
        onClose={() => {
          setRevocationDialogOpen(false)
        }}
        onConfirm={() => {
          revokeMutation.mutate(id, {
            onSuccess: () => {
              setRevocationDialogOpen(false)
            },
          })
        }}
      />
    </Box>
  )
}

export default ConsentDetailsPage
