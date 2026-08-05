export const routes = ['/admin/018', '/admin/019', '/admin/020', '/admin/021']
export function canAccess(path, permissions = []) {
  return routes.includes(path) && permissions.includes('admin:read')
}
