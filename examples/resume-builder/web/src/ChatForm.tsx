import { useEffect, useRef, useState, type FormEvent } from 'react'

type Props = {
  hidden: boolean
  sending: boolean
  onSend: (text: string) => void
}

export default function ChatForm({ hidden, sending, onSend }: Props) {
  const [text, setText] = useState('')
  const input = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (!hidden && !sending) input.current?.focus()
  }, [hidden, sending])

  function submit(event: FormEvent) {
    event.preventDefault()
    const trimmed = text.trim()
    if (!trimmed) return
    setText('')
    onSend(trimmed)
  }

  return (
    <form id="form" className={hidden ? 'hide' : undefined} onSubmit={submit}>
      <input
        ref={input}
        autoComplete="off"
        placeholder="Tell me about a role, or paste a job description…"
        value={text}
        onChange={(e) => setText(e.target.value)}
      />
      <button type="submit" disabled={sending}>Send</button>
    </form>
  )
}
