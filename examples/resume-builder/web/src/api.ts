export class HttpError extends Error {
  status: number

  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

export type Guest = { guestId: string; password: string }

export type Frame =
  | { type: 'text-start'; id: string }
  | { type: 'text-delta'; id: string; delta: string }
  | { type: 'tool-input-available'; toolName: string; input: unknown }
  | { type: 'tool-output-available'; output: string }
  | { type: 'error'; errorText: string }

export type StoredPart = {
  type: string
  text?: string
  data?: { toolName: string; input: unknown; output: string }
}

export type ChatMessage = { id: string; role: 'user' | 'assistant'; parts: StoredPart[] }

async function json<T>(res: Response): Promise<T> {
  if (!res.ok) throw new HttpError(res.status, 'HTTP ' + res.status)
  return res.json() as Promise<T>
}

function post(path: string, body: unknown, token?: string): Promise<Response> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (token !== undefined) headers.Authorization = token
  return fetch(path, { method: 'POST', headers, body: JSON.stringify(body) })
}

export async function signInWithPassword(identity: string, password: string): Promise<string> {
  const body = await json<{ token: string }>(
    await post('/api/collections/users/auth-with-password', { identity, password }),
  )
  return body.token
}

export async function createGuest(): Promise<{ token: string; guest: Guest }> {
  const body = await json<{ token: string; meta: Guest }>(await fetch('/api/guest', { method: 'POST' }))
  return { token: body.token, guest: body.meta }
}

export async function upgradeGuest(token: string, email: string, password: string): Promise<string> {
  const body = await json<{ token: string }>(await post('/api/guest/upgrade', { email, password }, token))
  return body.token
}

async function records<T>(token: string, path: string): Promise<{ items: T[] }> {
  return json(await fetch('/api/collections/' + path, { headers: { Authorization: token } }))
}

export async function loadHistory(token: string, conversationId: string): Promise<ChatMessage[]> {
  const convs = await records<{ id: string }>(
    token,
    'chat_conversation/records?perPage=1&filter='
      + encodeURIComponent('external_conversation_id="' + conversationId + '"'),
  )
  if (!convs.items.length) return []
  const stored = await records<{ content: ChatMessage }>(
    token,
    'chat_message/records?perPage=500&sort=seq&filter='
      + encodeURIComponent('conversation="' + convs.items[0].id + '"'),
  )
  return stored.items.map((rec) => rec.content)
}

export async function streamChat(
  token: string,
  conversationId: string,
  messages: ChatMessage[],
  onFrame: (f: Frame) => void,
): Promise<void> {
  const res = await post('/api/chat', { id: conversationId, messages }, token)
  if (!res.ok) throw new HttpError(res.status, 'HTTP ' + res.status + ': ' + (await res.text()))
  if (!res.body) return

  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })

    let split
    while ((split = buffer.indexOf('\n\n')) !== -1) {
      const raw = buffer.slice(0, split)
      buffer = buffer.slice(split + 2)
      if (!raw.startsWith('data: ')) continue
      const payload = raw.slice(6)
      if (payload === '[DONE]') continue
      let frame: Frame
      try {
        frame = JSON.parse(payload)
      } catch {
        onFrame({ type: 'error', errorText: 'bad frame: ' + payload })
        continue
      }
      onFrame(frame)
    }
  }
}
