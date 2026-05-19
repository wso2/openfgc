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
  Chip,
  IconButton,
  ListingTable,
  Popover,
  Skeleton,
  TablePagination,
  Tooltip,
  Typography,
} from '@wso2/oxygen-ui'
import { CircleCheckBig, Eye, ShieldX } from '@wso2/oxygen-ui-icons-react'
import { Fragment, type MouseEvent, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link as RouterLink, useNavigate } from 'react-router-dom'
import type { ConsentRecord } from '../../../types/consent'
import { formatEpochTimestamp, formatIsoDateTime } from '../../../utils/dateTime'
import { CONSENT_REGISTRY_ROWS_PER_PAGE_OPTIONS } from '../constants'
import { getConsentStatusChipColor, getConsentStatusLabelKey } from '../utils/statusChip'

interface ConsentRegistryTableProps {
  rows: ConsentRecord[]
  totalCount: number
  isLoading: boolean
  page: number
  rowsPerPage: number
  onPageChange: (page: number) => void
  onRowsPerPageChange: (rowsPerPage: number) => void
  onApprove: (consentID: string) => void
  onRevoke: (consentID: string) => void
  isMutating: boolean
}

type SortField = 'type' | 'status' | 'updatedAt' | 'expirationTime'
type SortDirection = 'asc' | 'desc'

const DATE_TIME_FORMAT_OPTIONS: Intl.DateTimeFormatOptions = {
  month: 'short',
  day: '2-digit',
  year: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
  hour12: false,
}

const PURPOSE_PREVIEW_COUNT = 2

const CONSENT_REGISTRY_COLUMN_WIDTHS = {
  purposes: '34%',
  type: '11%',
  status: '13%',
  updated: '16%',
  expiration: '16%',
  actions: '10%',
} as const

function sortConsentRows(
  rows: ConsentRecord[],
  sortField: SortField,
  sortDirection: SortDirection,
): ConsentRecord[] {
  const sortedRows = [...rows].sort((leftRow, rightRow) => {
    const leftValue = leftRow[sortField]
    const rightValue = rightRow[sortField]

    if (leftValue == null && rightValue == null) {
      return 0
    }

    if (leftValue == null) {
      return 1
    }

    if (rightValue == null) {
      return -1
    }

    if (leftValue < rightValue) {
      return -1
    }

    if (leftValue > rightValue) {
      return 1
    }

    return 0
  })

  return sortDirection === 'asc' ? sortedRows : sortedRows.reverse()
}

function ConsentRegistryTable({
  rows,
  totalCount,
  isLoading,
  page,
  rowsPerPage,
  onPageChange,
  onRowsPerPageChange,
  onApprove,
  onRevoke,
  isMutating,
}: ConsentRegistryTableProps): React.JSX.Element {
  const { t } = useTranslation('common')
  const navigate = useNavigate()
  const [sortField, setSortField] = useState<SortField>('updatedAt')
  const [sortDirection, setSortDirection] = useState<SortDirection>('desc')
  const [purposesPopoverAnchor, setPurposesPopoverAnchor] = useState<HTMLElement | null>(null)
  const [selectedPurposes, setSelectedPurposes] = useState<string[]>([])

  const sortedRows = useMemo(
    () => sortConsentRows(rows, sortField, sortDirection),
    [rows, sortDirection, sortField],
  )

  const groupedRows = useMemo(() => {
    const groupedMap = new Map<string, ConsentRecord[]>()

    sortedRows.forEach((row) => {
      const existingRows = groupedMap.get(row.clientName)

      if (existingRows) {
        groupedMap.set(row.clientName, [...existingRows, row])
        return
      }

      groupedMap.set(row.clientName, [row])
    })

    return Array.from(groupedMap.entries()).map(([clientName, clientRows]) => ({
      clientName,
      clientRows,
    }))
  }, [sortedRows])

  const selectedRowIds: readonly string[] = []
  const isPurposesPopoverOpen = Boolean(purposesPopoverAnchor)

  const handlePurposesPopoverClose = (): void => {
    setPurposesPopoverAnchor(null)
    setSelectedPurposes([])
  }

  const handleStopPropagation = (event: MouseEvent<HTMLElement>): void => {
    event.stopPropagation()
  }

  return (
    <ListingTable.Provider
      density="standard"
      page={page}
      rowsPerPage={rowsPerPage}
      totalCount={totalCount}
      selected={selectedRowIds}
      sortField={sortField}
      sortDirection={sortDirection}
      isSelected={() => false}
      onBulkDelete={() => {}}
      onClearSelection={() => {}}
      onDensityChange={() => {}}
      onPageChange={onPageChange}
      onRowsPerPageChange={onRowsPerPageChange}
      onSearchChange={() => {}}
      onSelectAll={() => {}}
      onSelectionChange={() => {}}
      onSortChange={(nextField, nextDirection) => {
        setSortField(nextField as SortField)
        setSortDirection(nextDirection)
      }}
    >
      <ListingTable.Container sx={{ minWidth: 1080 }}>
        <ListingTable
          density="standard"
          variant="table"
          aria-label={t('consentRegistry.table.tableAriaLabel')}
          sx={{ tableLayout: 'fixed' }}
        >
          <ListingTable.Head>
            <ListingTable.Row>
              <ListingTable.Cell sx={{ width: CONSENT_REGISTRY_COLUMN_WIDTHS.purposes }}>
                {t('consentRegistry.table.headers.purposes')}
              </ListingTable.Cell>
              <ListingTable.Cell sx={{ width: CONSENT_REGISTRY_COLUMN_WIDTHS.type }}>
                <ListingTable.SortLabel field="type">
                  {t('consentRegistry.table.headers.type')}
                </ListingTable.SortLabel>
              </ListingTable.Cell>
              <ListingTable.Cell sx={{ width: CONSENT_REGISTRY_COLUMN_WIDTHS.status }}>
                <ListingTable.SortLabel field="status">
                  {t('consentRegistry.table.headers.status')}
                </ListingTable.SortLabel>
              </ListingTable.Cell>
              <ListingTable.Cell sx={{ width: CONSENT_REGISTRY_COLUMN_WIDTHS.updated }}>
                <ListingTable.SortLabel field="updatedAt">
                  {t('consentRegistry.table.headers.updated')}
                </ListingTable.SortLabel>
              </ListingTable.Cell>
              <ListingTable.Cell sx={{ width: CONSENT_REGISTRY_COLUMN_WIDTHS.expiration }}>
                <ListingTable.SortLabel field="expirationTime">
                  {t('consentRegistry.table.headers.expiration')}
                </ListingTable.SortLabel>
              </ListingTable.Cell>
              <ListingTable.Cell
                align="center"
                sx={{ width: CONSENT_REGISTRY_COLUMN_WIDTHS.actions }}
              >
                {t('consentRegistry.table.headers.actions')}
              </ListingTable.Cell>
            </ListingTable.Row>
          </ListingTable.Head>

          <ListingTable.Body>
            {isLoading
              ? Array.from({ length: rowsPerPage }, (_, rowIndex) => (
                  <ListingTable.Row key={`skeleton-row-${rowIndex}`} variant="table">
                    <ListingTable.Cell sx={{ width: CONSENT_REGISTRY_COLUMN_WIDTHS.purposes }}>
                      <Box sx={{ display: 'flex', gap: 0.75, flexWrap: 'wrap' }}>
                        <Skeleton variant="rounded" width={140} height={24} />
                      </Box>
                    </ListingTable.Cell>
                    <ListingTable.Cell sx={{ width: CONSENT_REGISTRY_COLUMN_WIDTHS.type }}>
                      <Skeleton variant="text" width="70%" />
                    </ListingTable.Cell>
                    <ListingTable.Cell sx={{ width: CONSENT_REGISTRY_COLUMN_WIDTHS.status }}>
                      <Skeleton variant="rounded" width={72} height={24} />
                    </ListingTable.Cell>
                    <ListingTable.Cell sx={{ width: CONSENT_REGISTRY_COLUMN_WIDTHS.updated }}>
                      <Skeleton variant="text" width="86%" />
                    </ListingTable.Cell>
                    <ListingTable.Cell sx={{ width: CONSENT_REGISTRY_COLUMN_WIDTHS.expiration }}>
                      <Skeleton variant="text" width="86%" />
                    </ListingTable.Cell>
                    <ListingTable.Cell
                      align="center"
                      sx={{ width: CONSENT_REGISTRY_COLUMN_WIDTHS.actions }}
                    >
                      <Box sx={{ display: 'flex', justifyContent: 'center', gap: 0.75 }}>
                        <Skeleton variant="circular" width={24} height={24} />
                        <Skeleton variant="circular" width={24} height={24} />
                      </Box>
                    </ListingTable.Cell>
                  </ListingTable.Row>
                ))
              : groupedRows.map((group) => (
                  <Fragment key={group.clientName}>
                    <ListingTable.Row
                      variant="table"
                      sx={{
                        bgcolor: 'action.hover',
                      }}
                    >
                      <ListingTable.Cell colSpan={6} sx={{ fontWeight: 700 }}>
                        {t('consentRegistry.table.clientLabel', { client: group.clientName })}
                      </ListingTable.Cell>
                    </ListingTable.Row>

                    {group.clientRows.map((row) => (
                      <ListingTable.Row
                        key={row.id}
                        hover
                        variant="table"
                        onClick={() => {
                          navigate(`/consents/${encodeURIComponent(row.id)}`)
                        }}
                        sx={{ cursor: 'pointer' }}
                      >
                        <ListingTable.Cell
                          sx={{ width: CONSENT_REGISTRY_COLUMN_WIDTHS.purposes, fontWeight: 500 }}
                        >
                          <Box
                            sx={{
                              display: 'flex',
                              alignItems: 'center',
                              gap: 0.75,
                              flexWrap: 'wrap',
                            }}
                          >
                            {row.purposes.slice(0, PURPOSE_PREVIEW_COUNT).map((purpose) => (
                              <Chip
                                key={`${row.id}-${purpose}`}
                                size="small"
                                label={purpose}
                                variant="outlined"
                              />
                            ))}
                            {row.purposes.length > PURPOSE_PREVIEW_COUNT ? (
                              <Chip
                                size="small"
                                color="primary"
                                variant="outlined"
                                label={t('consentRegistry.table.purposes.more', {
                                  count: row.purposes.length - PURPOSE_PREVIEW_COUNT,
                                  defaultValue: '+{{count}} more',
                                })}
                                onClick={(event) => {
                                  event.stopPropagation()
                                  setPurposesPopoverAnchor(event.currentTarget)
                                  setSelectedPurposes(row.purposes)
                                }}
                              />
                            ) : null}
                          </Box>
                        </ListingTable.Cell>
                        <ListingTable.Cell sx={{ width: CONSENT_REGISTRY_COLUMN_WIDTHS.type }}>
                          {row.type}
                        </ListingTable.Cell>
                        <ListingTable.Cell sx={{ width: CONSENT_REGISTRY_COLUMN_WIDTHS.status }}>
                          <Chip
                            size="small"
                            color={getConsentStatusChipColor(row.status)}
                            label={t(
                              `consentRegistry.status.${getConsentStatusLabelKey(row.status)}`,
                            )}
                            variant="outlined"
                          />
                        </ListingTable.Cell>
                        <ListingTable.Cell sx={{ width: CONSENT_REGISTRY_COLUMN_WIDTHS.updated }}>
                          {formatIsoDateTime(row.updatedAt, DATE_TIME_FORMAT_OPTIONS)}
                        </ListingTable.Cell>
                        <ListingTable.Cell
                          sx={
                            row.expirationTime === 0
                              ? {
                                  width: CONSENT_REGISTRY_COLUMN_WIDTHS.expiration,
                                  color: 'text.disabled',
                                }
                              : {
                                  width: CONSENT_REGISTRY_COLUMN_WIDTHS.expiration,
                                }
                          }
                        >
                          {row.expirationTime === 0
                            ? t('consentRegistry.table.notApplicable')
                            : formatEpochTimestamp(row.expirationTime, DATE_TIME_FORMAT_OPTIONS)}
                        </ListingTable.Cell>
                        <ListingTable.Cell
                          align="center"
                          sx={{ width: CONSENT_REGISTRY_COLUMN_WIDTHS.actions }}
                        >
                          <ListingTable.RowActions visibility="always">
                            <Tooltip title={t('consentRegistry.actions.view')}>
                              <IconButton
                                size="small"
                                component={RouterLink}
                                to={`/consents/${encodeURIComponent(row.id)}`}
                                aria-label={t('consentRegistry.actions.view')}
                                onClick={handleStopPropagation}
                              >
                                <Eye size={16} />
                              </IconButton>
                            </Tooltip>
                            {row.canApprove ? (
                              <Tooltip title={t('consentRegistry.actions.approve')}>
                                <span>
                                  <IconButton
                                    size="small"
                                    color="warning"
                                    aria-label={t('consentRegistry.actions.approve')}
                                    disabled={isMutating}
                                    onClick={(event) => {
                                      event.stopPropagation()
                                      onApprove(row.id)
                                    }}
                                  >
                                    <CircleCheckBig size={16} />
                                  </IconButton>
                                </span>
                              </Tooltip>
                            ) : (
                              <Tooltip title={t('consentRegistry.actions.revoke')}>
                                <span>
                                  <IconButton
                                    size="small"
                                    color="error"
                                    disabled={!row.canRevoke || isMutating}
                                    aria-label={t('consentRegistry.actions.revoke')}
                                    onClick={(event) => {
                                      event.stopPropagation()
                                      onRevoke(row.id)
                                    }}
                                  >
                                    <ShieldX size={16} />
                                  </IconButton>
                                </span>
                              </Tooltip>
                            )}
                          </ListingTable.RowActions>
                        </ListingTable.Cell>
                      </ListingTable.Row>
                    ))}
                  </Fragment>
                ))}
          </ListingTable.Body>
        </ListingTable>

        <Popover
          open={isPurposesPopoverOpen}
          anchorEl={purposesPopoverAnchor}
          onClose={handlePurposesPopoverClose}
          anchorOrigin={{ vertical: 'bottom', horizontal: 'left' }}
          transformOrigin={{ vertical: -4, horizontal: 'left' }}
          PaperProps={{
            sx: {
              mt: 0.5,
              borderRadius: 1,
              border: 2,
              borderColor: 'divider',
              boxShadow: 6,
              overflow: 'hidden',
            },
          }}
        >
          <Box sx={{ minWidth: 260, maxWidth: 420 }}>
            <Box
              sx={{
                px: 2,
                py: 1.5,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                bgcolor: 'action.hover',
                borderBottom: 1,
                borderColor: 'divider',
              }}
            >
              <Typography variant="subtitle2" sx={{ fontWeight: 700 }}>
                {t('consentRegistry.table.purposes.title', 'Consent purposes')}
              </Typography>
              <Chip
                size="small"
                color="default"
                variant="filled"
                label={selectedPurposes.length}
                sx={{ height: 20, '& .MuiChip-label': { px: 0.75, fontWeight: 600 } }}
              />
            </Box>
            <Box
              sx={{
                p: 2,
                display: 'flex',
                gap: 0.75,
                flexWrap: 'wrap',
                maxHeight: 280,
                overflowY: 'auto',
              }}
            >
              {selectedPurposes.map((purpose) => (
                <Chip key={purpose} size="small" label={purpose} variant="outlined" />
              ))}
            </Box>
            <Box
              sx={{
                px: 2,
                py: 1,
                borderTop: 1,
                borderColor: 'divider',
                bgcolor: 'background.paper',
              }}
            >
              <Typography variant="caption" color="text.secondary">
                {t('consentRegistry.table.purposes.hint', 'Showing all purposes of the consent')}
              </Typography>
            </Box>
          </Box>
        </Popover>

        <TablePagination
          component="div"
          count={totalCount}
          page={page}
          rowsPerPage={rowsPerPage}
          rowsPerPageOptions={[...CONSENT_REGISTRY_ROWS_PER_PAGE_OPTIONS]}
          onPageChange={(_, nextPage) => {
            onPageChange(nextPage)
          }}
          onRowsPerPageChange={(event) => {
            onRowsPerPageChange(Number(event.target.value))
          }}
        />
      </ListingTable.Container>
    </ListingTable.Provider>
  )
}

export default ConsentRegistryTable
