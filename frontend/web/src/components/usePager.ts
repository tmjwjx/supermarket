import { useState } from 'react'

// 游标翻页 用栈记住走过的 page_token 以便回到上一页 key 变了就回到第一页
export function usePager(key: string) {
  const [state, setState] = useState<{ key: string; tokens: string[] }>({ key, tokens: [''] })
  const tokens = state.key === key ? state.tokens : ['']
  return {
    token: tokens[tokens.length - 1],
    page: tokens.length,
    next(token: string) {
      if (token) setState({ key, tokens: [...tokens, token] })
    },
    prev() {
      if (tokens.length > 1) setState({ key, tokens: tokens.slice(0, -1) })
    },
  }
}
