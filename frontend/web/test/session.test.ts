import assert from 'node:assert/strict'
import { test } from 'node:test'
import { safeFrom, sameOriginPath } from '../src/session/index.ts'

Object.defineProperty(globalThis, 'window', { value: { location: { origin: 'http://localhost:5174' } } })

test('safeFrom 只放行同源站内路径并保留查询和锚点', () => {
  assert.equal(safeFrom('/orders/1?tab=pay#top'), '/orders/1?tab=pay#top')
  assert.equal(safeFrom('http://localhost:5174/cart'), '/cart')
  assert.equal(safeFrom('/a/../cart'), '/cart')
})

test('safeFrom 拒绝外站 反斜杠 控制字符和登录注册页', () => {
  for (const raw of [
    null,
    '',
    '//evil.example/x',
    'https://evil.example/x',
    '/\\evil.example',
    '\\\\evil.example',
    '/\t/evil.example',
    '/\n/evil.example',
    'javascript:alert(1)',
    'http://localhost:5174.evil.example/',
    '/login?from=/me',
    '/register',
  ]) {
    assert.equal(safeFrom(raw), '/me', String(raw))
  }
})

test('sameOriginPath 拒绝时返回空串 地址页据此不给回去的链接', () => {
  assert.equal(sameOriginPath('//evil.example'), '')
  assert.equal(sameOriginPath('/\\evil.example'), '')
  assert.equal(sameOriginPath('/cart?step=2'), '/cart?step=2')
})
