'use client'
import { Box, Typography, CircularProgress } from '@mui/material'

export function UploadDropZone({ onDrop, onDrag, dragOver, uploading }) {
  return (
    <Box
      onDragOver={(e)=>onDrag(e,true)}
      onDragLeave={(e)=>onDrag(e,false)}
      onDrop={onDrop}
      sx={{ mt:2, border:'2px dashed', borderColor: dragOver? 'primary.main':'divider', p:6, textAlign:'center', borderRadius:2, bgcolor: dragOver? 'action.hover':'background.paper' }}
    >
      <Typography variant='body1' color='text.secondary'>Drag & drop a file here or click Upload above</Typography>
      {uploading && <Box sx={{ mt:2 }}><CircularProgress size={28} /></Box>}
    </Box>
  )
}
