import { send } from './client.ts'
import { readBool, readList, readString } from './read.ts'

export type Address = {
  id: string
  receiver: string
  phone: string
  province: string
  city: string
  district: string
  detail: string
  isDefault: boolean
}

export function listAddresses() {
  return send('/v1/addresses', {})
}

export function createAddress(input: {
  receiver: string
  phone: string
  province: string
  city: string
  district: string
  detail: string
}) {
  return send('/v1/addresses', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
}

export function updateAddress(id: string, input: {
  receiver: string
  phone: string
  province: string
  city: string
  district: string
  detail: string
}) {
  return send(`/v1/addresses/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
}

export function deleteAddress(id: string) {
  return send(`/v1/addresses/${encodeURIComponent(id)}`, { method: 'DELETE' })
}

export function setDefaultAddress(id: string) {
  return send(`/v1/addresses/${encodeURIComponent(id)}:default`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: '{}',
  })
}

export function readAddresses(body: unknown): Address[] {
  return readList(body, 'addresses').flatMap((row) => {
    const id = readString(row, 'id')
    if (!id) return []
    return [
      {
        id,
        receiver: readString(row, 'receiver'),
        phone: readString(row, 'phone'),
        province: readString(row, 'province'),
        city: readString(row, 'city'),
        district: readString(row, 'district'),
        detail: readString(row, 'detail'),
        isDefault: readBool(row, 'is_default', 'isDefault'),
      },
    ]
  })
}
