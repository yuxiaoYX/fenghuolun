const TOKEN_KEY = 'fhl_admin'

export type Envelope<T> = {
  ok?: boolean
  data?: T
  error?: { code?: string; message?: string }
}

export class ApiError extends Error {
  code: string
  status: number
  constructor(message: string, code: string, status: number) {
    super(message)
    this.code = code
    this.status = status
  }
}

export function getToken(): string {
  return localStorage.getItem(TOKEN_KEY) ?? ''
}

export function setToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY)
}

export async function adminFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  if (!headers.has('Content-Type') && init.body) {
    headers.set('Content-Type', 'application/json')
  }
  const token = getToken()
  if (token) {
    headers.set('Authorization', 'Bearer ' + token)
  }
  let res: Response
  try {
    res = await fetch(path, { ...init, headers })
  } catch {
    throw new ApiError('后端未启动（默认 http://127.0.0.1:8088）', 'upstream', 0)
  }
  const body = (await res.json()) as Envelope<T>
  if (!body.ok) {
    const code = body.error?.code ?? 'error'
    if (code === 'unauthorized' && path !== '/api/v1/admin/login' && path !== '/api/v1/admin/password') {
      clearToken()
      if (location.pathname !== '/login') {
        location.href = '/login'
      }
    }
    throw new ApiError(body.error?.message ?? '请求失败', code, res.status)
  }
  return body.data as T
}

export type BindingRow = {
  id: string
  vinMasked: string
  modelCode: string
  modelName: string
  lng?: number | null
  lat?: number | null
  syncStatus: string
  syncError: string
  syncedAt: string | null
  disabled: boolean
}

export type JobRow = {
  id: string
  bindingId: string
  vinMasked: string
  kind: string
  status: string
  errorPublic: string
  errorInternal: string
  startedAt: string | null
  finishedAt: string | null
}

export type BindingDetail = BindingRow & {
  nickname: string
  trim: string
  isExtender: boolean
  refreshHint: string
  hasRefresh: boolean
  hasAccess: boolean
  sessionCount: number
  locReportedAt: string | null
  snapshot: Record<string, unknown> | null
  stale: boolean
  energy: {
    periodType?: number
    days?: Array<{
      countTime?: string
      totalKwh?: number | null
      drivingKwh?: number | null
      acKwh?: number | null
      recoveryKwh?: number | null
    }>
    totalKwh?: number | null
    fetchedAt?: string | null
  } | null
  jobs: JobRow[]
}

export type AdminHealth = {
  bindingsTotal: number
  bindingsActive: number
  bindingsDisabled: number
  tokenInvalid: number
  syncOkToday: number
  syncFailToday: number
  upstreamStreak: number
  cronSync: string
  settings: Settings
}

export type Settings = {
  cronSync: string
  corsOrigins: string
  snapshotKeep: number
  staleAfterSec: number
}

export type TableMeta = { name: string; label: string }

export type TablePage = {
  name: string
  total: number
  items: Record<string, unknown>[]
  note: string
}

export const api = {
  login: (username: string, password: string) =>
    adminFetch<{ session: string }>('/api/v1/admin/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    }),
  logout: () => adminFetch('/api/v1/admin/logout', { method: 'POST' }),
  account: () => adminFetch<{ username: string }>('/api/v1/admin/account'),
  password: (oldPassword: string, newPassword: string) =>
    adminFetch('/api/v1/admin/password', {
      method: 'POST',
      body: JSON.stringify({ oldPassword, newPassword }),
    }),
  health: () => adminFetch<AdminHealth>('/api/v1/admin/health'),
  healthz: async () => {
    const res = await fetch('/healthz')
    const body = (await res.json()) as Envelope<{ status?: string; phase?: string; version?: string }>
    return { status: body.data?.status ?? '?', phase: body.data?.phase ?? '?', version: body.data?.version ?? '' }
  },
  bindings: () => adminFetch<{ items: BindingRow[] }>('/api/v1/admin/bindings'),
  binding: (id: string) => adminFetch<BindingDetail>('/api/v1/admin/bindings/' + encodeURIComponent(id)),
  disable: (id: string) =>
    adminFetch('/api/v1/admin/bindings/' + encodeURIComponent(id) + '/disable', { method: 'POST' }),
  enable: (id: string) =>
    adminFetch('/api/v1/admin/bindings/' + encodeURIComponent(id) + '/enable', { method: 'POST' }),
  sync: (id: string) =>
    adminFetch('/api/v1/admin/bindings/' + encodeURIComponent(id) + '/sync', { method: 'POST' }),
  kick: (id: string) =>
    adminFetch('/api/v1/admin/bindings/' + encodeURIComponent(id) + '/kick', { method: 'POST' }),
  jobs: (q: { bindingId?: string; status?: string; page?: number; pageSize?: number }) => {
    const p = new URLSearchParams()
    if (q.bindingId) p.set('bindingId', q.bindingId)
    if (q.status) p.set('status', q.status)
    if (q.page) p.set('page', String(q.page))
    if (q.pageSize) p.set('pageSize', String(q.pageSize))
    const qs = p.toString()
    return adminFetch<{ total: number; items: JobRow[] }>('/api/v1/admin/jobs' + (qs ? '?' + qs : ''))
  },
  tables: () => adminFetch<{ items: TableMeta[] }>('/api/v1/admin/tables'),
  tableRows: (name: string, q: { bindingId?: string; page?: number; pageSize?: number }) => {
    const p = new URLSearchParams()
    if (q.bindingId) p.set('bindingId', q.bindingId)
    if (q.page) p.set('page', String(q.page))
    if (q.pageSize) p.set('pageSize', String(q.pageSize))
    const qs = p.toString()
    return adminFetch<TablePage>('/api/v1/admin/tables/' + encodeURIComponent(name) + (qs ? '?' + qs : ''))
  },
  settings: () => adminFetch<Settings>('/api/v1/admin/settings'),
  saveSettings: (body: Settings) =>
    adminFetch<Settings>('/api/v1/admin/settings', { method: 'PUT', body: JSON.stringify(body) }),
  system: (refresh = false) =>
    adminFetch<SystemInfo>('/api/v1/admin/system' + (refresh ? '?refresh=1' : '')),
  systemUpdate: () =>
    adminFetch<{ target: string; status: string }>('/api/v1/admin/system/update', { method: 'POST' }),
}

export type SystemInfo = {
  version: string
  commit?: string
  latest: string
  latestUrl?: string
  updateAvailable: boolean
  inDocker: boolean
  dockerAvailable: boolean
  canApply: boolean
  updating: boolean
  target?: string
  image?: string
  hint: string
}
