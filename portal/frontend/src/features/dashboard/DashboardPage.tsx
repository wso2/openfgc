import { Box, Stack, Typography } from '@wso2/oxygen-ui'
import { useTranslation } from 'react-i18next'
import HeaderBreadcrumbs from '../../components/layout/main-layout/HeaderBreadcrumbs'

function DashboardPage(): React.JSX.Element {
  const { t } = useTranslation('common')

  return (
    <Box component="main" sx={{ p: { xs: 2, md: 4 } }}>
      <Stack spacing={1}>
        <HeaderBreadcrumbs />
        <Typography variant="h4" fontWeight={700}>
          {t('dashboard.title')}
        </Typography>
      </Stack>
    </Box>
  )
}

export default DashboardPage
