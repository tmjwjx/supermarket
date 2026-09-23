import { send } from './client.ts'

export type Profile = {
  id: string
  phone: string
  nickname: string
  avatarUrl: string
  status: string
  createdAt: string
}

export function readAccessToken(body: unknown): string {
  if (!body || typeof body !== 'object') return ''
  const record = body as { access_token?: unknown; accessToken?: unknown }
  const token = record.access_token ?? record.accessToken
  return typeof token === 'string' ? token : ''
}

export function readUserId(body: unknown): string {
  return readProfile(body)?.id ?? ''
}

export function readProfile(body: unknown): Profile | null {
  const user = unwrapUser(body)
  if (!user) return null
  const profile: Profile = {
    id: readString(user, 'id'),
    phone: readString(user, 'phone'),
    nickname: readString(user, 'nickname'),
    avatarUrl: readString(user, 'avatar_url', 'avatarUrl'),
    status: statusLabel(user.status),
    createdAt: readTime(user.created_at ?? user.createdAt),
  }
  if (!profile.id && !profile.phone && !profile.nickname) return null
  return profile
}

export function register(input: { phone: string; password: string; nickname: string }) {
  const payload: Record<string, string> = {
    phone: input.phone,
    password: input.password,
  }
  if (input.nickname) payload.nickname = input.nickname
  return send('/v1/users/register', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
}

export function login(input: { phone: string; password: string }) {
  return send('/v1/users/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      phone: input.phone,
      password: input.password,
    }),
  })
}

export function getMe() {
  return send('/v1/users/me', {})
}

export function getUser(id: string) {
  return send(`/v1/users/${encodeURIComponent(id)}`, {})
}

function unwrapUser(body: unknown): Record<string, unknown> | null {
  if (!body || typeof body !== 'object') return null
  const record = body as Record<string, unknown>
  const nested = record.user
  if (nested && typeof nested === 'object') return nested as Record<string, unknown>
  if (typeof record.id === 'string' || typeof record.phone === 'string') return record
  return null
}

function readString(record: Record<string, unknown>, ...keys: string[]): string {
  for (const key of keys) {
    const value = record[key]
    if (typeof value === 'string' && value !== '') return value
  }
  return ''
}

function statusLabel(value: unknown): string {
  if (value === 1 || value === 'USER_STATUS_ACTIVE') return '正常'
  if (value === 2 || value === 'USER_STATUS_DISABLED') return '已停用'
  return ''
}

function readTime(value: unknown): string {
  if (typeof value === 'string') return value
  if (!value || typeof value !== 'object' || !('seconds' in value)) return ''
  const seconds = (value as { seconds?: unknown }).seconds
  const n = typeof seconds === 'number' ? seconds : typeof seconds === 'string' ? Number(seconds) : NaN
  if (!Number.isFinite(n)) return ''
  return new Date(n * 1000).toLocaleString('zh-CN')
}
