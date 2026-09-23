import { useState, type FormEvent } from 'react'
import { adminRoleText, createAdminUser, listAdminUsers, resetAdminPassword, updateAdminUser, type AdminUser } from '../api/admin.ts'
import { useLoad, useNote } from './shared.ts'
import Notice from './Notice.tsx'

const active = 1
const disabled = 2

export default function AdminUsersPage() {
  const users = useLoad<AdminUser[]>(listAdminUsers, [])
  const [form, setForm] = useState({ username: '', password: '', displayName: '', role: 'product' })
  const [passwords, setPasswords] = useState<Record<string, string>>({})
  const [busy, setBusy] = useState(false)
  const { note, ok, fail } = useNote()

  async function run(action: () => Promise<unknown>, done: string) {
    setBusy(true)
    try {
      await action()
      ok(done)
      users.reload()
      return true
    } catch (err) {
      fail(err, '操作失败')
      return false
    } finally {
      setBusy(false)
    }
  }

  async function onCreate(event: FormEvent) {
    event.preventDefault()
    const input = { username: form.username.trim(), password: form.password, display_name: form.displayName.trim(), role: form.role }
    if (await run(() => createAdminUser(input), `账号 ${input.username} 已创建`)) {
      setForm({ username: '', password: '', displayName: '', role: form.role })
    }
  }

  function roleSelect(value: string, onChange: (role: string) => void, off = false) {
    return (
      <select value={value} disabled={off} onChange={(e) => onChange(e.target.value)}>
        {Object.entries(adminRoleText).map(([role, text]) => <option key={role} value={role}>{text}</option>)}
      </select>
    )
  }

  return (
    <section>
      <h1>运营账号</h1>
      <p className="hint">只有超级管理员能管理账号 商品运营管商品和目录 订单运营管订单和对账</p>
      <Notice text={users.error} tone="error" />
      <Notice {...note} />
      <form onSubmit={(event) => void onCreate(event)}>
        <div className="row">
          <label>用户名<input value={form.username} onChange={(e) => setForm({ ...form, username: e.target.value })} required autoComplete="off" /></label>
          <label>初始密码<input type="password" value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} required autoComplete="new-password" /></label>
          <label>显示名<input value={form.displayName} onChange={(e) => setForm({ ...form, displayName: e.target.value })} /></label>
          <label>角色{roleSelect(form.role, (role) => setForm({ ...form, role }))}</label>
          <button type="submit" disabled={busy}>新建账号</button>
        </div>
      </form>
      <table>
        <thead><tr><th>用户名</th><th>显示名</th><th>角色</th><th>状态</th><th>操作</th></tr></thead>
        <tbody>
          {users.data.map((u) => (
            <tr key={u.id} className={u.status === disabled ? 'muted' : ''}>
              <td>{u.username}</td>
              <td>{u.displayName || '-'}</td>
              <td>{roleSelect(u.role, (role) => void run(() => updateAdminUser(u.id, role, 0), `${u.username} 角色已改`), busy)}</td>
              <td><span className={`tag ${u.status === active ? 'on' : 'off'}`}>{u.status === active ? '启用' : '停用'}</span></td>
              <td>
                <div className="actions">
                  <button
                    type="button"
                    disabled={busy}
                    onClick={() => void run(() => updateAdminUser(u.id, '', u.status === active ? disabled : active), u.status === active ? `${u.username} 已停用` : `${u.username} 已启用`)}
                  >
                    {u.status === active ? '停用' : '启用'}
                  </button>
                  <input
                    type="password"
                    value={passwords[u.id] ?? ''}
                    onChange={(e) => setPasswords((prev) => ({ ...prev, [u.id]: e.target.value }))}
                    placeholder="新密码"
                    autoComplete="new-password"
                  />
                  <button
                    type="button"
                    disabled={busy || !(passwords[u.id] ?? '')}
                    onClick={() =>
                      void run(() => resetAdminPassword(u.id, passwords[u.id] ?? ''), `${u.username} 密码已重置`).then((done) => {
                        if (done) setPasswords((prev) => ({ ...prev, [u.id]: '' }))
                      })
                    }
                  >
                    重置密码
                  </button>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      {!users.loading && users.data.length === 0 && !users.error ? <p>没有账号</p> : null}
    </section>
  )
}
