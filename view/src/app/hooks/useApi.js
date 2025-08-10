import { useState, useCallback } from 'react'

export function useApi(base) {
  const [error,setError] = useState(null)
  const [showError,setShowError] = useState(false)
  const handleErr = useCallback(e => { setError(e?.message||'Error'); setShowError(true) }, [])
  return { base, error, showError, setShowError, handleErr }
}
