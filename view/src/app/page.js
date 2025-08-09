'use client'

import { useState, useEffect, useCallback } from 'react'
import {
  AppBar, Toolbar, Typography, Container, Grid, Card, CardContent, Box, Button,
  CircularProgress, Table, TableHead, TableRow, TableCell, TableBody, Paper,
  Chip, Stack, Divider, IconButton, Snackbar, Alert
} from '@mui/material'
import CloudUploadIcon from '@mui/icons-material/CloudUpload'
import RefreshIcon from '@mui/icons-material/Refresh'
import DownloadIcon from '@mui/icons-material/Download'

export default function Home() {
  const [files, setFiles] = useState([])
  const [uploading, setUploading] = useState(false)
  const [loading, setLoading] = useState(false)
  const [dragOver, setDragOver] = useState(false)
  const [stats, setStats] = useState(null)
  const [statsLoading, setStatsLoading] = useState(false)
  const [error, setError] = useState(null)
  const [showError, setShowError] = useState(false)

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

  const uploadFile = async (file) => {
    if (!file) return
    setUploading(true)
    try {
      const fd = new FormData(); fd.append('file', file)
      const r = await fetch(`${API_BASE}/upload`, { method:'POST', body: fd })
      if (!r.ok) { let msg='upload failed'; try { const e=await r.json(); msg=e.error||msg } catch(_){} throw new Error(msg) }
      await r.json(); await Promise.all([fetchFiles(), fetchStats()])
    } catch (e) { handleErr(e) } finally { setUploading(false) }
  }

  const handleFileChange = (e) => uploadFile(e.target.files?.[0])
  const handleDrop = (e) => { e.preventDefault(); setDragOver(false); uploadFile(e.dataTransfer.files?.[0]) }
  const handleDrag = (e, over) => { e.preventDefault(); setDragOver(over) }

  useEffect(()=>{ fetchFiles(); fetchStats() }, [fetchFiles, fetchStats])

  const refreshAll = () => { fetchFiles(); fetchStats() }

  return (
    <Box sx={{ flexGrow:1, bgcolor:'background.default', minHeight:'100vh' }}>
      <AppBar position='static' color='primary' elevation={1}>
        <Toolbar>
          <Typography variant='h6' sx={{ flexGrow:1 }}>Go4Pack File Manager</Typography>
          <Button color='inherit' onClick={refreshAll} startIcon={<RefreshIcon />} disabled={loading||statsLoading}>Refresh</Button>
          <Button component='label' color='inherit' variant='outlined' startIcon={<CloudUploadIcon/>} disabled={uploading} sx={{ ml:2 }}>
            Upload
            <input hidden type='file' onChange={handleFileChange} />
          </Button>
        </Toolbar>
      </AppBar>
      <Container maxWidth='xl' sx={{ py:4 }}>
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
                        <TableCell>{f.filename}</TableCell>
                        <TableCell>{formatFileSize(f.size)}</TableCell>
                        <TableCell>{formatDate(f.created_at)}</TableCell>
                        <TableCell align='right'>
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
      <Snackbar open={showError} autoHideDuration={4000} onClose={()=>setShowError(false)} anchorOrigin={{ vertical:'bottom', horizontal:'right' }}>
        <Alert severity='error' onClose={()=>setShowError(false)} variant='filled' sx={{ fontSize:12 }}>
          {error}
        </Alert>
      </Snackbar>
    </Box>
  )
}
