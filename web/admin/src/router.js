export const routes = ['/admin/018', '/admin/019', '/admin/020', '/admin/021', '/admin/026', '/admin/027', '/admin/034', '/admin/035', '/admin/036', '/admin/037', '/admin/038', '/admin/039', '/admin/040']
export function canAccess(path, permissions = []) {
  return routes.includes(path) && permissions.includes('admin:read')
}
