'use client'
import React from 'react'
import { Table, TableHead, TableRow, TableCell, TableBody, IconButton, Box, Typography, TextField, ButtonGroup, Button, LinearProgress } from '@mui/material'
import RefreshIcon from '@mui/icons-material/Refresh'
import DownloadIcon from '@mui/icons-material/Download'
import PlayArrowIcon from '@mui/icons-material/PlayArrow'
import PictureAsPdfIcon from '@mui/icons-material/PictureAsPdf'
import CodeIcon from '@mui/icons-material/Code'
import SortIcon from '@mui/icons-material/Sort'
import DescriptionIcon from '@mui/icons-material/Description'

export function FilesTable({ files, loading, refreshAll, formatFileSize, formatDate, isPreviewable, isVideo, isPdf, isElf, isText, openPreview, API_BASE }) {
  const [query, setQuery] = React.useState('')
  const [filter, setFilter] = React.useState('all') // all | video | pdf | elf
  const [sort, setSort] = React.useState({ field: 'created_at', dir: 'desc' })
  const maxSize = React.useMemo(()=> files.reduce((m,f)=> Math.max(m,f.size||0),0), [files])

  const filtered = files.filter(f => {
    if (query && !f.filename.toLowerCase().includes(query.toLowerCase())) return false
    if (filter === 'video' && !isVideo(f)) return false
    if (filter === 'pdf' && !isPdf(f)) return false
    if (filter === 'elf' && !isElf(f)) return false
    return true
  })
  const sorted = [...filtered].sort((a,b)=>{
    const dir = sort.dir === 'asc' ? 1 : -1
    if (sort.field === 'filename') return a.filename.localeCompare(b.filename) * dir
    if (sort.field === 'size') return (a.size - b.size) * dir
    if (sort.field === 'created_at') return (new Date(a.created_at) - new Date(b.created_at)) * dir
    return 0
  })

  const toggleSort = (field) => {
    setSort(s => s.field === field ? { field, dir: s.dir === 'asc' ? 'desc':'asc' } : { field, dir:'desc' })
  }

  if (loading) return <Box sx={{ textAlign:'center', py:6 }}><RefreshIcon className='spin' /></Box>
  if (!files.length) return <Box sx={{ textAlign:'center', py:6, color:'text.secondary' }}>No files uploaded</Box>
  return (
    <Box>
      <Box sx={{ display:'flex', flexWrap:'wrap', gap:1, mb:1 }}>
        <TextField size='small' placeholder='Search filename' value={query} onChange={e=>setQuery(e.target.value)} />
        <ButtonGroup size='small' variant='outlined'>
          {['all','video','pdf','elf'].map(f=> <Button key={f} onClick={()=>setFilter(f)} variant={filter===f?'contained':'outlined'}>{f}</Button>)}
        </ButtonGroup>
      </Box>
      <Table size='small'>
        <TableHead>
          <TableRow>
            <TableCell onClick={()=>toggleSort('filename')} sx={{ cursor:'pointer' }}>Filename <SortIcon fontSize='inherit' sx={{ opacity: sort.field==='filename'?1:0.2 }} /></TableCell>
            <TableCell>Type</TableCell>
            <TableCell onClick={()=>toggleSort('size')} sx={{ cursor:'pointer' }}>Size <SortIcon fontSize='inherit' sx={{ opacity: sort.field==='size'?1:0.2 }} /></TableCell>
            <TableCell onClick={()=>toggleSort('created_at')} sx={{ cursor:'pointer' }}>Uploaded <SortIcon fontSize='inherit' sx={{ opacity: sort.field==='created_at'?1:0.2 }} /></TableCell>
            <TableCell align='right'>Actions</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {sorted.map(f => (
            <TableRow key={f.id} hover>
              <TableCell sx={{ cursor: isPreviewable(f)?'pointer':'default', color: isPreviewable(f)?'primary.main':'inherit' }} onClick={()=> isPreviewable(f)&&openPreview(f)}>{f.filename}</TableCell>
              <TableCell>{f.mime || (isElf(f)?'application/x-elf':'-')}</TableCell>
              <TableCell>
                <Box sx={{ display:'flex', alignItems:'center', gap:1 }}>
                  <Box sx={{ flexGrow:1 }}>
                    <LinearProgress variant='determinate' value={maxSize? (f.size/maxSize)*100 : 0} sx={{ height:6, borderRadius:1, bgcolor:'divider' }} />
                  </Box>
                  <Typography variant='caption'>{formatFileSize(f.size)}</Typography>
                </Box>
              </TableCell>
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
                {isElf(f) && (
                  <IconButton size='small' color='secondary' onClick={()=>openPreview(f)} sx={{ mr:0.5 }}>
                    <CodeIcon fontSize='inherit'/>
                  </IconButton>
                )}
                {isText(f) && (
                  <IconButton size='small' color='secondary' onClick={()=>openPreview(f)} sx={{ mr:0.5 }}>
                    <DescriptionIcon fontSize='inherit'/>
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
    </Box>
  )
}
