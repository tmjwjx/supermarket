import { send } from './client.ts'
import { readBool, readList, readNumber, readString } from './read.ts'

export type Notice = {
  id: string
  title: string
  body: string
  orderId: string
  read: boolean
}

export function listNotifications() {
  return send('/v1/notifications', {})
}

export function markNotificationsRead(ids: string[]) {
  return send('/v1/notifications:read', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ ids }),
  })
}

export function readNotifications(body: unknown): { items: Notice[]; unread: number } {
  const items = readList(body, 'notifications').flatMap((row) => {
    const id = readString(row, 'id')
    if (!id) return []
    return [
      {
        id,
        title: readString(row, 'title'),
        body: readString(row, 'body'),
        orderId: readString(row, 'order_id', 'orderId'),
        read: readBool(row, 'read'),
      },
    ]
  })
  const record = body && typeof body === 'object' ? (body as Record<string, unknown>) : {}
  return { items, unread: readNumber(record, 'unread') }
}
