'use client'

import { useState, useEffect } from 'react'
import {
  AppBar, Toolbar, Typography, Container, Grid, Card, CardContent, Box, Button,
  CircularProgress, Paper, Stack, IconButton, Snackbar, Alert, LinearProgress,
} from '@mui/material'
import CloudUploadIcon from '@mui/icons-material/CloudUpload'
import RefreshIcon from '@mui/icons-material/Refresh'
import DownloadIcon from '@mui/icons-material/Download'
import PlayArrowIcon from '@mui/icons-material/PlayArrow'
import CloseIcon from '@mui/icons-material/Close'
import PictureAsPdfIcon from '@mui/icons-material/PictureAsPdf'
import CodeIcon from '@mui/icons-material/Code'
// New component imports
import { UploadSessionCard } from './components/UploadSessionCard'
import { StatsCards } from './components/StatsCards'
import { UploadDropZone } from './components/UploadDropZone'
import { FilesTable } from './components/FilesTable'
import { PreviewDialog } from './components/PreviewDialog'
// Add hook imports
import { useApi } from './hooks/useApi'
import { useFiles } from './hooks/useFiles'
import { useStats } from './hooks/useStats'
import { usePool } from './hooks/usePool'

export default function Home() {
  // Remove local state now handled by hooks
  // const [files, setFiles] = useState([])
  const [uploading, setUploading] = useState(false)
  const [loading, setLoading] = useState(false) // will be overridden by hook loading
  const [dragOver, setDragOver] = useState(false)
  // const [stats, setStats] = useState(null)
  // const [statsLoading, setStatsLoading] = useState(false)
  // const [error, setError] = useState(null)
  // const [showError, setShowError] = useState(false)
  const [uploadQueue, setUploadQueue] = useState([])
  const [uploadProgress, setUploadProgress] = useState({ done: 0, total: 0 })
  const [uploadSession, setUploadSession] = useState(null)
  const [previewFile, setPreviewFile] = useState(null)
  const [previewOpen, setPreviewOpen] = useState(false)
  // const [poolStats, setPoolStats] = useState(null)
  // const [poolLoading, setPoolLoading] = useState(false)
  // const [page,setPage] = useState(1)
  // const [pageSize,setPageSize] = useState(50)
  // const [total,setTotal] = useState(0)
  // const [pages,setPages] = useState(0)
  const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://127.0.0.1:8080/api/fileio'

  // Hooks integration
  const { error, showError, setShowError, handleErr } = useApi(API_BASE)
  const { files, setFiles, loading: filesLoading, fetchFiles, page, pageSize, total, pages, setPage, setPageSize } = useFiles(API_BASE, handleErr)
  const { stats, statsLoading, fetchStats } = useStats(API_BASE, handleErr)
  const { poolStats } = usePool(API_BASE)

  const effectiveLoading = filesLoading // unify naming

  const formatFileSize = (bytes) => {
    if (bytes === undefined || bytes === null) return '-'
    if (bytes === 0) return '0 B'
    const k = 1024; const units = ['B','KB','MB','GB','TB']
    const i = Math.floor(Math.log(bytes)/Math.log(k))
    return `${(bytes/Math.pow(k,i)).toFixed(2)} ${units[i]}`
  }
  const formatDate = (ds) => new Date(ds).toLocaleString()

  // handleErr now from hook

  // Removed local fetchFiles/fetchStats/fetchPool definitions (now in hooks)

  const shouldBatchMulti = (arr) => {
    if (arr.length < 5) return false
    const total = arr.reduce((a,b)=>a+b.size,0)
    const avg = total / arr.length
    // Batch if many small files (avg < 256KB) or total <= 32MB and count <= 200
    return (avg < 256*1024 && arr.length >= 5) || (total <= 32*1024*1024 && arr.length <= 200)
  }

  const multiUploadRequest = async (arr) => {
    const fd = new FormData()
    arr.forEach(f=>fd.append('files', f))
    const start = Date.now()
    const r = await fetch(`${API_BASE}/upload/multi`, { method:'POST', body: fd })
    if (!r.ok) { let msg='multi upload failed'; try { const e=await r.json(); msg=e.error||msg } catch(_){} throw new Error(msg) }
    const end = Date.now()
    const resp = await r.json()
    // Update session entries
    const perFileDuration = (end-start) / (arr.length || 1)
    setUploadSession(s => s ? { ...s, files: [ ...s.files, ...(resp.results||[]).map(f=> ({ name: f.filename, size: f.original_size, start: start, end: end, durationMs: perFileDuration })) ] } : s)
    return resp
  }

  const uploadSingle = async (file) => {
    const fd = new FormData(); fd.append('file', file)
    const r = await fetch(`${API_BASE}/upload`, { method:'POST', body: fd })
    if (!r.ok) { let msg='upload failed'; try { const e=await r.json(); msg=e.error||msg } catch(_){} throw new Error(msg) }
    await r.json()
  }

  // Optimized parallel upload with adaptive concurrency & batching
  const uploadFiles = async (fileList) => {
    if (!fileList || fileList.length === 0) return
    // Convert & sort (largest first improves bandwidth utilization when mixed sizes)
    const filesArr = Array.from(fileList).sort((a,b)=> b.size - a.size)
    const sessionStart = Date.now()
    setUploadSession({ start: sessionStart, end: null, files: [] })
    setUploadQueue(filesArr.map(f=>f.name))
    setUploadProgress({ done:0, total: filesArr.length })
    setUploading(true)

    // Batch path (many small files) to reduce HTTP overhead
    if (shouldBatchMulti(filesArr)) {
      try {
        await multiUploadRequest(filesArr)
        setUploadProgress({ done: filesArr.length, total: filesArr.length })
        await Promise.all([fetchFiles(), fetchStats()])
      } catch (e) { handleErr(e) }
      finally {
        const end = Date.now()
        setUploadSession(s => s ? { ...s, end } : s)
        setUploading(false)
        setTimeout(()=>{ setUploadQueue([]); setUploadProgress({done:0,total:0}) }, 800)
      }
      return
    }

    // Parallel single-file path
    // Adaptive concurrency: base on hardware concurrency & avg size
    const totalBytes = filesArr.reduce((a,b)=>a+b.size,0)
    const avgSize = totalBytes / filesArr.length
    const cores = (typeof navigator !== 'undefined' && navigator.hardwareConcurrency) ? navigator.hardwareConcurrency : 4
    let base = Math.min(16, Math.max(2, Math.ceil(cores * 0.75)))
    if (avgSize < 512*1024) base = Math.min(32, base * 2) // more concurrency for tiny files
    let concurrency = Math.min(base, filesArr.length)

    let index = 0
    let done = 0

    const recordFile = (entry) => {
      setUploadSession(s => s ? { ...s, files: [...s.files, entry] } : s)
    }

    // Dynamic scaling: if queue remains large & workers finish quickly, spawn extra (up to cap)
    const maybeScale = (startTime) => {
      if (Date.now() - startTime < 300 && concurrency < Math.min(64, filesArr.length)) {
        concurrency += 1
        // Launch an extra worker
        worker()
      }
    }

    const worker = async () => {
      // capture worker start for scaling heuristic
      const wStart = Date.now()
      while (true) {
        let file
        if (index < filesArr.length) {
          file = filesArr[index++]
        } else {
          return
        }
        const start = Date.now()
        try { await uploadSingle(file) } catch (e) { handleErr(e) }
        const end = Date.now()
        recordFile({ name: file.name, size: file.size, start, end, durationMs: end - start })
        done += 1
        setUploadProgress(p => ({ ...p, done }))
        if (done === concurrency) { maybeScale(wStart) }
      }
    }

    try {
      await Promise.all([...Array(concurrency)].map(()=>worker()))
      await Promise.all([fetchFiles(), fetchStats()])
    } finally {
      const end = Date.now()
      setUploadSession(s => s ? { ...s, end } : s)
      setUploading(false)
      setTimeout(()=>{ setUploadQueue([]); setUploadProgress({done:0,total:0}) }, 800)
    }
  }

  const uploadFile = (file) => uploadFiles([file])

  const handleFileChange = (e) => uploadFiles(e.target.files)
  const handleDrop = (e) => { e.preventDefault(); setDragOver(false); uploadFiles(e.dataTransfer.files) }
  const handleDrag = (e, over) => { e.preventDefault(); setDragOver(over) }

  useEffect(()=>{ fetchFiles(); fetchStats() }, [fetchFiles, fetchStats])

  const refreshAll = () => { fetchFiles(); fetchStats() }

  // NEW: generic analysis fetcher using meta ?type=
  const fetchAnalysis = async (fileObj, type, prop) => {
    try {
      const r = await fetch(`${API_BASE}/meta/${fileObj.id}?type=${type}`)
      if (!r.ok) return
      const d = await r.json()
      if (d.analysis) {
        fileObj[prop] = d.analysis
        setFiles(fs => fs.map(x => x.id === fileObj.id ? { ...x, [prop]: d.analysis } : x))
      }
    } catch (_) { /* ignore */ }
  }

  const openPreview = async (file) => {
    // ELF analysis via new meta interface
    if ((file.is_elf || (file.available_analysis||[]).includes('elf')) && !file.elf_analysis) {
      await fetchAnalysis(file, 'elf', 'elf_analysis')
    }
    // GZIP analysis via new meta interface
    if ((isGzip(file) || (file.available_analysis||[]).includes('gzip')) && !file.gzip_analysis) {
      await fetchAnalysis(file, 'gzip', 'gzip_analysis')
    }
    setPreviewFile({ ...file })
    setPreviewOpen(true)
  }
  const closePreview = () => { setPreviewOpen(false); setPreviewFile(null) }
  const isVideo = (f) => !!f && typeof f.mime === 'string' && f.mime.startsWith('video/')
  const isPdf = (f) => !!f && typeof f.mime === 'string' && f.mime === 'application/pdf'
  const isElf = (f) => !!f && (f.is_elf || !!f.elf_analysis)
  const isText = (f) => !!f && typeof f.mime === 'string' && f.mime.startsWith('text/plain')
  const isGzip = (f) => !!f && typeof f.mime === 'string' && ['application/gzip','application/x-gzip'].includes(f.mime)
  const isPreviewable = (f) => isVideo(f) || isPdf(f) || isElf(f) || isText(f) || isGzip(f)

  const handlePageChange = (delta) => { setPage(p => Math.min(Math.max(1, p+delta), pages||1)) }
  const handlePageSizeChange = (e) => { setPageSize(parseInt(e.target.value)||50); setPage(1) }

  return (
    <Box sx={{ flexGrow:1, bgcolor:'background.default', minHeight:'100vh' }}>
      <AppBar position='static' color='primary' elevation={1}>
        <Toolbar>
          <Typography variant='h6' sx={{ flexGrow:1 }}>Go4Pack File Manager</Typography>
          <Button color='inherit' onClick={refreshAll} startIcon={<RefreshIcon />} disabled={effectiveLoading||statsLoading}>Refresh</Button>
          <Button component='label' color='inherit' variant='outlined' startIcon={<CloudUploadIcon/>} disabled={uploading} sx={{ ml:2 }}>
            {uploading ? 'Uploading' : 'Upload'}
            <input hidden type='file' multiple onChange={handleFileChange} />
          </Button>
        </Toolbar>
      </AppBar>
      <Container maxWidth='xl' sx={{ py:4 }}>
        <UploadSessionCard uploadSession={uploadSession} uploadProgress={uploadProgress} formatFileSize={formatFileSize} />
        {uploading && (
          <Box sx={{ mb:3 }}>
            <LinearProgress variant={uploadProgress.total? 'determinate':'indeterminate'} value={uploadProgress.total? (uploadProgress.done / uploadProgress.total)*100 : undefined} />
            <Typography variant='caption' sx={{ display:'block', mt:0.5 }}>
              Uploading {uploadProgress.done}/{uploadProgress.total} {uploadQueue[uploadProgress.done-1] ? `- ${uploadQueue[uploadProgress.done-1]}`:''}
            </Typography>
          </Box>
        )}
        <StatsCards stats={stats} statsLoading={statsLoading} formatFileSize={formatFileSize} poolStats={poolStats} />
        <Grid container spacing={3} sx={{ mt:1 }}>
          <Grid item xs={12}>
            <Card variant='outlined'>
              <CardContent>
                <Typography variant='h6' gutterBottom>Upload Files</Typography>
                <UploadDropZone onDrop={handleDrop} onDrag={handleDrag} dragOver={dragOver} uploading={uploading} />
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12}>
            <Paper elevation={2} sx={{ p:2 }}>
              <Stack direction='row' alignItems='center' justifyContent='space-between' sx={{ mb:2 }}>
                <Typography variant='h6'>Files</Typography>
                <Stack direction='row' spacing={1} alignItems='center'>
                  <Button size='small' startIcon={<RefreshIcon/>} onClick={refreshAll} disabled={effectiveLoading||statsLoading}>Refresh</Button>
                  <select value={pageSize} onChange={handlePageSizeChange} style={{ fontSize:12, padding:'4px 6px' }}>
                    {[25,50,100,200].map(s=> <option key={s} value={s}>{s}/page</option>)}
                  </select>
                  <Stack direction='row' spacing={0.5} alignItems='center'>
                    <Button size='small' disabled={page<=1} onClick={()=>handlePageChange(-1)}>Prev</Button>
                    <Typography variant='caption'>{page}/{pages||1}</Typography>
                    <Button size='small' disabled={page>=pages} onClick={()=>handlePageChange(1)}>Next</Button>
                  </Stack>
                </Stack>
              </Stack>
              <FilesTable files={files} loading={effectiveLoading} refreshAll={refreshAll} formatFileSize={formatFileSize} formatDate={formatDate} isVideo={isVideo} isPdf={isPdf} isElf={isElf} isText={isText} isPreviewable={isPreviewable} openPreview={openPreview} API_BASE={API_BASE} />
            </Paper>
          </Grid>
          <Grid item xs={12}>
            <Box sx={{ textAlign:'center', py:3, color:'text.secondary', fontSize:12 }}>File Manager powered by Go4Pack API</Box>
          </Grid>
        </Grid>
      </Container>
      <PreviewDialog open={previewOpen} file={previewFile} onClose={closePreview} API_BASE={API_BASE} isVideo={isVideo} isPdf={isPdf} isElf={isElf} isText={isText} />
      <Snackbar open={showError} autoHideDuration={4000} onClose={()=>setShowError(false)} anchorOrigin={{ vertical:'bottom', horizontal:'right' }}>
        <Alert severity='error' onClose={()=>setShowError(false)} variant='filled' sx={{ fontSize:12 }}>
          {error}
        </Alert>
      </Snackbar>
    </Box>
  )
}
