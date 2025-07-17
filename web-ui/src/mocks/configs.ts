export const configs = [
  {
    tenant: 'tenant-a',
    configDrift: false,
    auditStatus: 'Compliant',
    details: 'All configs match desired state.'
  },
  {
    tenant: 'tenant-b',
    configDrift: true,
    auditStatus: 'Drifted',
    details: 'Resource quota mismatch.'
  },
  {
    tenant: 'tenant-c',
    configDrift: false,
    auditStatus: 'Compliant',
    details: 'All configs match desired state.'
  },
]; 