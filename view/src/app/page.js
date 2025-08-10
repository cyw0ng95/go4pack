'use client'

import { useState, useEffect, useCallback } from 'react'
import {
  AppBar, Toolbar, Typography, Container, Grid, Card, CardContent, Box, Button,
  CircularProgress, Table, TableHead, TableRow, TableCell, TableBody, Paper,
  Chip, Stack, Divider, IconButton, Snackbar, Alert, LinearProgress,
  Dialog, DialogTitle, DialogContent, IconButton as MIconButton
} from '@mui/material'
import CloudUploadIcon from '@mui/icons-material/CloudUpload'
import RefreshIcon from '@mui/icons-material/Refresh'
import DownloadIcon from '@mui/icons-material/Download'
import PlayArrowIcon from '@mui/icons-material/PlayArrow'
import CloseIcon from '@mui/icons-material/Close'

export default function Home() {
  const [files, setFiles] = useState([])
  const [uploading, setUploading] = useState(false)
  const [loading, setLoading] = useState(false)
  const [dragOver, setDragOver] = useState(false)
  const [stats, setStats] = useState(null)
  const [statsLoading, setStatsLoading] = useState(false)
  const [error, setError] = useState(null)
  const [showError, setShowError] = useState(false)
  const [uploadQueue, setUploadQueue] = useState([]) // filenames queued
  const [uploadProgress, setUploadProgress] = useState({ done: 0, total: 0 })
  const [uploadSession, setUploadSession] = useState(null) // {start, end, files:[{name,size,start,end,durationMs}]}
  const [previewFile, setPreviewFile] = useState(null)
  const [previewOpen, setPreviewOpen] = useState(false)

  const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:8080/api/fileio'

  const formatFileSize = (bytes) => {
    if (bytes === undefined || bytes === null) return '-'
    if (bytes === 0) return '0 B'
    const k = 1024; const units = ['B','KB','MB','GB','TB']
    const i = Math.floor(Math.log(bytes)/Math.log(k))
    return `${(bytes/Math.pow(k,i)).toFixed(2)} ${units[i]}`
  }
  const formatDate = (ds) => new Date(ds).toLocaleString()

  const handleErr = (e) => { setError(e?.message || 'Error'); setShowError(true) }

  const fetchFiles = useCallback(async () => {
    setLoading(true)
    try {
      const r = await fetch(`${API_BASE}/list`)
      if (!r.ok) throw new Error(`list failed ${r.status}`)
      const d = await r.json(); setFiles(d.files||[])
    } catch (e) { handleErr(e) } finally { setLoading(false) }
  }, [API_BASE])

  const fetchStats = useCallback(async () => {
    setStatsLoading(true)
    try {
      const r = await fetch(`${API_BASE}/stats`)
      if (!r.ok) throw new Error(`stats failed ${r.status}`)
      const d = await r.json(); setStats(d)
    } catch (e) { handleErr(e) } finally { setStatsLoading(false) }
  }, [API_BASE])

  // Helper: decide if batch multi-upload is beneficial
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

  // Helper metrics for session
  const renderUploadSession = () => {
    if (!uploadSession) return null
    const { start, end, files: uf } = uploadSession
    const now = Date.now()
    const effectiveEnd = end || now
    const durationMs = effectiveEnd - start
    const totalBytes = uf.reduce((a,b)=> a + (b.size||0), 0)
    const avgPerFile = uf.length ? (durationMs / uf.length) : 0
    const throughput = durationMs > 0 ? (totalBytes / (durationMs/1000)) : 0 // bytes/sec
    const fmtDur = (ms) => {
      if (ms < 1000) return ms + ' ms'
      const s = ms/1000
      if (s < 60) return s.toFixed(2) + ' s'
      const m = Math.floor(s/60); const rem = s - m*60
      return `${m}m ${rem.toFixed(1)}s`
    }
    const fmtBps = (bps) => {
      if (bps <= 0) return '-'
      const k = 1024
      const units = ['B/s','KB/s','MB/s','GB/s']
      let i=0; while (bps >= k && i < units.length-1){ bps/=k; i++ }
      return bps.toFixed(2)+' '+units[i]
    }
    return (
      <Card variant='outlined' sx={{ mb:3 }}>
        <CardContent>
          <Stack direction='row' justifyContent='space-between' alignItems='flex-start' spacing={2} flexWrap='wrap'>
            <Box sx={{ minWidth: 180 }}>
              <Typography variant='overline'>Upload Session</Typography>
              <Typography variant='body2'>Start: {new Date(start).toLocaleTimeString()}</Typography>
              <Typography variant='body2'>End: {end ? new Date(end).toLocaleTimeString() : '…'}</Typography>
              <Typography variant='body2'>Duration: {fmtDur(durationMs)}</Typography>
            </Box>
            <Box sx={{ minWidth: 180 }}>
              <Typography variant='overline'>Performance</Typography>
              <Typography variant='body2'>Files: {uploadProgress.total}</Typography>
              <Typography variant='body2'>Completed: {uf.length}</Typography>
              <Typography variant='body2'>Avg/File: {fmtDur(avgPerFile)}</Typography>
              <Typography variant='body2'>Throughput: {fmtBps(throughput)}</Typography>
            </Box>
            <Box sx={{ minWidth: 220, flexGrow:1 }}>
              <Typography variant='overline'>Recent Files</Typography>
              <Stack spacing={0.5} sx={{ maxHeight:120, overflow:'auto', mt:0.5 }}>
                {uf.slice(-5).reverse().map(f=> (
                  <Typography key={f.name+f.start} variant='caption' sx={{ display:'block' }}>
                    {f.name} • {formatFileSize(f.size)} • {fmtDur(f.durationMs)}
                  </Typography>
                ))}
                {uf.length === 0 && <Typography variant='caption' color='text.secondary'>Pending…</Typography>}
              </Stack>
            </Box>
          </Stack>
        </CardContent>
      </Card>
    )
  }

  const openPreview = (file) => { setPreviewFile(file); setPreviewOpen(true) }
  const closePreview = () => { setPreviewOpen(false); setPreviewFile(null) }
  const isVideo = (f) => !!f && typeof f.mime === 'string' && f.mime.startsWith('video/')

  return (
    <Box sx={{ flexGrow:1, bgcolor:'background.default', minHeight:'100vh' }}>
      <AppBar position='static' color='primary' elevation={1}>
        <Toolbar>
          <Typography variant='h6' sx={{ flexGrow:1 }}>Go4Pack File Manager</Typography>
          <Button color='inherit' onClick={refreshAll} startIcon={<RefreshIcon />} disabled={loading||statsLoading}>Refresh</Button>
          <Button component='label' color='inherit' variant='outlined' startIcon={<CloudUploadIcon/>} disabled={uploading} sx={{ ml:2 }}>
            {uploading ? 'Uploading' : 'Upload'}
            <input hidden type='file' multiple onChange={handleFileChange} />
          </Button>
        </Toolbar>
      </AppBar>
      <Container maxWidth='xl' sx={{ py:4 }}>
        {renderUploadSession()}
        {uploading && (
          <Box sx={{ mb:3 }}>
            <LinearProgress variant={uploadProgress.total? 'determinate':'indeterminate'} value={uploadProgress.total? (uploadProgress.done / uploadProgress.total)*100 : undefined} />
            <Typography variant='caption' sx={{ display:'block', mt:0.5 }}>
              Uploading {uploadProgress.done}/{uploadProgress.total} {uploadQueue[uploadProgress.done-1] ? `- ${uploadQueue[uploadProgress.done-1]}`:''}
            </Typography>
          </Box>
        )}
        <Grid container spacing={3}>
          <Grid item xs={12} md={3}>
            <Card><CardContent>
              <Typography variant='overline'>Files</Typography>
              <Typography variant='h4' sx={{ mt:1 }}>{statsLoading? '…' : stats?.file_count ?? 0}</Typography>
              <Typography variant='caption' color='text.secondary'>Unique {statsLoading? '…' : stats?.unique_hash_count ?? 0}</Typography>
            </CardContent></Card>
          </Grid>
          <Grid item xs={12} md={3}>
            <Card><CardContent>
              <Typography variant='overline'>Original vs Compressed</Typography>
              <Typography variant='body2' sx={{ mt:1 }}>{statsLoading||!stats? '…' : `${formatFileSize(stats.total_original_size)} → ${formatFileSize(stats.total_compressed_size)}`}</Typography>
              <Typography variant='caption' color='text.secondary'>Saved {statsLoading||!stats? '…' : formatFileSize(stats.space_saved)}</Typography>
            </CardContent></Card>
          </Grid>
          <Grid item xs={12} md={3}>
            <Card><CardContent>
              <Typography variant='overline'>Physical Usage</Typography>
              <Typography variant='body2' sx={{ mt:1 }}>{statsLoading||!stats? '…' : formatFileSize(stats.physical_objects_size||0)}</Typography>
              <Typography variant='caption' color='text.secondary'>Blobs {statsLoading||!stats? '…' : stats.physical_objects_count}</Typography>
            </CardContent></Card>
          </Grid>
          <Grid item xs={12} md={3}>
            <Card><CardContent>
              <Typography variant='overline'>Compression Ratio</Typography>
              <Typography variant='h5' sx={{ mt:1 }}>{statsLoading||!stats? '…' : (stats.compression_ratio? stats.compression_ratio.toFixed(2):'1.00')}</Typography>
              <Typography variant='caption' color='text.secondary'>Saved % {statsLoading||!stats? '…' : (stats.space_saved_percentage? stats.space_saved_percentage.toFixed(2)+'%':'0%')}</Typography>
            </CardContent></Card>
          </Grid>

          {stats && (
            <Grid item xs={12} md={6}>
              <Card>
                <CardContent>
                  <Typography variant='subtitle2' gutterBottom>Dedup Savings</Typography>
                  <Stack spacing={1} sx={{ fontSize:12 }}>
                    <Box>Logical Compressed: {formatFileSize(stats.total_compressed_size||0)}</Box>
                    <Box>Physical Compressed: {formatFileSize(stats.physical_objects_size||0)}</Box>
                    <Box>Dedup Saved (Compressed): {formatFileSize(stats.dedup_saved_compressed||0)} ({stats.dedup_saved_compr_pct? stats.dedup_saved_compr_pct.toFixed(2)+'%':'0%'})</Box>
                    <Box>Dedup Saved (Original Basis): {formatFileSize(stats.dedup_saved_original||0)} ({stats.dedup_saved_original_pct? stats.dedup_saved_original_pct.toFixed(2)+'%':'0%'})</Box>
                  </Stack>
                </CardContent>
              </Card>
            </Grid>
          )}
          {stats && (
            <Grid item xs={12} md={6}>
              <Card>
                <CardContent>
                  <Typography variant='subtitle2' gutterBottom>Compression & MIME Types</Typography>
                  <Typography variant='caption' color='text.secondary'>Compression</Typography>
                  <Stack direction='row' spacing={1} flexWrap='wrap' sx={{ my:1 }}>
                    {Object.entries(stats.compression_types||{}).map(([k,v]) => <Chip key={k} label={`${k||'unknown'}: ${v}`} size='small' color='primary' variant='outlined' />)}
                  </Stack>
                  <Divider sx={{ my:1 }} />
                  <Typography variant='caption' color='text.secondary'>MIME</Typography>
                  <Stack direction='row' spacing={1} flexWrap='wrap' sx={{ mt:1 }}>
                    {Object.entries(stats.mime_types||{}).map(([k,v]) => <Chip key={k} label={`${k||'unknown'}: ${v}`} size='small' color='secondary' variant='outlined' />)}
                  </Stack>
                </CardContent>
              </Card>
            </Grid>
          )}

          <Grid item xs={12}>
            <Card variant='outlined'>
              <CardContent>
                <Typography variant='h6' gutterBottom>Upload Files</Typography>
                <Box
                  onDragOver={(e)=>handleDrag(e,true)}
                  onDragLeave={(e)=>handleDrag(e,false)}
                  onDrop={handleDrop}
                  sx={{ mt:2, border:'2px dashed', borderColor: dragOver? 'primary.main':'divider', p:6, textAlign:'center', borderRadius:2, bgcolor: dragOver? 'action.hover':'background.paper' }}
                >
                  <Typography variant='body1' color='text.secondary'>Drag & drop a file here or click Upload above</Typography>
                  {uploading && <Box sx={{ mt:2 }}><CircularProgress size={28} /></Box>}
                </Box>
              </CardContent>
            </Card>
          </Grid>

          <Grid item xs={12}>
            <Paper elevation={2} sx={{ p:2 }}>
              <Stack direction='row' alignItems='center' justifyContent='space-between' sx={{ mb:2 }}>
                <Typography variant='h6'>Files</Typography>
                <Button size='small' startIcon={<RefreshIcon/>} onClick={refreshAll} disabled={loading||statsLoading}>Refresh</Button>
              </Stack>
              {loading ? (
                <Box sx={{ textAlign:'center', py:6 }}><CircularProgress /></Box>
              ) : files.length === 0 ? (
                <Box sx={{ textAlign:'center', py:6, color:'text.secondary' }}>No files uploaded</Box>
              ) : (
                <Table size='small'>
                  <TableHead>
                    <TableRow>
                      <TableCell>Filename</TableCell>
                      <TableCell>Size</TableCell>
                      <TableCell>Uploaded</TableCell>
                      <TableCell align='right'>Actions</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {files.map(f => (
                      <TableRow key={f.id} hover>
                        <TableCell sx={{ cursor: isVideo(f)?'pointer':'default', color: isVideo(f)?'primary.main':'inherit' }} onClick={()=> isVideo(f)&&openPreview(f)}>{f.filename}</TableCell>
                        <TableCell>{formatFileSize(f.size)}</TableCell>
                        <TableCell>{formatDate(f.created_at)}</TableCell>
                        <TableCell align='right'>
                          {isVideo(f) && (
                            <IconButton size='small' color='secondary' onClick={()=>openPreview(f)} sx={{ mr:0.5 }}>
                              <PlayArrowIcon fontSize='inherit'/>
                            </IconButton>
                          )}
                          <IconButton size='small' color='primary' onClick={()=> window.open(`${API_BASE}/download/${encodeURIComponent(f.filename)}`,'_blank')}>
                            <DownloadIcon fontSize='inherit' />
                          </IconButton>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </Paper>
          </Grid>

          <Grid item xs={12}>
            <Box sx={{ textAlign:'center', py:3, color:'text.secondary', fontSize:12 }}>File Manager powered by Go4Pack API</Box>
          </Grid>
        </Grid>
      </Container>
      <Dialog open={previewOpen} onClose={closePreview} maxWidth='md' fullWidth>
        <DialogTitle sx={{ pr:5 }}>
          {previewFile?.filename}
          <MIconButton onClick={closePreview} size='small' sx={{ position:'absolute', right:8, top:8 }}><CloseIcon fontSize='small'/></MIconButton>
        </DialogTitle>
        <DialogContent dividers>
          {isVideo(previewFile) ? (
            <Box sx={{ aspectRatio:'16/9', width:'100%', bgcolor:'black' }}>
              <video
                key={previewFile?.id}
                controls
                autoPlay
                style={{ width:'100%', height:'100%', objectFit:'contain' }}
                src={`${API_BASE}/download/${encodeURIComponent(previewFile?.filename || '')}`}
              />
            </Box>
          ) : (
            <Typography variant='body2' color='text.secondary'>No preview available.</Typography>
          )}
        </DialogContent>
      </Dialog>
      <Snackbar open={showError} autoHideDuration={4000} onClose={()=>setShowError(false)} anchorOrigin={{ vertical:'bottom', horizontal:'right' }}>
        <Alert severity='error' onClose={()=>setShowError(false)} variant='filled' sx={{ fontSize:12 }}>
          {error}
        </Alert>
      </Snackbar>
    </Box>
  )
}
