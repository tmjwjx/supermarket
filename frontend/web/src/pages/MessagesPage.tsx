import { useEffect, useState } from 'react'
import { listNotifications, markNotificationsRead, readNotifications, type Notice } from '../api/notice.ts'

export default function MessagesPage() {
  const [items, setItems] = useState<Notice[]>([])
  const [unread, setUnread] = useState(0)
  const [message, setMessage] = useState('')
  const [failed, setFailed] = useState(false)

  function load() {
    return listNotifications().then((body) => {
      const next = readNotifications(body)
      setItems(next.items)
      setUnread(next.unread)
    })
  }

  useEffect(() => {
    let gone = false
    load().catch((err: unknown) => {
      if (gone) return
      setFailed(true)
      setMessage(err instanceof Error ? err.message : '消息加载失败')
    })
    return () => {
      gone = true
    }
  }, [])

  async function onRead(ids: string[]) {
    if (ids.length === 0) return
    setMessage('')
    setFailed(false)
    try {
      await markNotificationsRead(ids)
      await load()
    } catch (err) {
      setFailed(true)
      setMessage(err instanceof Error ? err.message : '标记已读失败')
    }
  }

  const unreadIds = items.filter((item) => !item.read).map((item) => item.id)

  return (
    <section>
      <div className="slip">
        <h1>消息</h1>
        <p className="lead">未读 {unread}</p>
        {unreadIds.length > 0 ? (
          <button className="quiet" type="button" onClick={() => onRead(unreadIds)}>
            全部已读
          </button>
        ) : null}
      </div>
      {items.length === 0 && !message ? <p className="note">还没有消息</p> : null}
      <div className="goods">
        {items.map((item) => (
          <article className="item" key={item.id}>
            <strong>{item.title || '通知'}</strong>
            <span className="meta">{item.body}</span>
            {item.read ? <span className="meta">已读</span> : (
              <button className="quiet" type="button" onClick={() => onRead([item.id])}>
                标为已读
              </button>
            )}
          </article>
        ))}
      </div>
      {message ? <p className={failed ? 'note bad' : 'note'}>{message}</p> : null}
    </section>
  )
}
