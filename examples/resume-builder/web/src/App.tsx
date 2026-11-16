import { useEffect, useRef, useState, type MouseEvent } from 'react'
import AccountForm from './AccountForm'
import ChatForm from './ChatForm'
import Log from './Log'
import { HttpError } from './api'
import { useChat } from './useChat'
import { useSession } from './useSession'

function describe(err: unknown): string {
  return err instanceof HttpError ? err.message : String(err)
}

export default function App() {
  const session = useSession()
  const chat = useChat(session)
  const started = useRef(false)
  const [hasSignedIn, setHasSignedIn] = useState(false)
  const [authOpen, setAuthOpen] = useState(false)

  useEffect(() => {
    if (started.current) return
    started.current = true
    chat.start()
  }, [])

  useEffect(() => {
    if (!session.signedIn) return
    setHasSignedIn(true)
    setAuthOpen(false)
  }, [session.signedIn])

  const signedOut = hasSignedIn && !session.signedIn

  async function signIn(email: string, password: string) {
    try {
      await session.signIn(email, password)
    } catch (err) {
      chat.note(describe(err), true)
    }
  }

  async function upgrade(email: string, password: string): Promise<boolean> {
    try {
      await session.upgrade(email, password)
    } catch (err) {
      chat.note(describe(err), true)
      return false
    }
    setAuthOpen(false)
    chat.note('account saved — sign in with this email and password next time')
    return true
  }

  function toggleAccount(event: MouseEvent) {
    event.preventDefault()
    setAuthOpen((open) => !open)
  }

  return (
    <main>
      <h1>kdnk — resume-builder example</h1>
      <Log entries={chat.entries} />
      <AccountForm
        open={authOpen || signedOut}
        isGuest={session.isGuest}
        signedIn={session.signedIn}
        onSignIn={signIn}
        onUpgrade={upgrade}
      />
      <ChatForm hidden={signedOut} sending={chat.sending} onSend={chat.send} />
      <p id="account">
        <a href="#" onClick={toggleAccount}>{session.isGuest ? 'save your account' : 'sign in'}</a>
      </p>
    </main>
  )
}
