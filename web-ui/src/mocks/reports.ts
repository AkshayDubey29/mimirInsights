export const reports = [
  {
    tenant: 'tenant-a',
    capacityStatus: 'Optimal',
    alloyTuning: 'No change needed',
    details: 'Current allocation matches usage.'
  },
  {
    tenant: 'tenant-b',
    capacityStatus: 'Under-provisioned',
    alloyTuning: 'Increase replicas',
    details: 'CPU usage exceeds 80% of limit.'
  },
  {
    tenant: 'tenant-c',
    capacityStatus: 'Over-provisioned',
    alloyTuning: 'Reduce replicas',
    details: 'Memory usage below 30% of limit.'
  },
]; 