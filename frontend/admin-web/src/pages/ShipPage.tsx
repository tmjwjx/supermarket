import { useState, type FormEvent } from 'react'
import { useSearchParams } from 'react-router-dom'
import { shipOrder } from '../api/admin.ts'
import { useNote } from './shared.ts'
import Notice from './Notice.tsx'

export default function ShipPage() {
  const [params] = useSearchParams()
  const [orderId, setOrderId] = useState(params.get('id') ?? '')
  const [busy, setBusy] = useState(false)
  const { note, ok, fail, clear } = useNote()

  async function onSubmit(event: FormEvent) {
    event.preventDefault()
    clear()
    const id = orderId.trim()
    if (!id) return fail(null, '请填写订单 id')
    setBusy(true)
    try {
      await shipOrder(id)
      ok('已发货')
    } catch (err) {
      fail(err, '发货失败')
    } finally {
      setBusy(false)
    }
  }

  return (
    <form onSubmit={(event) => void onSubmit(event)}>
      <h1>发货</h1>
      <p className="hint">只有已支付的订单能发货</p>
      <label>
        订单 id
        <input value={orderId} onChange={(event) => setOrderId(event.target.value)} />
      </label>
      <button type="submit" disabled={busy}>发货</button>
      <Notice {...note} />
    </form>
  )
}
