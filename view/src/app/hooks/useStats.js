import { useState, useCallback } from 'react'

export function useStats(API_BASE, handleErr) {
  const [stats,setStats] = useState(null)
  const [statsLoading,setStatsLoading] = useState(false)
  const fetchStats = useCallback( async () => {
    setStatsLoading(true)
    try {
      const r = await fetch(`${API_BASE}/stats`)
      if (!r.ok) throw new Error(`stats failed ${r.status}`)
      const d = await r.json(); setStats(d)
    } catch(e){ handleErr(e) } finally { setStatsLoading(false) }
  }, [API_BASE, handleErr])
  return { stats, statsLoading, fetchStats }
}
