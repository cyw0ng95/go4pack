'use client'
import { Card, CardContent, Stack, Box, Typography } from '@mui/material'

export function UploadSessionCard({ uploadSession, uploadProgress, formatFileSize }) {
  if (!uploadSession) return null
  const { start, end, files: uf } = uploadSession
  const now = Date.now()
  const effectiveEnd = end || now
  const durationMs = effectiveEnd - start
  const totalBytes = uf.reduce((a,b)=> a + (b.size||0), 0)
  const avgPerFile = uf.length ? (durationMs / uf.length) : 0
  const throughput = durationMs > 0 ? (totalBytes / (durationMs/1000)) : 0
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
