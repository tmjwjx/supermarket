import assert from 'node:assert/strict'
import { test } from 'node:test'
import { safeFrom } from '../src/session/index.ts'

Object.defineProperty(globalThis, 'window', { value: { location: { origin: 'http://localhost:5173' } } })

test('safeFrom 只放行同源站内路径并保留查询和锚点', () => {
  assert.equal(safeFrom('/orders?status=2#list'), '/orders?status=2#list')
  assert.equal(safeFrom('http://localhost:5173/brands'), '/brands')
})

test('safeFrom 拒绝外站 反斜杠 控制字符和登录页', () => {
  for (const raw of [
    null,
    '',
    '//evil.example/x',
    'https://evil.example/x',
    '/\\evil.example',
    '/\t/evil.example',
    '/\r/evil.example',
    'javascript:alert(1)',
    '/login',
  ]) {
    assert.equal(safeFrom(raw), '/products', String(raw))
  }
})
