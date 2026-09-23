import { useState, type FormEvent } from 'react'
import {
  AttributeKind,
  createAttribute,
  createTemplate,
  deleteAttribute,
  deleteTemplate,
  listAttributes,
  listTemplates,
  updateAttribute,
  type Attribute,
  type AttributeInput,
  type Template,
} from '../api/admin.ts'
import { useLoad, useNote } from './shared.ts'
import Notice from './Notice.tsx'

const kindText: Record<number, string> = { [AttributeKind.Spec]: '规格', [AttributeKind.Param]: '参数' }

// 可选值按逗号或换行拆开 去空去重
function splitOptions(text: string): string[] {
  const out: string[] = []
  for (const part of text.split(/[,，\n]/)) {
    const value = part.trim()
    if (value && !out.includes(value)) out.push(value)
  }
  return out
}

function AttributeForm({
  initial,
  submitText,
  onSubmit,
  onCancel,
}: {
  initial?: Attribute
  submitText: string
  onSubmit: (input: AttributeInput) => Promise<boolean>
  onCancel?: () => void
}) {
  const [name, setName] = useState(initial?.name ?? '')
  const [kind, setKind] = useState<number>(initial?.kind ?? AttributeKind.Spec)
  const [options, setOptions] = useState(initial?.options.join('\n') ?? '')
  const [allowCustom, setAllowCustom] = useState(initial?.allowCustom ?? false)
  const [sort, setSort] = useState(String(initial?.sort ?? 0))

  async function submit(event: FormEvent) {
    event.preventDefault()
    const done = await onSubmit({
      name: name.trim(),
      kind,
      options: splitOptions(options),
      allow_custom: allowCustom,
      sort: Number(sort) || 0,
    })
    if (done && !initial) {
      setName('')
      setOptions('')
      setAllowCustom(false)
      setSort('0')
    }
  }

  return (
    <form onSubmit={(event) => void submit(event)}>
      <div className="row">
        <label>属性名<input value={name} onChange={(e) => setName(e.target.value)} required /></label>
        <label>
          类型
          <select value={kind} onChange={(e) => setKind(Number(e.target.value))}>
            <option value={AttributeKind.Spec}>规格 决定 SKU</option>
            <option value={AttributeKind.Param}>参数 展示用</option>
          </select>
        </label>
        <label>排序<input value={sort} onChange={(e) => setSort(e.target.value)} inputMode="numeric" /></label>
        <label className="check"><input type="checkbox" checked={allowCustom} onChange={(e) => setAllowCustom(e.target.checked)} />允许手填</label>
      </div>
      <label>
        可选值 每行一个或用逗号分隔
        <textarea value={options} onChange={(e) => setOptions(e.target.value)} style={{ minHeight: '4rem' }} />
      </label>
      <div className="actions">
        <button type="submit">{submitText}</button>
        {onCancel ? <button type="button" onClick={onCancel}>取消</button> : null}
      </div>
    </form>
  )
}

export default function AttributesPage() {
  const templates = useLoad<Template[]>(listTemplates, [])
  const [current, setCurrent] = useState('')
  const attrs = useLoad<Attribute[]>(() => (current ? listAttributes(current) : Promise.resolve([])), [], [current])
  const [editing, setEditing] = useState('')
  const [templateName, setTemplateName] = useState('')
  const { note, ok, fail } = useNote()

  async function run(action: () => Promise<unknown>, done: string): Promise<boolean> {
    try {
      await action()
      ok(done)
      return true
    } catch (err) {
      fail(err, '操作失败')
      return false
    }
  }

  const currentTemplate = templates.data.find((t) => t.id === current)

  return (
    <section>
      <h1>属性模板</h1>
      <Notice text={templates.error} tone="error" />
      <Notice {...note} />
      <form
        className="row"
        onSubmit={(event) => {
          event.preventDefault()
          void run(() => createTemplate(templateName.trim()), '模板已创建').then((done) => {
            if (done) {
              setTemplateName('')
              templates.reload()
            }
          })
        }}
      >
        <label>新模板名<input value={templateName} onChange={(e) => setTemplateName(e.target.value)} required /></label>
        <button type="submit">新建模板</button>
      </form>
      <table>
        <thead><tr><th>模板</th><th>操作</th></tr></thead>
        <tbody>
          {templates.data.map((t) => (
            <tr key={t.id} className={t.id === current ? 'current' : ''}>
              <td>{t.id === current ? <strong>{t.name}</strong> : t.name}<div className="hint">{t.id}</div></td>
              <td>
                <div className="actions">
                  <button type="button" onClick={() => { setCurrent(t.id); setEditing('') }}>管理属性</button>
                  <button
                    type="button"
                    className="danger"
                    onClick={() => {
                      if (!window.confirm(`删除模板 ${t.name}`)) return
                      void run(() => deleteTemplate(t.id), '模板已删除').then((done) => {
                        if (!done) return
                        if (current === t.id) setCurrent('')
                        templates.reload()
                      })
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

      {current ? (
        <>
          <h2>{currentTemplate?.name ?? current} 的属性</h2>
          <Notice text={attrs.error} tone="error" />
          <table>
            <thead>
              <tr><th>名称</th><th>类型</th><th>可选值</th><th>允许手填</th><th>排序</th><th>操作</th></tr>
            </thead>
            <tbody>
              {attrs.data.map((a) => editing === a.id ? (
                <tr key={a.id}>
                  <td colSpan={6}>
                    <AttributeForm
                      initial={a}
                      submitText="保存修改"
                      onCancel={() => setEditing('')}
                      onSubmit={async (input) => {
                        const done = await run(() => updateAttribute(a.id, input), '属性已修改')
                        if (done) {
                          setEditing('')
                          attrs.reload()
                        }
                        return done
                      }}
                    />
                  </td>
                </tr>
              ) : (
                <tr key={a.id}>
                  <td>{a.name}</td>
                  <td>{kindText[a.kind] ?? a.kind}</td>
                  <td>{a.options.join(' / ') || '-'}</td>
                  <td>{a.allowCustom ? '是' : '否'}</td>
                  <td>{a.sort}</td>
                  <td>
                    <div className="actions">
                      <button type="button" onClick={() => setEditing(a.id)}>修改</button>
                      <button
                        type="button"
                        className="danger"
                        onClick={() => {
                          if (!window.confirm(`删除属性 ${a.name}`)) return
                          void run(() => deleteAttribute(a.id), '属性已删除').then((done) => done && attrs.reload())
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
          {!attrs.loading && attrs.data.length === 0 ? <p>还没有属性</p> : null}
          <h2>新建属性</h2>
          <AttributeForm
            submitText="新建属性"
            onSubmit={async (input) => {
              const done = await run(() => createAttribute(current, input), '属性已创建')
              if (done) attrs.reload()
              return done
            }}
          />
        </>
      ) : (
        <p className="hint">选择一个模板管理它的属性</p>
      )}
    </section>
  )
}
