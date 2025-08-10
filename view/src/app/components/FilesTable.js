'use client'
import { Table, TableHead, TableRow, TableCell, TableBody, IconButton, Box, Typography } from '@mui/material'
import RefreshIcon from '@mui/icons-material/Refresh'
import DownloadIcon from '@mui/icons-material/Download'
import PlayArrowIcon from '@mui/icons-material/PlayArrow'
import PictureAsPdfIcon from '@mui/icons-material/PictureAsPdf'

export function FilesTable({ files, loading, refreshAll, formatFileSize, formatDate, isPreviewable, isVideo, isPdf, openPreview, API_BASE }) {
  if (loading) return <Box sx={{ textAlign:'center', py:6 }}><RefreshIcon className='spin' /></Box>
  if (!files.length) return <Box sx={{ textAlign:'center', py:6, color:'text.secondary' }}>No files uploaded</Box>
  return (
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
            <TableCell sx={{ cursor: isPreviewable(f)?'pointer':'default', color: isPreviewable(f)?'primary.main':'inherit' }} onClick={()=> isPreviewable(f)&&openPreview(f)}>{f.filename}</TableCell>
            <TableCell>{formatFileSize(f.size)}</TableCell>
            <TableCell>{formatDate(f.created_at)}</TableCell>
            <TableCell align='right'>
              {isVideo(f) && (
                <IconButton size='small' color='secondary' onClick={()=>openPreview(f)} sx={{ mr:0.5 }}>
                  <PlayArrowIcon fontSize='inherit'/>
                </IconButton>
              )}
              {isPdf(f) && (
                <IconButton size='small' color='secondary' onClick={()=>openPreview(f)} sx={{ mr:0.5 }}>
                  <PictureAsPdfIcon fontSize='inherit'/>
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
  )
}
