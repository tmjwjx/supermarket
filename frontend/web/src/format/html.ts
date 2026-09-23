// 图文详情白名单清洗 丢掉脚本和事件属性后才允许插入页面
const keepTags = new Set([
  'a', 'b', 'blockquote', 'br', 'caption', 'code', 'dd', 'del', 'div', 'dl', 'dt', 'em', 'figcaption', 'figure',
  'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'hr', 'i', 'img', 'li', 'ol', 'p', 'pre', 's', 'small', 'span', 'strong',
  'sub', 'sup', 'table', 'tbody', 'td', 'tfoot', 'th', 'thead', 'tr', 'u', 'ul',
])

const dropTags = new Set([
  'base', 'button', 'embed', 'form', 'frame', 'frameset', 'iframe', 'input', 'link', 'math', 'meta', 'noscript',
  'object', 'script', 'select', 'style', 'svg', 'template', 'textarea', 'title',
])

const keepAttrs = new Set(['alt', 'title', 'width', 'height', 'colspan', 'rowspan', 'align'])

export function sanitizeHtml(raw: string): string {
  if (!raw) return ''
  const doc = new DOMParser().parseFromString(raw, 'text/html')
  clean(doc.body)
  return doc.body.innerHTML
}

function clean(parent: Element) {
  for (const node of Array.from(parent.childNodes)) {
    if (node.nodeType === Node.TEXT_NODE) continue
    if (node.nodeType !== Node.ELEMENT_NODE) {
      node.remove()
      continue
    }
    const el = node as Element
    const tag = el.tagName.toLowerCase()
    if (dropTags.has(tag)) {
      el.remove()
      continue
    }
    clean(el)
    if (!keepTags.has(tag)) {
      el.replaceWith(...Array.from(el.childNodes))
      continue
    }
    for (const attr of Array.from(el.attributes)) {
      const name = attr.name.toLowerCase()
      const ok =
        keepAttrs.has(name) ||
        (tag === 'a' && name === 'href' && safeUrl(attr.value, true)) ||
        (tag === 'img' && name === 'src' && safeUrl(attr.value, false))
      if (!ok) el.removeAttribute(attr.name)
    }
    if (tag === 'a') {
      el.setAttribute('target', '_blank')
      el.setAttribute('rel', 'noopener noreferrer nofollow')
    }
  }
}

// 只放行 http https 和相对地址 链接额外放行 mailto
function safeUrl(value: string, link: boolean): boolean {
  const cleaned = Array.from(value)
    .filter((ch) => ch.charCodeAt(0) > 0x20 && ch.charCodeAt(0) !== 0x7f)
    .join('')
  const scheme = /^([a-z][a-z0-9+.-]*):/i.exec(cleaned)?.[1]?.toLowerCase()
  if (!scheme) return true
  if (scheme === 'http' || scheme === 'https') return true
  return link && scheme === 'mailto'
}
