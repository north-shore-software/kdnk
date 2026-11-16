import { useRef, useState } from 'react'
import { createGuest, HttpError, signInWithPassword, upgradeGuest, type Guest } from './api'

const TOKEN_KEY = 'kdnk.resume-builder.token'
const GUEST_KEY = 'kdnk.resume-builder.guest'

function storedGuest(): Guest | null {
  try {
    return JSON.parse(localStorage.getItem(GUEST_KEY) ?? 'null')
  } catch {
    return null
  }
}

function prefixed(err: unknown, prefix: string): unknown {
  return err instanceof HttpError ? new HttpError(err.status, prefix + 'HTTP ' + err.status) : err
}

export function useSession() {
  const tokenRef = useRef(localStorage.getItem(TOKEN_KEY) ?? '')
  const [token, setToken] = useState(tokenRef.current)
  const [isGuest, setIsGuest] = useState(() => Boolean(localStorage.getItem(GUEST_KEY)))
  const [signedIn, setSignedIn] = useState(false)

  function accept(next: string) {
    tokenRef.current = next
    localStorage.setItem(TOKEN_KEY, next)
    setToken(next)
  }

  function forgetGuest() {
    localStorage.removeItem(GUEST_KEY)
    setIsGuest(false)
  }

  async function ensureAuth(): Promise<string> {
    if (!tokenRef.current) {
      const guest = storedGuest()
      let done = false
      if (guest?.guestId) {
        try {
          accept(await signInWithPassword(guest.guestId, guest.password))
          done = true
        } catch (err) {
          if (!(err instanceof HttpError)) throw err
        }
      }
      if (!done) {
        forgetGuest()
        let created
        try {
          created = await createGuest()
        } catch (err) {
          throw prefixed(err, 'guest sign-in failed: ')
        }
        localStorage.setItem(GUEST_KEY, JSON.stringify(created.guest))
        setIsGuest(true)
        accept(created.token)
      }
    }
    setSignedIn(true)
    return tokenRef.current
  }

  function reauth(): Promise<string> {
    tokenRef.current = ''
    localStorage.removeItem(TOKEN_KEY)
    setToken('')
    setSignedIn(false)
    return ensureAuth()
  }

  async function signIn(email: string, password: string): Promise<void> {
    let next
    try {
      next = await signInWithPassword(email.trim(), password)
    } catch (err) {
      throw prefixed(err, 'sign in failed: ')
    }
    forgetGuest()
    accept(next)
    location.reload()
  }

  async function upgrade(email: string, password: string): Promise<void> {
    let next
    try {
      next = await upgradeGuest(tokenRef.current, email.trim(), password)
    } catch (err) {
      throw prefixed(err, 'could not save account: ')
    }
    forgetGuest()
    accept(next)
    setSignedIn(true)
  }

  return { token, isGuest, signedIn, ensureAuth, reauth, signIn, upgrade }
}
