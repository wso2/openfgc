export type ConsentStatus = 'Active' | 'Pending' | 'Revoked' | 'Expired'

export interface ConsentRecord {
  id: string
  clientName: string
  type: string
  status: ConsentStatus
  purposes: string[]
  createdAt: string
  canRevoke: boolean
  canApprove: boolean
}

export interface ConsentRegistryFilters {
  status: 'All' | ConsentStatus
  startDate: string
  endDate: string
  consentType: string
}
