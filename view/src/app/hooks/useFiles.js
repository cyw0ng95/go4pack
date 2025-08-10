import { useState, useCallback } from 'react'

export function useFiles(API_BASE, handleErr) {
  const [files,setFiles] = useState([])
  const [loading,setLoading] = useState(false)
  const [page,setPage] = useState(1)
  const [pageSize,setPageSize] = useState(50)
  const [total,setTotal] = useState(0)
  const [pages,setPages] = useState(0)

  const fetchFiles = useCallback(async () => {
    setLoading(true)
    try {
      const r = await fetch(`${API_BASE}/list?page=${page}&page_size=${pageSize}`)
      if (!r.ok) throw new Error(`list failed ${r.status}`)
      const d = await r.json(); setFiles(d.files||[]); setTotal(d.total||0); setPages(d.pages||0)
    } catch(e){ handleErr(e) } finally { setLoading(false) }
  }, [API_BASE, page, pageSize, handleErr])

  return { files, setFiles, loading, fetchFiles, page, pageSize, total, pages, setPage, setPageSize }
}
