'use client'
import { Dialog, DialogTitle, DialogContent, Box, Typography, IconButton as MIconButton, Alert, Divider } from '@mui/material'
import CloseIcon from '@mui/icons-material/Close'

export function PreviewDialog({ open, file, onClose, API_BASE, isVideo, isPdf, isElf }) {
  return (
    <Dialog open={open} onClose={onClose} maxWidth='md' fullWidth>
      <DialogTitle sx={{ pr:5 }}>
        {file?.filename}
        <MIconButton onClick={onClose} size='small' sx={{ position:'absolute', right:8, top:8 }}><CloseIcon fontSize='small'/></MIconButton>
      </DialogTitle>
      <DialogContent dividers>
        {isVideo(file) ? (
          <Box sx={{ aspectRatio:'16/9', width:'100%', bgcolor:'black' }}>
            <video
              key={file?.id}
              controls
              autoPlay
              style={{ width:'100%', height:'100%', objectFit:'contain' }}
              src={`${API_BASE}/download/${encodeURIComponent(file?.filename || '')}`}
            />
          </Box>
        ) : isPdf(file) ? (
          <Box sx={{ width:'100%', height:'80vh' }}>
            <iframe
              key={file?.id}
              title={file?.filename}
              style={{ border:0, width:'100%', height:'100%' }}
              src={`${API_BASE}/download/${encodeURIComponent(file?.filename || '')}#toolbar=0`}
            />
          </Box>
        ) : isElf(file) ? (
          <Box sx={{ width:'100%', maxHeight:'70vh', overflow:'auto', fontFamily:'monospace', fontSize:12 }}>
            <Alert severity='info' sx={{ mb:1 }}>ELF Metadata</Alert>
            {file?.elf_analysis ? renderElf(JSON.parse(file.elf_analysis)) : 'No ELF data'}
          </Box>
        ) : (
          <Typography variant='body2' color='text.secondary'>No preview available.</Typography>
        )}
      </DialogContent>
    </Dialog>
  )
}

function renderElf(obj) {
  const entries = Object.entries(obj)
  return (
    <Box component='pre' sx={{ m:0 }}>
      {entries.map(([k,v]) => (
        <div key={k}><strong>{k}:</strong> {formatVal(v)}</div>
      ))}
    </Box>
  )
}
function formatVal(v){ if(Array.isArray(v)) return v.join(', '); if(typeof v==='object') return JSON.stringify(v); return String(v) }
