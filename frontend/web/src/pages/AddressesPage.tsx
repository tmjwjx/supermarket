import { useEffect, useState, type FormEvent } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import {
  createAddress,
  deleteAddress,
  listAddresses,
  readAddresses,
  setDefaultAddress,
  updateAddress,
  type Address,
} from '../api/address.ts'
import { sameOriginPath } from '../session/index.ts'

type Draft = { receiver: string; phone: string; province: string; city: string; district: string; detail: string }

const blank: Draft = { receiver: '', phone: '', province: '', city: '', district: '', detail: '' }

const fields: { key: keyof Draft; label: string }[] = [
  { key: 'receiver', label: '收货人' },
  { key: 'phone', label: '手机号' },
  { key: 'province', label: '省' },
  { key: 'city', label: '市' },
  { key: 'district', label: '区' },
  { key: 'detail', label: '详细地址' },
]

export default function AddressesPage() {
  const [params] = useSearchParams()
  const back = backPath(params.get('from'))
  const [items, setItems] = useState<Address[]>([])
  const [ready, setReady] = useState(false)
  const [editing, setEditing] = useState<string | null>(null)
  const [draft, setDraft] = useState<Draft>(blank)
  const [message, setMessage] = useState('')
  const [failed, setFailed] = useState(false)

  async function reload() {
    setItems(readAddresses(await listAddresses()))
  }

  useEffect(() => {
    let gone = false
    listAddresses()
      .then((body) => {
        if (!gone) setItems(readAddresses(body))
      })
      .catch((err: unknown) => {
        if (gone) return
        setFailed(true)
        setMessage(err instanceof Error ? err.message : '地址加载失败')
      })
      .finally(() => {
        if (!gone) setReady(true)
      })
    return () => {
      gone = true
    }
  }, [])

  async function run(task: () => Promise<unknown>, done: string, fail: string) {
    setFailed(false)
    setMessage('')
    try {
      await task()
      await reload()
      setMessage(done)
      return true
    } catch (err) {
      setFailed(true)
      setMessage(err instanceof Error ? err.message : fail)
      return false
    }
  }

  function startCreate() {
    setEditing('')
    setDraft(blank)
  }

  function startEdit(item: Address) {
    setEditing(item.id)
    setDraft({
      receiver: item.receiver,
      phone: item.phone,
      province: item.province,
      city: item.city,
      district: item.district,
      detail: item.detail,
    })
  }

  async function onSubmit(event: FormEvent) {
    event.preventDefault()
    if (editing === null) return
    const input = Object.fromEntries(Object.entries(draft).map(([key, value]) => [key, value.trim()])) as Draft
    if (!/^1\d{10}$/.test(input.phone)) {
      setFailed(true)
      setMessage('手机号得是 11 位')
      return
    }
    const ok = editing
      ? await run(() => updateAddress(editing, input), '地址已修改', '地址没改成')
      : await run(() => createAddress(input), '地址已添加', '地址没存上')
    if (ok) {
      setEditing(null)
      setDraft(blank)
    }
  }

  function onDelete(item: Address) {
    if (!window.confirm(`删除 ${item.receiver} 的这条地址`)) return
    void run(() => deleteAddress(item.id), '地址已删除', '地址没删掉').then((ok) => {
      if (ok && editing === item.id) setEditing(null)
    })
  }

  return (
    <section className="slip">
      <h1>收货地址</h1>
      <p className="lead">结算时从这里挑一个</p>
      {back ? (
        <p className="switch">
          <Link to={back}>回去结算</Link>
        </p>
      ) : null}
      {ready && items.length === 0 && !message ? <p className="note">还没有收货地址</p> : null}
      <ul className="skus">
        {items.map((item) => (
          <li key={item.id}>
            <strong>
              {item.receiver} {item.phone}
              {item.isDefault ? <span className="tag">默认</span> : null}
            </strong>
            <span>
              {item.province}
              {item.city}
              {item.district}
              {item.detail}
            </span>
            <div className="actions">
              <button className="quiet" type="button" onClick={() => startEdit(item)}>
                修改
              </button>
              <button className="quiet" type="button" onClick={() => onDelete(item)}>
                删除
              </button>
              {item.isDefault ? null : (
                <button
                  className="quiet"
                  type="button"
                  onClick={() => void run(() => setDefaultAddress(item.id), '已设为默认', '没设成默认')}
                >
                  设为默认
                </button>
              )}
            </div>
          </li>
        ))}
      </ul>
      {editing === null ? (
        <button className="primary" type="button" onClick={startCreate}>
          新增地址
        </button>
      ) : (
        <form onSubmit={(event) => void onSubmit(event)}>
          <h2>{editing ? '修改地址' : '新增地址'}</h2>
          {fields.map((field) => (
            <label key={field.key}>
              {field.label}
              <input
                value={draft[field.key]}
                onChange={(event) => setDraft((current) => ({ ...current, [field.key]: event.target.value }))}
                inputMode={field.key === 'phone' ? 'numeric' : undefined}
                required
              />
            </label>
          ))}
          <div className="actions">
            <button className="primary" type="submit">
              保存地址
            </button>
            <button className="quiet" type="button" onClick={() => setEditing(null)}>
              取消
            </button>
          </div>
        </form>
      )}
      {message ? <p className={failed ? 'note bad' : 'note'}>{message}</p> : null}
    </section>
  )
}

// 只接受同源站内路径 防止 from 被拿去跳外站 拒绝时不给回去的链接
function backPath(raw: string | null): string {
  return sameOriginPath(raw)
}
