import { Box, Stack, Typography } from '@wso2/oxygen-ui'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router-dom'
import HeaderBreadcrumbs from '../../components/layout/main-layout/HeaderBreadcrumbs'
import ConsentRegistryFilters from './components/ConsentRegistryFilters'
import ConsentRegistryTable from './components/ConsentRegistryTable'
import { CONSENT_REGISTRY_MOCK_DATA } from './data/consents'
import type {
  ConsentRegistryFilters as ConsentRegistryFiltersModel,
  ConsentRecord,
} from '../../types/consent'

const DEFAULT_FILTERS: ConsentRegistryFiltersModel = {
  status: 'All',
  startDate: '',
  endDate: '',
  consentType: '',
}

const FILTER_STATUS_VALUES: ConsentRegistryFiltersModel['status'][] = [
  'All',
  'Active',
  'Pending',
  'Revoked',
  'Expired',
]

function isValidFilterStatus(value: string): value is ConsentRegistryFiltersModel['status'] {
  return FILTER_STATUS_VALUES.includes(value as ConsentRegistryFiltersModel['status'])
}

function getFiltersFromSearchParams(searchParams: URLSearchParams): ConsentRegistryFiltersModel {
  const statusParam = searchParams.get('status')

  return {
    status: statusParam && isValidFilterStatus(statusParam) ? statusParam : DEFAULT_FILTERS.status,
    startDate: searchParams.get('startDate') ?? DEFAULT_FILTERS.startDate,
    endDate: searchParams.get('endDate') ?? DEFAULT_FILTERS.endDate,
    consentType: searchParams.get('consentType') ?? DEFAULT_FILTERS.consentType,
  }
}

function toSearchParams(filters: ConsentRegistryFiltersModel): URLSearchParams {
  const params = new URLSearchParams()

  if (filters.status !== DEFAULT_FILTERS.status) {
    params.set('status', filters.status)
  }

  if (filters.startDate) {
    params.set('startDate', filters.startDate)
  }

  if (filters.endDate) {
    params.set('endDate', filters.endDate)
  }

  if (filters.consentType.trim()) {
    params.set('consentType', filters.consentType.trim())
  }

  return params
}

function isWithinDateRange(record: ConsentRecord, startDate: string, endDate: string): boolean {
  const createdAtDate = new Date(record.createdAt)

  if (startDate) {
    const normalizedStartDate = new Date(`${startDate}T00:00:00`)

    if (createdAtDate < normalizedStartDate) {
      return false
    }
  }

  if (endDate) {
    const normalizedEndDate = new Date(`${endDate}T23:59:59`)

    if (createdAtDate > normalizedEndDate) {
      return false
    }
  }

  return true
}

function ConsentRegistryPage(): React.JSX.Element {
  const { t } = useTranslation('common')
  const [searchParams, setSearchParams] = useSearchParams()

  const filters = useMemo(() => getFiltersFromSearchParams(searchParams), [searchParams])

  const filteredRows = useMemo(() => {
    return CONSENT_REGISTRY_MOCK_DATA.filter((record) => {
      const statusMatch = filters.status === 'All' || record.status === filters.status
      const typeMatch = record.type.toLowerCase().includes(filters.consentType.trim().toLowerCase())
      const dateRangeMatch = isWithinDateRange(record, filters.startDate, filters.endDate)

      return statusMatch && typeMatch && dateRangeMatch
    })
  }, [filters])

  return (
    <Box component="main" sx={{ p: { xs: 2, md: 4 } }}>
      <Stack spacing={3}>
        <Stack spacing={1}>
          <HeaderBreadcrumbs />
          <Typography variant="h4" fontWeight={700}>
            {t('consentRegistry.title')}
          </Typography>
        </Stack>

        <ConsentRegistryFilters
          filters={filters}
          onFilterChange={(nextFilters) => {
            setSearchParams(toSearchParams(nextFilters), { replace: true })
          }}
          onClear={() => {
            setSearchParams({}, { replace: true })
          }}
        />

        <ConsentRegistryTable rows={filteredRows} />
      </Stack>
    </Box>
  )
}

export default ConsentRegistryPage
