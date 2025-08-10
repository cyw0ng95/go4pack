import { useState, useCallback, useEffect } from 'react'

export function usePool(API_BASE) {
  const [poolStats,setPoolStats] = useState(null)
  const fetchPool = useCallback(async () => {
    try {
      const r = await fetch(`${API_BASE.replace('/fileio','')}/pool/stats`)
      if (!r.ok) throw new Error('pool stats failed')
      const d = await r.json()
      setPoolStats(prev => prev ? { ...prev, ...d.pool } : d.pool)
    } catch(_){}
  }, [API_BASE])
  useEffect(()=>{ fetchPool(); const iv = setInterval(fetchPool, 1000); return ()=> clearInterval(iv) }, [fetchPool])
  return { poolStats }
}
