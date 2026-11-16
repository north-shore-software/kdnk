import { useEffect, useRef, useState, type FormEvent } from 'react'

type Props = {
  open: boolean
  isGuest: boolean
  signedIn: boolean
  onSignIn: (email: string, password: string) => void
  onUpgrade: (email: string, password: string) => Promise<boolean>
}

export default function AccountForm({ open, isGuest, signedIn, onSignIn, onUpgrade }: Props) {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const emailInput = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (open) emailInput.current?.focus()
  }, [open])

  async function upgrade() {
    if (await onUpgrade(email, password)) setPassword('')
  }

  function submit(event: FormEvent) {
    event.preventDefault()
    if (isGuest) upgrade()
    else onSignIn(email, password)
  }

  return (
    <form id="auth" className={open ? 'show' : undefined} onSubmit={submit}>
      <input
        ref={emailInput}
        type="email"
        autoComplete="username"
        placeholder="email"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
      />
      <input
        type="password"
        autoComplete="current-password"
        placeholder="password"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
      />
      <button type="button" disabled={!signedIn || !isGuest} onClick={upgrade}>Save account</button>
      <button type="button" onClick={() => onSignIn(email, password)}>Sign in</button>
    </form>
  )
}
