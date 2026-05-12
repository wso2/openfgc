import { Box, Breadcrumbs, Link, Typography } from '@wso2/oxygen-ui'
import { ChevronRight } from '@wso2/oxygen-ui-icons-react'
import { useTranslation } from 'react-i18next'
import { Link as RouterLink, useLocation } from 'react-router-dom'

interface BreadcrumbItem {
  label: string
  path: string
  isCurrent: boolean
}

function buildBreadcrumbItems(
  pathname: string,
  homeLabel: string,
  consentsLabel: string,
): BreadcrumbItem[] {
  const consentDetailsMatch = pathname.match(/^\/consents\/([^/]+)$/)

  if (consentDetailsMatch) {
    return [
      {
        label: homeLabel,
        path: '/dashboard',
        isCurrent: false,
      },
      {
        label: consentsLabel,
        path: '/consents',
        isCurrent: false,
      },
      {
        label: consentDetailsMatch[1],
        path: pathname,
        isCurrent: true,
      },
    ]
  }

  if (pathname.startsWith('/consents')) {
    return [
      {
        label: homeLabel,
        path: '/dashboard',
        isCurrent: false,
      },
      {
        label: consentsLabel,
        path: '/consents',
        isCurrent: true,
      },
    ]
  }

  return [
    {
      label: homeLabel,
      path: '/dashboard',
      isCurrent: true,
    },
  ]
}

function HeaderBreadcrumbs(): React.JSX.Element {
  const { t } = useTranslation('common')
  const location = useLocation()

  const breadcrumbItems = buildBreadcrumbItems(
    location.pathname,
    t('layout.home'),
    t('sidebar.allConsents'),
  )

  return (
    <Box component="nav" aria-label={t('layout.breadcrumbAriaLabel')}>
      <Breadcrumbs
        separator={
          <Box component="span" sx={{ display: 'inline-flex', transform: 'translateY(1px)' }}>
            <ChevronRight size={14} aria-hidden="true" />
          </Box>
        }
      >
        {breadcrumbItems.map((item) =>
          item.isCurrent ? (
            <Typography
              key={item.path}
              component="span"
              variant="body2"
              color="text.primary"
              fontWeight={600}
              aria-current="page"
            >
              {item.label}
            </Typography>
          ) : (
            <Link
              key={item.path}
              component={RouterLink}
              to={item.path}
              underline="hover"
              color="text.secondary"
              variant="body2"
              sx={{ '&:hover': { color: 'text.primary' } }}
            >
              {item.label}
            </Link>
          ),
        )}
      </Breadcrumbs>
    </Box>
  )
}

export default HeaderBreadcrumbs
