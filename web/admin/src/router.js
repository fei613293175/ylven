export const routes = ['/admin/018', '/admin/019', '/admin/020', '/admin/021', '/admin/026', '/admin/027', '/admin/034', '/admin/035', '/admin/036', '/admin/037', '/admin/038', '/admin/039', '/admin/040', '/admin/051', '/admin/062', '/admin/063', '/admin/064', '/admin/065', '/admin/066', '/admin/067', '/admin/068', '/admin/069', '/admin/070', '/admin/071', '/admin/072', '/admin/073', '/admin/074', '/admin/075', '/admin/p03-home', '/admin/p03-conversations', '/admin/p03-exports', '/admin/p03-metrics']
export function canAccess(path, permissions = []) {
  return routes.includes(path) && permissions.includes('admin:read')
}
