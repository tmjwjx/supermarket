import { useState, type FormEvent } from 'react'
import {
  createCategory,
  deleteCategory,
  listCategories,
  listTemplates,
  updateCategory,
  type Category,
  type CategoryInput,
  type Template,
} from '../api/admin.ts'
import { useLoad, useNote } from './shared.ts'
import Notice from './Notice.tsx'

type Draft = { name: string; parentId: string; sort: string; visible: boolean; templateId: string; iconUrl: string }

function toDraft(c?: Category): Draft {
  return {
    name: c?.name ?? '',
    parentId: c?.parentId ?? '',
    sort: String(c?.sort ?? 0),
    visible: c?.visible ?? true,
    templateId: c?.templateId ?? '',
    iconUrl: c?.iconUrl ?? '',
  }
}

function toPatch(d: Draft): Omit<CategoryInput, 'parent_id'> {
  return {
    name: d.name.trim(),
    icon_url: d.iconUrl.trim(),
    sort: Number(d.sort) || 0,
    visible: d.visible,
    attribute_template_id: d.templateId,
  }
}

function toInput(d: Draft): CategoryInput {
  return { ...toPatch(d), parent_id: d.parentId }
}

export default function CategoriesPage() {
  const cats = useLoad<Category[]>(listCategories, [])
  const templates = useLoad<Template[]>(listTemplates, [])
  const [draft, setDraft] = useState<Draft>(toDraft())
  const [editing, setEditing] = useState('')
  const [edit, setEdit] = useState<Draft>(toDraft())
  const { note, ok, fail } = useNote()

  const bySort = (a: Category, b: Category) => a.sort - b.sort
  const parents = cats.data.filter((c) => !c.parentId).sort(bySort)
  const parentIds = new Set(parents.map((p) => p.id))
  const orphans = cats.data.filter((c) => c.parentId && !parentIds.has(c.parentId))
  const templateName = (id: string) => templates.data.find((t) => t.id === id)?.name ?? (id ? id : '-')

  async function run(action: () => Promise<unknown>, done: string) {
    try {
      await action()
      ok(done)
      cats.reload()
      return true
    } catch (err) {
      fail(err, '操作失败')
      return false
    }
  }

  async function onCreate(event: FormEvent) {
    event.preventDefault()
    if (await run(() => createCategory(toInput(draft)), '分类已创建')) setDraft(toDraft())
  }

  function templateSelect(value: string, onChange: (value: string) => void) {
    return (
      <select value={value} onChange={(e) => onChange(e.target.value)}>
        <option value="">不绑定</option>
        {templates.data.map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}
      </select>
    )
  }

  function line(c: Category, isChild: boolean) {
    if (editing === c.id) {
      return (
        <tr key={c.id} className={isChild ? 'child' : ''}>
          <td><input value={edit.name} onChange={(e) => setEdit({ ...edit, name: e.target.value })} /></td>
          <td><input value={edit.sort} onChange={(e) => setEdit({ ...edit, sort: e.target.value })} inputMode="numeric" /></td>
          <td><input type="checkbox" checked={edit.visible} onChange={(e) => setEdit({ ...edit, visible: e.target.checked })} /></td>
          <td>{templateSelect(edit.templateId, (templateId) => setEdit({ ...edit, templateId }))}</td>
          <td>
            <div className="actions">
              <button
                type="button"
                onClick={() => void run(() => updateCategory(c.id, toPatch(edit)), '分类已修改').then((done) => done && setEditing(''))}
              >
                保存
              </button>
              <button type="button" onClick={() => setEditing('')}>取消</button>
            </div>
          </td>
        </tr>
      )
    }
    return (
      <tr key={c.id} className={`${isChild ? 'child' : ''} ${c.visible ? '' : 'muted'}`}>
        <td>{c.name}<div className="hint">{c.id}</div></td>
        <td>{c.sort}</td>
        <td><span className={`tag ${c.visible ? 'on' : 'off'}`}>{c.visible ? '显示' : '隐藏'}</span></td>
        <td>{templateName(c.templateId)}</td>
        <td>
          <div className="actions">
            <button type="button" onClick={() => { setEditing(c.id); setEdit(toDraft(c)) }}>修改</button>
            <button
              type="button"
              onClick={() => void run(() => updateCategory(c.id, toPatch({ ...toDraft(c), visible: !c.visible })), c.visible ? '已隐藏' : '已显示')}
            >
              {c.visible ? '隐藏' : '显示'}
            </button>
            <button
              type="button"
              className="danger"
              onClick={() => {
                if (window.confirm(`删除分类 ${c.name}`)) void run(() => deleteCategory(c.id), '分类已删除')
              }}
            >
              删除
            </button>
          </div>
        </td>
      </tr>
    )
  }

  return (
    <section>
      <h1>分类</h1>
      <p className="hint">两级分类 商品只能挂在二级分类 灰色为隐藏</p>
      <Notice text={cats.error || templates.error} tone="error" />
      <Notice {...note} />
      <form onSubmit={(event) => void onCreate(event)}>
        <div className="row">
          <label>名称<input value={draft.name} onChange={(e) => setDraft({ ...draft, name: e.target.value })} required /></label>
          <label>
            父级
            <select value={draft.parentId} onChange={(e) => setDraft({ ...draft, parentId: e.target.value })}>
              <option value="">无 作为一级分类</option>
              {parents.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}
            </select>
          </label>
          <label>排序<input value={draft.sort} onChange={(e) => setDraft({ ...draft, sort: e.target.value })} inputMode="numeric" /></label>
          <label>绑定模板{templateSelect(draft.templateId, (templateId) => setDraft({ ...draft, templateId }))}</label>
          <label className="check"><input type="checkbox" checked={draft.visible} onChange={(e) => setDraft({ ...draft, visible: e.target.checked })} />显示</label>
          <button type="submit">新建分类</button>
        </div>
      </form>
      <table>
        <thead><tr><th>名称</th><th>排序</th><th>显示</th><th>模板</th><th>操作</th></tr></thead>
        <tbody>
          {parents.flatMap((p) => [line(p, false), ...cats.data.filter((c) => c.parentId === p.id).sort(bySort).map((c) => line(c, true))])}
          {orphans.map((c) => line(c, true))}
        </tbody>
      </table>
      {!cats.loading && cats.data.length === 0 ? <p>还没有分类</p> : null}
    </section>
  )
}
