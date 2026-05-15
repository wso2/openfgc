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
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Box,
  Button,
  Card,
  CardContent,
  CardHeader,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  Stack,
  Skeleton,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Tooltip,
  Typography,
  Avatar,
} from '@wso2/oxygen-ui'
import { ChevronRight, CheckCircle, CircleHelp, Eye, XCircle } from '@wso2/oxygen-ui-icons-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useParams, useNavigate } from 'react-router-dom'
import HeaderBreadcrumbs from '../../components/layout/main-layout/HeaderBreadcrumbs'
import { formatEpochTimestamp, formatIsoDateTime } from '../../utils/dateTime'
import ConsentApprovalDialog from './components/ConsentApprovalDialog'
import ConsentRevocationDialog from './components/ConsentRevocationDialog'
import {
  getConsentStatusChipColor,
  getConsentStatusLabelKey,
  isConsentApprovableStatus,
  isConsentRevokableStatus,
} from './utils/statusChip'
import {
  useApproveConsentMutation,
  useConsentDetailQuery,
  useRevokeConsentMutation,
} from './hooks/useConsentQueries'

const LIFECYCLE_DATE_FORMAT_OPTIONS: Intl.DateTimeFormatOptions = {
  day: '2-digit',
  month: '2-digit',
  year: 'numeric',
}

const LIFECYCLE_TIME_FORMAT_OPTIONS: Intl.DateTimeFormatOptions = {
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
}

const PURPOSE_ELEMENTS_COLUMN_WIDTHS = {
  element: '28%',
  approved: '14%',
  required: '18%',
  description: '40%',
} as const

const ELEMENT_NAME_MAX_DISPLAY_LENGTH = 28

type DurationUnit = 'hour' | 'day' | 'year'

interface DurationDisplayParts {
  value: number
  unit: DurationUnit
}

function getDurationDisplayParts(
  durationInSeconds: number | undefined,
): DurationDisplayParts | null {
  if (!durationInSeconds || durationInSeconds <= 0) {
    return null
  }

  const totalHours = durationInSeconds / 3600

  if (totalHours < 24) {
    return { value: Math.max(1, Math.floor(totalHours)), unit: 'hour' }
  }

  const totalDays = totalHours / 24

  if (totalDays < 365) {
    return { value: Math.floor(totalDays), unit: 'day' }
  }

  return { value: Math.floor(totalDays / 365), unit: 'year' }
}

function getDurationUnitLabelKey(unit: DurationUnit, isSingular: boolean): string {
  if (unit === 'hour') {
    return isSingular
      ? 'consentRegistry.details.durationUnitHourSingular'
      : 'consentRegistry.details.durationUnitHourPlural'
  }

  if (unit === 'day') {
    return isSingular
      ? 'consentRegistry.details.durationUnitDaySingular'
      : 'consentRegistry.details.durationUnitDayPlural'
  }

  return isSingular
    ? 'consentRegistry.details.durationUnitYearSingular'
    : 'consentRegistry.details.durationUnitYearPlural'
}

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

function truncateElementName(elementName: string, maxLength: number): string {
  if (elementName.length <= maxLength) {
    return elementName
  }

  return `${elementName.slice(0, Math.max(maxLength - 1, 1))}...`
}

function isEmptyAuthorizationResources(resources: unknown): boolean {
  if (resources == null) {
    return true
  }

  if (typeof resources === 'string') {
    return resources.trim().length === 0
  }

  if (Array.isArray(resources)) {
    return resources.length === 0
  }

  if (typeof resources === 'object') {
    return Object.keys(resources as Record<string, unknown>).length === 0
  }

  return false
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
  const hasFrequency = detail ? detail.frequency != null && detail.frequency !== 0 : false
  const hasDuration = detail
    ? detail.dataAccessValidityDuration != null && detail.dataAccessValidityDuration !== 0
    : false
  const hasValidityTime = detail ? detail.validityTime != null && detail.validityTime !== 0 : false
  const durationDisplay = detail ? getDurationDisplayParts(detail.dataAccessValidityDuration) : null
  const durationUnitLabel = durationDisplay
    ? t(getDurationUnitLabelKey(durationDisplay.unit, durationDisplay.value === 1))
    : ''

  if (consentDetailQuery.isLoading) {
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

        <Card sx={{ boxShadow: 1 }}>
          <CardHeader title={<Skeleton variant="text" width={180} />} sx={{ pb: 0 }} />
          <Divider />
          <CardContent sx={{ p: 2 }}>
            {Array.from({ length: 2 }).map((_, index) => (
              <Box key={`purpose-skeleton-${String(index)}`} sx={{ mb: index === 1 ? 0 : 1 }}>
                <Skeleton variant="rounded" height={42} />
              </Box>
            ))}
          </CardContent>
        </Card>

        <Card sx={{ boxShadow: 1 }}>
          <CardHeader title={<Skeleton variant="text" width={180} />} sx={{ pb: 0 }} />
          <Divider />
          <CardContent sx={{ p: 2 }}>
            {Array.from({ length: 4 }).map((_, index) => (
              <Box
                key={`table-row-skeleton-${String(index)}`}
                sx={{ display: 'grid', gridTemplateColumns: '1.2fr 1fr 1fr 0.8fr', gap: 2, mb: 1 }}
              >
                <Skeleton variant="text" width="85%" />
                <Skeleton variant="rounded" width={80} height={24} />
                <Skeleton variant="text" width="75%" />
                <Skeleton variant="rounded" width={110} height={28} />
              </Box>
            ))}
          </CardContent>
        </Card>

        <Card sx={{ boxShadow: 1 }}>
          <CardHeader title={<Skeleton variant="text" width={180} />} sx={{ pb: 0 }} />
          <Divider />
          <CardContent sx={{ p: 2 }}>
            {Array.from({ length: 4 }).map((_, index) => (
              <Box
                key={`lifecycle-row-skeleton-${String(index)}`}
                sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr 2fr', gap: 2, mb: 1 }}
              >
                <Skeleton variant="text" width="70%" />
                <Skeleton variant="text" width="80%" />
                <Skeleton variant="text" width="80%" />
                <Skeleton variant="text" width="95%" />
              </Box>
            ))}
          </CardContent>
        </Card>
      </Box>
    )
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

      {/* Consent Details Section */}
      <Card sx={{ boxShadow: 1 }}>
        <CardHeader
          title={
            <Stack direction="row" spacing={1} alignItems="center">
              <Typography variant="body2" fontWeight={400}>
                {t('consentRegistry.details.consentId', 'Consent ID')}: {id}
              </Typography>
            </Stack>
          }
          action={
            <Stack direction="row" spacing={1}>
              <Chip
                label={t(`consentRegistry.status.${getConsentStatusLabelKey(detail.status)}`)}
                color={getConsentStatusChipColor(detail.status)}
                size="small"
                variant="outlined"
              />
              <Chip label={detail.type} color="default" size="small" variant="outlined" />
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
            <Box>
              <Typography
                variant="caption"
                color="text.secondary"
                fontWeight={700}
                sx={{ display: 'block', mb: 1, textTransform: 'uppercase', letterSpacing: 0.5 }}
              >
                {t('consentRegistry.details.created', 'Created')}
              </Typography>
              <Typography variant="body2" fontWeight={500}>
                {formatEpochTimestamp(detail.createdTime)}
              </Typography>
            </Box>
            <Box>
              <Typography
                variant="caption"
                color="text.secondary"
                fontWeight={700}
                sx={{ display: 'block', mb: 1, textTransform: 'uppercase', letterSpacing: 0.5 }}
              >
                {t('consentRegistry.details.updated', 'Updated')}
              </Typography>
              <Typography variant="body2" fontWeight={500}>
                {formatEpochTimestamp(detail.updatedTime)}
              </Typography>
            </Box>
            <Box>
              <Typography
                variant="caption"
                color="text.secondary"
                fontWeight={700}
                sx={{ display: 'block', mb: 1, textTransform: 'uppercase', letterSpacing: 0.5 }}
              >
                {t('consentRegistry.details.validUntil', 'Valid Until')}
              </Typography>
              <Typography variant="body2" fontWeight={500}>
                <Box
                  component="span"
                  sx={{ color: hasValidityTime ? 'text.primary' : 'text.disabled' }}
                >
                  {hasValidityTime
                    ? formatEpochTimestamp(detail.validityTime)
                    : t('consentRegistry.table.notApplicable', 'Not applicable')}
                </Box>
              </Typography>
            </Box>
            <Box>
              <Typography
                variant="caption"
                color="text.secondary"
                fontWeight={700}
                sx={{ display: 'block', mb: 1, textTransform: 'uppercase', letterSpacing: 0.5 }}
              >
                {t('consentRegistry.details.clientId', 'Client ID')}
              </Typography>
              <Typography variant="body2" fontWeight={500}>
                {detail.clientId}
              </Typography>
            </Box>
            <Box>
              <Typography
                variant="caption"
                color="text.secondary"
                fontWeight={700}
                sx={{ display: 'block', mb: 1, textTransform: 'uppercase', letterSpacing: 0.5 }}
              >
                {t('consentRegistry.details.recurring', 'Recurring')}
              </Typography>
              <Typography variant="body2" fontWeight={500}>
                {detail.recurringIndicator
                  ? t('consentRegistry.details.values.yes', 'Yes')
                  : t('consentRegistry.details.values.no', 'No')}
              </Typography>
            </Box>
            {hasFrequency ? (
              <Box>
                <Typography
                  variant="caption"
                  color="text.secondary"
                  fontWeight={700}
                  sx={{ display: 'block', mb: 1, textTransform: 'uppercase', letterSpacing: 0.5 }}
                >
                  <Box
                    component="span"
                    sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5 }}
                  >
                    {t('consentRegistry.details.frequency', 'Access Limit')}
                    <Tooltip
                      title={t(
                        'consentRegistry.details.frequencyHelp',
                        'This indicates how many times this consent can be accessed per day.',
                      )}
                    >
                      <Box
                        component="span"
                        sx={{
                          display: 'inline-flex',
                          alignItems: 'center',
                          color: 'text.disabled',
                        }}
                      >
                        <CircleHelp size={14} />
                      </Box>
                    </Tooltip>
                  </Box>
                </Typography>
                <Typography variant="body2" fontWeight={500}>
                  {detail.frequency}{' '}
                  {detail.frequency === 1
                    ? t('consentRegistry.details.frequencyUnitSingular', 'time per day')
                    : t('consentRegistry.details.frequencyUnitPlural', 'times per day')}
                </Typography>
              </Box>
            ) : null}
            {hasDuration ? (
              <Box>
                <Typography
                  variant="caption"
                  color="text.secondary"
                  fontWeight={700}
                  sx={{ display: 'block', mb: 1, textTransform: 'uppercase', letterSpacing: 0.5 }}
                >
                  <Box
                    component="span"
                    sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5 }}
                  >
                    {t('consentRegistry.details.duration', 'Lookback Period')}
                    <Tooltip
                      title={t(
                        'consentRegistry.details.durationHelp',
                        'This defines how far back data can be accessed. For example, if set to 6 months, data from up to 6 months ago is accessible.',
                      )}
                    >
                      <Box
                        component="span"
                        sx={{
                          display: 'inline-flex',
                          alignItems: 'center',
                          color: 'text.disabled',
                        }}
                      >
                        <CircleHelp size={14} />
                      </Box>
                    </Tooltip>
                  </Box>
                </Typography>
                <Typography variant="body2" fontWeight={500}>
                  {durationDisplay ? `${durationDisplay.value} ${durationUnitLabel}` : '-'}
                </Typography>
              </Box>
            ) : null}
          </Box>
        </CardContent>
      </Card>

      {/* Consent Purposes Section */}
      <Card sx={{ boxShadow: 1 }}>
        <CardHeader
          title={
            <Typography variant="subtitle1" fontWeight={600}>
              {t('consentRegistry.details.section.purposes', 'Consent Purposes')}
            </Typography>
          }
          sx={{ pb: 0 }}
        />
        <Divider />
        <CardContent sx={{ p: 2 }}>
          {detail.purposes.map((purpose) => {
            const approved = purpose.elements.filter((element) => element.isUserApproved).length
            const total = purpose.elements.length

            return (
              <Accordion
                key={purpose.name}
                disableGutters
                elevation={0}
                sx={{
                  mb: 1,
                  border: 1,
                  borderColor: 'divider',
                  borderRadius: 1,
                  overflow: 'hidden',
                  '&:before': { display: 'none' },
                  '&.Mui-expanded': { mt: 0, mb: 1 },
                  '&:last-of-type': { mb: 0 },
                  '&.Mui-expanded:last-of-type': { mb: 0 },
                }}
              >
                <AccordionSummary
                  expandIcon={<ChevronRight />}
                  sx={{ '&:hover': { bgcolor: 'action.hover' } }}
                >
                  <Stack direction="row" spacing={1.5} alignItems="center">
                    <Chip
                      label={`${approved}/${total} approved`}
                      color="primary"
                      size="small"
                      sx={{
                        height: 20,
                        '& .MuiChip-label': { px: 0.75, fontSize: '0.6875rem', fontWeight: 500 },
                      }}
                    />
                    <Typography variant="body2" fontWeight={600}>
                      {purpose.name}
                    </Typography>
                  </Stack>
                </AccordionSummary>
                <AccordionDetails sx={{ p: 0 }}>
                  <TableContainer>
                    <Table
                      size="small"
                      sx={{
                        tableLayout: 'fixed',
                        '& tbody tr:hover': { bgcolor: 'action.hover' },
                      }}
                    >
                      <TableHead>
                        <TableRow>
                          <TableCell
                            sx={{
                              fontWeight: 700,
                              width: PURPOSE_ELEMENTS_COLUMN_WIDTHS.element,
                            }}
                          >
                            {t('consentRegistry.details.table.element', 'Element')}
                          </TableCell>
                          <TableCell
                            sx={{
                              fontWeight: 700,
                              width: PURPOSE_ELEMENTS_COLUMN_WIDTHS.approved,
                            }}
                          >
                            {t('consentRegistry.details.table.approved', 'Approved')}
                          </TableCell>
                          <TableCell
                            sx={{
                              fontWeight: 700,
                              width: PURPOSE_ELEMENTS_COLUMN_WIDTHS.required,
                            }}
                          >
                            {t('consentRegistry.details.table.required', 'Required')}
                          </TableCell>
                          <TableCell
                            sx={{
                              fontWeight: 700,
                              width: PURPOSE_ELEMENTS_COLUMN_WIDTHS.description,
                            }}
                          >
                            {t('consentRegistry.details.table.description', 'Description')}
                          </TableCell>
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {purpose.elements.map((element) => (
                          <TableRow key={element.name}>
                            <TableCell sx={{ width: PURPOSE_ELEMENTS_COLUMN_WIDTHS.element }}>
                              <Tooltip
                                title={element.name}
                                disableHoverListener={
                                  element.name.length <= ELEMENT_NAME_MAX_DISPLAY_LENGTH
                                }
                              >
                                <Box
                                  component="code"
                                  sx={{
                                    display: 'inline-block',
                                    maxWidth: '100%',
                                    overflow: 'hidden',
                                    textOverflow: 'ellipsis',
                                    whiteSpace: 'nowrap',
                                    verticalAlign: 'bottom',
                                  }}
                                >
                                  {truncateElementName(
                                    element.name,
                                    ELEMENT_NAME_MAX_DISPLAY_LENGTH,
                                  )}
                                </Box>
                              </Tooltip>
                            </TableCell>
                            <TableCell>
                              {element.isUserApproved ? (
                                <Box sx={{ color: 'success.main', display: 'inline-flex' }}>
                                  <CheckCircle size={16} />
                                </Box>
                              ) : (
                                <Box
                                  role="img"
                                  aria-label={t(
                                    'consentRegistry.details.notApproved',
                                    'Not approved',
                                  )}
                                  sx={{ color: 'error.main', display: 'inline-flex' }}
                                >
                                  <XCircle size={16} />
                                </Box>
                              )}
                            </TableCell>
                            <TableCell>
                              <Chip
                                label={
                                  element.isMandatory
                                    ? t('consentRegistry.details.values.required', 'Required')
                                    : t('consentRegistry.details.values.optional', 'Optional')
                                }
                                size="small"
                                color={element.isMandatory ? 'error' : 'default'}
                                variant="outlined"
                              />
                            </TableCell>
                            <TableCell>{element.description ?? '-'}</TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </TableContainer>
                </AccordionDetails>
              </Accordion>
            )
          })}
        </CardContent>
      </Card>

      {/* Authorizations Section */}
      <Card sx={{ boxShadow: 1 }}>
        <CardHeader
          title={
            <Typography variant="subtitle1" fontWeight={600}>
              {t('consentRegistry.details.section.authorizations', 'Authorizations')}
            </Typography>
          }
          sx={{ pb: 0 }}
        />
        <Divider />
        <TableContainer>
          <Table sx={{ '& tbody tr:hover': { bgcolor: 'action.hover' } }}>
            <TableHead>
              <TableRow sx={{ bgcolor: 'action.default' }}>
                <TableCell sx={{ fontWeight: 700 }}>
                  {t('consentRegistry.details.table.user', 'User')}
                </TableCell>
                <TableCell sx={{ fontWeight: 700 }}>
                  {t('consentRegistry.details.table.status', 'Status')}
                </TableCell>
                <TableCell sx={{ fontWeight: 700 }}>
                  {t('consentRegistry.details.table.updated', 'Updated')}
                </TableCell>
                <TableCell sx={{ fontWeight: 700 }}>
                  {t('consentRegistry.details.table.resources', 'Resources')}
                </TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {(detail.authorizations ?? []).map((authorization) => {
                const resourcesEmpty = isEmptyAuthorizationResources(authorization.resources)

                return (
                  <TableRow key={authorization.id}>
                    <TableCell>
                      <Stack direction="row" spacing={1} alignItems="center">
                        <Avatar sx={{ width: 24, height: 24, fontSize: '0.75rem' }}>
                          {(authorization.userId ?? 'U').charAt(0).toUpperCase()}
                        </Avatar>
                        <Typography variant="body2">{authorization.userId ?? '-'}</Typography>
                      </Stack>
                    </TableCell>
                    <TableCell>
                      <Chip
                        label={t(
                          `consentRegistry.status.${getConsentStatusLabelKey(
                            authorization.status,
                            'authorization',
                          )}`,
                        )}
                        color={getConsentStatusChipColor(authorization.status)}
                        size="small"
                        variant="outlined"
                      />
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2">
                        {formatEpochTimestamp(authorization.updatedTime)}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Tooltip
                        title={t(
                          'consentRegistry.details.actions.noResourcesTooltip',
                          'No resources available',
                        )}
                        disableHoverListener={!resourcesEmpty}
                      >
                        <span>
                          <Button
                            size="small"
                            variant="contained"
                            color="secondary"
                            startIcon={<Eye size={14} />}
                            disabled={resourcesEmpty}
                            onClick={() => {
                              setSelectedResourcesJson(
                                formatResourcesForModal(authorization.resources),
                              )
                              setResourcesModalOpen(true)
                            }}
                          >
                            {t('consentRegistry.details.actions.viewResources', 'View Resources')}
                          </Button>
                        </span>
                      </Tooltip>
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </TableContainer>
      </Card>

      <Dialog
        open={resourcesModalOpen}
        onClose={() => {
          setResourcesModalOpen(false)
        }}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle sx={{ borderBottom: 1, borderColor: 'divider' }}>
          {t('consentRegistry.details.resourcesModal.title', 'Authorization Resources')}
        </DialogTitle>
        <DialogContent sx={{ pt: 2.5, pb: 2 }}>
          <Box
            component="pre"
            sx={{
              mb: 0,
              p: 2,
              border: 1,
              borderColor: 'divider',
              borderRadius: 1,
              bgcolor: 'background.default',
              color: 'text.primary',
              fontSize: '0.8125rem',
              fontFamily: 'monospace',
              whiteSpace: 'pre',
              overflow: 'auto',
              maxHeight: '60vh',
            }}
          >
            {selectedResourcesJson}
          </Box>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 3 }}>
          <Button
            variant="outlined"
            onClick={() => {
              setResourcesModalOpen(false)
            }}
          >
            {t('consentRegistry.details.resourcesModal.close', 'Close')}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Consent Lifecycle Section intentionally have mock data until backend API
        provides lifecycle timeline fields. */}

      <Card sx={{ boxShadow: 1 }}>
        <CardHeader
          title={
            <Typography variant="subtitle1" fontWeight={600}>
              {t('consentRegistry.details.section.lifecycle', 'Consent Lifecycle')}
            </Typography>
          }
          sx={{ pb: 0 }}
        />
        <Divider />
        <TableContainer>
          <Table sx={{ '& tbody tr:hover': { bgcolor: 'action.hover' } }}>
            <TableHead>
              <TableRow sx={{ bgcolor: 'action.default' }}>
                <TableCell sx={{ fontWeight: 700 }}>
                  {t('consentRegistry.details.table.eventType', 'Event Type')}
                </TableCell>
                <TableCell sx={{ fontWeight: 700 }}>
                  {t('consentRegistry.details.table.date', 'Date')}
                </TableCell>
                <TableCell sx={{ fontWeight: 700 }}>
                  {t('consentRegistry.details.table.time', 'Time')}
                </TableCell>
                <TableCell sx={{ fontWeight: 700 }}>
                  {t('consentRegistry.details.table.description', 'Description')}
                </TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              <TableRow>
                <TableCell>
                  <Stack direction="row" spacing={1} alignItems="center">
                    <Box
                      sx={{ width: 8, height: 8, borderRadius: '50%', bgcolor: 'action.disabled' }}
                    />
                    <Typography variant="body2" fontWeight={600}>
                      Pending
                    </Typography>
                  </Stack>
                </TableCell>
                <TableCell>
                  <Typography variant="body2">
                    {formatIsoDateTime('2026-03-01T09:15:04Z', LIFECYCLE_DATE_FORMAT_OPTIONS)}
                  </Typography>
                </TableCell>
                <TableCell>
                  <Typography variant="body2">
                    {formatIsoDateTime('2026-03-01T09:15:04Z', LIFECYCLE_TIME_FORMAT_OPTIONS)}
                  </Typography>
                </TableCell>
                <TableCell>
                  <Typography variant="body2">
                    Initial consent request generated by client application.
                  </Typography>
                </TableCell>
              </TableRow>
              <TableRow>
                <TableCell>
                  <Stack direction="row" spacing={1} alignItems="center">
                    <Box
                      sx={{ width: 8, height: 8, borderRadius: '50%', bgcolor: 'success.main' }}
                    />
                    <Typography variant="body2" fontWeight={600}>
                      Approved
                    </Typography>
                  </Stack>
                </TableCell>
                <TableCell>
                  <Typography variant="body2">
                    {formatIsoDateTime('2026-03-02T15:29:57Z', LIFECYCLE_DATE_FORMAT_OPTIONS)}
                  </Typography>
                </TableCell>
                <TableCell>
                  <Typography variant="body2">
                    {formatIsoDateTime('2026-03-02T15:29:57Z', LIFECYCLE_TIME_FORMAT_OPTIONS)}
                  </Typography>
                </TableCell>
                <TableCell>
                  <Typography variant="body2">
                    Authorization confirmed by user through banking portal.
                  </Typography>
                </TableCell>
              </TableRow>
              <TableRow>
                <TableCell>
                  <Stack direction="row" spacing={1} alignItems="center">
                    <Box
                      sx={{ width: 8, height: 8, borderRadius: '50%', bgcolor: 'primary.main' }}
                    />
                    <Typography variant="body2" fontWeight={600}>
                      Active
                    </Typography>
                  </Stack>
                </TableCell>
                <TableCell>
                  <Typography variant="body2">
                    {formatIsoDateTime('2026-03-02T15:30:00Z', LIFECYCLE_DATE_FORMAT_OPTIONS)}
                  </Typography>
                </TableCell>
                <TableCell>
                  <Typography variant="body2">
                    {formatIsoDateTime('2026-03-02T15:30:00Z', LIFECYCLE_TIME_FORMAT_OPTIONS)}
                  </Typography>
                </TableCell>
                <TableCell>
                  <Typography variant="body2">
                    Consent is now live and can be utilized for data sharing.
                  </Typography>
                </TableCell>
              </TableRow>
              <TableRow sx={{ opacity: 0.6 }}>
                <TableCell>
                  <Stack direction="row" spacing={1} alignItems="center">
                    <Box
                      sx={{ width: 8, height: 8, borderRadius: '50%', bgcolor: 'action.disabled' }}
                    />
                    <Typography variant="body2" fontWeight={600} color="text.secondary">
                      Expired
                    </Typography>
                  </Stack>
                </TableCell>
                <TableCell>
                  <Typography variant="body2" color="text.secondary">
                    {formatIsoDateTime('2026-05-30T23:59:59Z', LIFECYCLE_DATE_FORMAT_OPTIONS)}
                  </Typography>
                </TableCell>
                <TableCell>
                  <Typography variant="body2" color="text.secondary">
                    {formatIsoDateTime('2026-05-30T23:59:59Z', LIFECYCLE_TIME_FORMAT_OPTIONS)}
                  </Typography>
                </TableCell>
                <TableCell>
                  <Typography variant="body2" color="text.secondary">
                    Scheduled expiry date based on 90-day duration policy.
                  </Typography>
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </TableContainer>
      </Card>

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
