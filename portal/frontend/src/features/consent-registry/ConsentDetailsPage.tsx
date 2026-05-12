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
  Divider,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
  Avatar,
} from '@wso2/oxygen-ui'
import { ChevronRight, Download, CheckCircle, XCircle } from '@wso2/oxygen-ui-icons-react'
import { useTranslation } from 'react-i18next'
import { useParams, useNavigate } from 'react-router-dom'
import HeaderBreadcrumbs from '../../components/layout/main-layout/HeaderBreadcrumbs'

function ConsentDetailsPage(): React.JSX.Element {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

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
          <Button
            variant="outlined"
            color="primary"
            startIcon={<Download size={16} />}
            size="small"
          >
            {t('consentRegistry.details.download', 'Download')}
          </Button>
          <Button variant="contained" color="error" size="small">
            {t('consentRegistry.details.revoke', 'Revoke')}
          </Button>
        </Stack>
      </Box>

      {/* Consent Details Section */}
      <Card sx={{ boxShadow: 1 }}>
        <CardHeader
          title={
            <Stack direction="row" spacing={1} alignItems="center">
              <Typography variant="body2" fontWeight={600}>
                {t('consentRegistry.details.consentId', 'Consent ID')}: {id}
              </Typography>
            </Stack>
          }
          action={
            <Stack direction="row" spacing={1}>
              <Chip
                label={t('consentRegistry.status.active', 'ACTIVE')}
                color="success"
                size="small"
                variant="filled"
              />
              <Chip
                label={t('consentRegistry.details.consentType', 'Account Access')}
                size="small"
                variant="filled"
                sx={{ bgcolor: 'warning.main', color: 'warning.contrastText' }}
              />
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
                fontWeight={600}
                sx={{ display: 'block', mb: 1, textTransform: 'uppercase', letterSpacing: 0.5 }}
              >
                {t('consentRegistry.details.clientId', 'Client ID')}
              </Typography>
              <Typography variant="body2" fontWeight={600}>
                client-12345
              </Typography>
            </Box>
            <Box>
              <Typography
                variant="caption"
                color="text.secondary"
                fontWeight={600}
                sx={{ display: 'block', mb: 1, textTransform: 'uppercase', letterSpacing: 0.5 }}
              >
                {t('consentRegistry.details.recurring', 'Recurring')}
              </Typography>
              <Typography variant="body2" fontWeight={600}>
                Yes
              </Typography>
            </Box>
            <Box>
              <Typography
                variant="caption"
                color="text.secondary"
                fontWeight={600}
                sx={{ display: 'block', mb: 1, textTransform: 'uppercase', letterSpacing: 0.5 }}
              >
                {t('consentRegistry.details.frequency', 'Frequency')}
              </Typography>
              <Typography variant="body2" fontWeight={600}>
                Monthly
              </Typography>
            </Box>
            <Box>
              <Typography
                variant="caption"
                color="text.secondary"
                fontWeight={600}
                sx={{ display: 'block', mb: 1, textTransform: 'uppercase', letterSpacing: 0.5 }}
              >
                {t('consentRegistry.details.duration', 'Duration')}
              </Typography>
              <Typography variant="body2" fontWeight={600}>
                12 Months
              </Typography>
            </Box>
            <Box>
              <Typography
                variant="caption"
                color="text.secondary"
                fontWeight={600}
                sx={{ display: 'block', mb: 1, textTransform: 'uppercase', letterSpacing: 0.5 }}
              >
                {t('consentRegistry.details.created', 'Created')}
              </Typography>
              <Typography variant="body2" fontWeight={600}>
                2023-10-01
              </Typography>
            </Box>
            <Box>
              <Typography
                variant="caption"
                color="text.secondary"
                fontWeight={600}
                sx={{ display: 'block', mb: 1, textTransform: 'uppercase', letterSpacing: 0.5 }}
              >
                {t('consentRegistry.details.validUntil', 'Valid Until')}
              </Typography>
              <Typography variant="body2" fontWeight={600}>
                2024-10-01
              </Typography>
            </Box>
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
          <Accordion
            disableGutters
            elevation={0}
            sx={{
              border: 1,
              borderColor: 'divider',
              borderRadius: 1,
              overflow: 'hidden',
              '&:before': { display: 'none' },
              '&.Mui-expanded': { m: 0 },
            }}
          >
            <AccordionSummary
              expandIcon={<ChevronRight />}
              sx={{ '&:hover': { bgcolor: 'action.hover' } }}
            >
              <Stack direction="row" spacing={1.5} alignItems="center">
                <Typography variant="body2" fontWeight={600}>
                  {t('consentRegistry.details.purpose.accountAccess', 'Account Access')}
                </Typography>
                <Chip
                  label="2/3 approved"
                  color="primary"
                  size="small"
                  sx={{
                    height: 20,
                    '& .MuiChip-label': { px: 0.75, fontSize: '0.6875rem', fontWeight: 500 },
                  }}
                />
              </Stack>
            </AccordionSummary>
            <AccordionDetails sx={{ p: 0 }}>
              <TableContainer>
                <Table size="small" sx={{ '& tbody tr:hover': { bgcolor: 'action.hover' } }}>
                  <TableHead>
                    <TableRow>
                      <TableCell sx={{ fontWeight: 700 }}>
                        {t('consentRegistry.details.table.element', 'Element')}
                      </TableCell>
                      <TableCell sx={{ fontWeight: 700 }}>
                        {t('consentRegistry.details.table.approved', 'Approved')}
                      </TableCell>
                      <TableCell sx={{ fontWeight: 700 }}>
                        {t('consentRegistry.details.table.required', 'Required')}
                      </TableCell>
                      <TableCell sx={{ fontWeight: 700 }}>
                        {t('consentRegistry.details.table.type', 'Type')}
                      </TableCell>
                      <TableCell sx={{ fontWeight: 700 }}>
                        {t('consentRegistry.details.table.description', 'Description')}
                      </TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    <TableRow>
                      <TableCell>
                        <code>accountNumber</code>
                      </TableCell>
                      <TableCell>
                        <Box sx={{ color: 'success.main', display: 'inline-flex' }}>
                          <CheckCircle size={16} />
                        </Box>
                      </TableCell>
                      <TableCell>
                        <Chip label="Required" size="small" color="error" variant="outlined" />
                      </TableCell>
                      <TableCell>string</TableCell>
                      <TableCell>Primary account number</TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell>
                        <code>balance</code>
                      </TableCell>
                      <TableCell>
                        <Box sx={{ color: 'success.main', display: 'inline-flex' }}>
                          <CheckCircle size={16} />
                        </Box>
                      </TableCell>
                      <TableCell>
                        <Chip label="Optional" size="small" variant="outlined" />
                      </TableCell>
                      <TableCell>number</TableCell>
                      <TableCell>Current account balance</TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell>
                        <code>transactionHistory</code>
                      </TableCell>
                      <TableCell>
                        <Box
                          role="img"
                          aria-label={t('consentRegistry.details.notApproved', 'Not approved')}
                          sx={{ color: 'error.main', display: 'inline-flex' }}
                        >
                          <XCircle size={16} />
                        </Box>
                      </TableCell>
                      <TableCell>
                        <Chip label="Optional" size="small" variant="outlined" />
                      </TableCell>
                      <TableCell>array</TableCell>
                      <TableCell>Last 90 days of transactions</TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </TableContainer>
            </AccordionDetails>
          </Accordion>
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
                  {t('consentRegistry.details.table.authId', 'Auth ID')}
                </TableCell>
                <TableCell sx={{ fontWeight: 700 }}>
                  {t('consentRegistry.details.table.user', 'User')}
                </TableCell>
                <TableCell sx={{ fontWeight: 700 }}>
                  {t('consentRegistry.details.table.type', 'Type')}
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
              <TableRow>
                <TableCell>
                  <code>auth-001...</code>
                </TableCell>
                <TableCell>
                  <Stack direction="row" spacing={1} alignItems="center">
                    <Avatar sx={{ width: 24, height: 24, fontSize: '0.75rem' }}>U</Avatar>
                    <Typography variant="body2">user@example.com</Typography>
                  </Stack>
                </TableCell>
                <TableCell>authorisation</TableCell>
                <TableCell>
                  <Chip label="Approved" color="success" size="small" variant="filled" />
                </TableCell>
                <TableCell>
                  <Typography variant="body2">02/03/2026, 15:29:57</Typography>
                </TableCell>
                <TableCell>—</TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </TableContainer>
      </Card>

      {/* Consent Lifecycle Section */}
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
                  <Typography variant="body2">01/03/2026</Typography>
                </TableCell>
                <TableCell>
                  <Typography variant="body2">09:15:04 AM</Typography>
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
                  <Typography variant="body2">02/03/2026</Typography>
                </TableCell>
                <TableCell>
                  <Typography variant="body2">03:29:57 PM</Typography>
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
                  <Typography variant="body2">02/03/2026</Typography>
                </TableCell>
                <TableCell>
                  <Typography variant="body2">03:30:00 PM</Typography>
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
                    30/05/2026
                  </Typography>
                </TableCell>
                <TableCell>
                  <Typography variant="body2" color="text.secondary">
                    11:59:59 PM
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
    </Box>
  )
}

export default ConsentDetailsPage
