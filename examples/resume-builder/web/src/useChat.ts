import { useRef, useState } from 'react'
import { HttpError, loadHistory, streamChat, type ChatMessage, type Frame } from './api'
import type { useSession } from './useSession'

const CONVERSATION_KEY = 'kdnk.resume-builder.conversation'
const OPENING = "Let's build your career audit. Tell me about your current or most recent role."

export type Entry =
  | { kind: 'message'; from: 'you' | 'assistant'; text: string }
  | { kind: 'note'; text: string; error: boolean }

function conversation(): string {
  let id = localStorage.getItem(CONVERSATION_KEY)
  if (!id) {
    id = crypto.randomUUID()
    localStorage.setItem(CONVERSATION_KEY, id)
  }
  return id
}

function describe(err: unknown): string {
  return err instanceof HttpError ? err.message : String(err)
}

function isUnauthorized(err: unknown): boolean {
  return err instanceof HttpError && err.status === 401
}

export function useChat(session: ReturnType<typeof useSession>) {
  const [conversationId] = useState(conversation)
  const listRef = useRef<Entry[]>([])
  const [entries, setEntries] = useState<Entry[]>([])
  const [sending, setSending] = useState(false)
  const messagesRef = useRef<ChatMessage[]>([])

  function push(entry: Entry): number {
    listRef.current = [...listRef.current, entry]
    setEntries(listRef.current)
    return listRef.current.length - 1
  }

  function appendText(index: number, delta: string) {
    const list = listRef.current.slice()
    const entry = list[index]
    if (entry.kind !== 'message') return
    list[index] = { ...entry, text: entry.text + delta }
    listRef.current = list
    setEntries(list)
  }

  function note(text: string, error = false) {
    push({ kind: 'note', text, error })
  }

  function showStored(msg: ChatMessage) {
    for (const part of msg.parts ?? []) {
      if (part.type === 'text') {
        if (part.text) push({ kind: 'message', from: msg.role === 'user' ? 'you' : 'assistant', text: part.text })
      } else if (part.type.startsWith('tool-') && part.data) {
        note('→ ' + part.data.toolName + ' ' + JSON.stringify(part.data.input))
        note('← ' + part.data.output)
      }
    }
  }

  async function start(): Promise<void> {
    try {
      let token = await session.ensureAuth()
      let history
      try {
        history = await loadHistory(token, conversationId)
      } catch (err) {
        if (!isUnauthorized(err)) throw err
        token = await session.reauth()
        history = await loadHistory(token, conversationId)
      }
      for (const msg of history) {
        showStored(msg)
        if (msg.role === 'user') messagesRef.current.push(msg)
      }
      if (!history.length) push({ kind: 'message', from: 'assistant', text: OPENING })
    } catch (err) {
      note(describe(err), true)
    }
  }

  async function send(text: string): Promise<void> {
    setSending(true)
    push({ kind: 'message', from: 'you', text })
    messagesRef.current.push({ id: crypto.randomUUID(), role: 'user', parts: [{ type: 'text', text }] })

    const targets: Record<string, number> = {}
    const onFrame = (frame: Frame) => {
      if (frame.type === 'text-start') {
        targets[frame.id] = push({ kind: 'message', from: 'assistant', text: '' })
      } else if (frame.type === 'text-delta') {
        if (frame.id in targets) appendText(targets[frame.id], frame.delta)
      } else if (frame.type === 'tool-input-available') {
        note('→ ' + frame.toolName + ' ' + JSON.stringify(frame.input))
      } else if (frame.type === 'tool-output-available') {
        note('← ' + frame.output)
      } else if (frame.type === 'error') {
        note(frame.errorText, true)
      }
    }

    try {
      try {
        await streamChat(session.token, conversationId, messagesRef.current, onFrame)
      } catch (err) {
        if (!isUnauthorized(err)) throw err
        await streamChat(await session.reauth(), conversationId, messagesRef.current, onFrame)
      }
    } catch (err) {
      note(describe(err), true)
    } finally {
      setSending(false)
    }
  }

  return { entries, sending, start, send, note }
}
