import { useEffect, useRef } from 'react'
import type { Entry } from './useChat'

export default function Log({ entries }: { entries: Entry[] }) {
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (ref.current) ref.current.scrollTop = ref.current.scrollHeight
  }, [entries])

  return (
    <div id="log" ref={ref}>
      {entries.map((entry, i) =>
        entry.kind === 'message' ? (
          <div key={i} className={entry.from === 'you' ? 'msg you' : 'msg'}>
            <span className="role">{entry.from}</span>
            <span>{entry.text}</span>
          </div>
        ) : (
          <div key={i} className={entry.error ? 'tool err' : 'tool'}>
            {entry.text}
          </div>
        ),
      )}
    </div>
  )
}
