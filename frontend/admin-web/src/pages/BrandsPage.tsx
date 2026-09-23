import { useState, type FormEvent } from 'react'
import { createBrand, deleteBrand, listBrands, updateBrand, type Brand, type BrandInput } from '../api/admin.ts'
import { useLoad, useNote } from './shared.ts'
import Notice from './Notice.tsx'

type Draft = { name: string; initial: string; logoUrl: string; description: string; visible: boolean; sort: string }

function toDraft(b?: Brand): Draft {
  return {
    name: b?.name ?? '',
    initial: b?.initial ?? '',
    logoUrl: b?.logoUrl ?? '',
    description: b?.description ?? '',
    visible: b?.visible ?? true,
    sort: String(b?.sort ?? 0),
  }
}

function toInput(d: Draft): BrandInput {
  return {
    name: d.name.trim(),
    initial: d.initial.trim().toUpperCase(),
    logo_url: d.logoUrl.trim(),
    description: d.description.trim(),
    visible: d.visible,
    sort: Number(d.sort) || 0,
  }
}

function BrandFields({ draft, onChange }: { draft: Draft; onChange: (next: Draft) => void }) {
  return (
    <div className="row">
      <label>名称<input value={draft.name} onChange={(e) => onChange({ ...draft, name: e.target.value })} required /></label>
      <label>首字母<input value={draft.initial} maxLength={1} onChange={(e) => onChange({ ...draft, initial: e.target.value })} /></label>
      <label>Logo 地址<input value={draft.logoUrl} onChange={(e) => onChange({ ...draft, logoUrl: e.target.value })} /></label>
      <label>描述<input value={draft.description} onChange={(e) => onChange({ ...draft, description: e.target.value })} /></label>
      <label>排序<input value={draft.sort} onChange={(e) => onChange({ ...draft, sort: e.target.value })} inputMode="numeric" /></label>
      <label className="check"><input type="checkbox" checked={draft.visible} onChange={(e) => onChange({ ...draft, visible: e.target.checked })} />显示</label>
    </div>
  )
}

export default function BrandsPage() {
  const brands = useLoad<Brand[]>(listBrands, [])
  const [draft, setDraft] = useState<Draft>(toDraft())
  const [editing, setEditing] = useState('')
  const [edit, setEdit] = useState<Draft>(toDraft())
  const { note, ok, fail } = useNote()

  async function run(action: () => Promise<unknown>, done: string) {
    try {
      await action()
      ok(done)
      brands.reload()
      return true
    } catch (err) {
      fail(err, '操作失败')
      return false
    }
  }

  async function onCreate(event: FormEvent) {
    event.preventDefault()
    if (await run(() => createBrand(toInput(draft)), '品牌已创建')) setDraft(toDraft())
  }

  const list = [...brands.data].sort((a, b) => a.sort - b.sort)

  return (
    <section>
      <h1>品牌</h1>
      <Notice text={brands.error} tone="error" />
      <Notice {...note} />
      <form onSubmit={(event) => void onCreate(event)}>
        <BrandFields draft={draft} onChange={setDraft} />
        <button type="submit">新建品牌</button>
      </form>
      <table>
        <thead><tr><th>名称</th><th>首字母</th><th>描述</th><th>排序</th><th>显示</th><th>操作</th></tr></thead>
        <tbody>
          {list.map((b) => editing === b.id ? (
            <tr key={b.id}>
              <td colSpan={6}>
                <BrandFields draft={edit} onChange={setEdit} />
                <div className="actions">
                  <button type="button" onClick={() => void run(() => updateBrand(b.id, toInput(edit)), '品牌已修改').then((done) => done && setEditing(''))}>保存</button>
                  <button type="button" onClick={() => setEditing('')}>取消</button>
                </div>
              </td>
            </tr>
          ) : (
            <tr key={b.id} className={b.visible ? '' : 'muted'}>
              <td>
                {b.logoUrl ? <img src={b.logoUrl} alt="" width={24} height={24} style={{ verticalAlign: 'middle', marginRight: 6 }} /> : null}
                {b.name}
                <div className="hint">{b.id}</div>
              </td>
              <td>{b.initial || '-'}</td>
              <td>{b.description || '-'}</td>
              <td>{b.sort}</td>
              <td><span className={`tag ${b.visible ? 'on' : 'off'}`}>{b.visible ? '显示' : '隐藏'}</span></td>
              <td>
                <div className="actions">
                  <button type="button" onClick={() => { setEditing(b.id); setEdit(toDraft(b)) }}>修改</button>
                  <button type="button" onClick={() => void run(() => updateBrand(b.id, toInput({ ...toDraft(b), visible: !b.visible })), b.visible ? '已隐藏' : '已显示')}>
                    {b.visible ? '隐藏' : '显示'}
                  </button>
                  <button
                    type="button"
                    className="danger"
                    onClick={() => {
                      if (window.confirm(`删除品牌 ${b.name}`)) void run(() => deleteBrand(b.id), '品牌已删除')
                    }}
                  >
                    删除
                  </button>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      {!brands.loading && list.length === 0 ? <p>还没有品牌</p> : null}
    </section>
  )
}
