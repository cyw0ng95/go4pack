'use client'
import React from 'react'
import { Dialog, DialogTitle, DialogContent, Box, Typography, IconButton as MIconButton, Alert, Divider, Chip, Stack, Accordion, AccordionSummary, AccordionDetails, Tooltip } from '@mui/material'
import CloseIcon from '@mui/icons-material/Close'
import ExpandMoreIcon from '@mui/icons-material/ExpandMore'
import ContentCopyIcon from '@mui/icons-material/ContentCopy'

export function PreviewDialog({ open, file, onClose, API_BASE, isVideo, isPdf, isElf }) {
  const status = file?.analysis_status
  const elfObj = (isElf(file) && file?.elf_analysis) ? safeParse(file.elf_analysis) : null
  const characteristics = elfObj?.characteristics || {}
  const chips = elfObj ? buildChips(characteristics) : []
  const copyAll = () => { if (!elfObj) return; navigator.clipboard?.writeText(JSON.stringify(elfObj, null, 2)) }
  const renderElfContent = () => {
    if (status === 'pending') return <Typography variant='body2' color='text.secondary'>Analysis in progress...</Typography>
    if (status === 'error') return <Typography variant='body2' color='error'>Failed to analyze ELF.</Typography>
    if (!elfObj) return <Typography variant='body2' color='text.secondary'>No ELF data.</Typography>
    return (
      <>
        <Alert severity='info' sx={{ mb:1, display:'flex', alignItems:'center', justifyContent:'space-between' }}>
          ELF Metadata
          <Tooltip title='Copy JSON'>
            <MIconButton size='small' onClick={copyAll} color='inherit'><ContentCopyIcon fontSize='inherit'/></MIconButton>
          </Tooltip>
        </Alert>
        <Stack direction='row' spacing={1} flexWrap='wrap' sx={{ mb:1 }}>
          {chips.map(c=> <Chip key={c.label} label={c.label} color={c.color} size='small' variant={c.variant||'filled'} />)}
        </Stack>
        <ElfAccordionView obj={elfObj} />
      </>
    )
  }
  return (
    <Dialog open={open} onClose={onClose} maxWidth='lg' fullWidth>
      <DialogTitle sx={{ pr:7 }}>
        {file?.filename}
        <MIconButton onClick={onClose} size='small' sx={{ position:'absolute', right:8, top:8 }}><CloseIcon fontSize='small'/></MIconButton>
      </DialogTitle>
      <DialogContent dividers sx={{ bgcolor: isElf(file)?'background.paper':'' }}>
        {isVideo(file) ? (
          <Box sx={{ aspectRatio:'16/9', width:'100%', bgcolor:'black' }}>
            <video key={file?.id} controls autoPlay style={{ width:'100%', height:'100%', objectFit:'contain' }} src={`${API_BASE}/download/${encodeURIComponent(file?.filename || '')}`} />
          </Box>
        ) : isPdf(file) ? (
          <Box sx={{ width:'100%', height:'80vh' }}>
            <iframe key={file?.id} title={file?.filename} style={{ border:0, width:'100%', height:'100%' }} src={`${API_BASE}/download/${encodeURIComponent(file?.filename || '')}#toolbar=0`} />
          </Box>
        ) : isElf(file) ? (
          <Box sx={{ width:'100%', maxHeight:'75vh', overflow:'auto', fontSize:13 }}>
            {renderElfContent()}
          </Box>
        ) : (
          <Typography variant='body2' color='text.secondary'>No preview available.</Typography>
        )}
      </DialogContent>
    </Dialog>
  )
}

function safeParse(v){
  if(!v) return null
  if (typeof v === 'object') return v
  if (typeof v === 'string') { try { return JSON.parse(v) } catch(_){ return null } }
  return null
}

function buildChips(ch) {
  const chips = []
  if (ch.static) chips.push({ label:'static', color:'primary' })
  if (ch.pie) chips.push({ label:'PIE', color:'secondary' })
  if (ch.stripped) chips.push({ label:'stripped', color:'warning' })
  if (ch.tls) chips.push({ label:'TLS', color:'info' })
  if (ch.go_binary) chips.push({ label:'Go', color:'success' })
  if (ch.libc) chips.push({ label: ch.libc, color:'default', variant:'outlined' })
  if (ch.compiler) chips.push({ label: shortCompiler(ch.compiler), color:'default', variant:'outlined' })
  return chips
}
function shortCompiler(s){ if(!s) return ''; return s.length>28? s.slice(0,25)+'…': s }

function ElfAccordionView({ obj }) {
  const groups = groupElf(obj)
  return (
    <Box>
      {groups.map(g => (
        <Accordion key={g.name} defaultExpanded={g.defaultExpanded} disableGutters square>
          <AccordionSummary expandIcon={<ExpandMoreIcon />}> <Typography fontWeight={600}>{g.title}</Typography> </AccordionSummary>
          <AccordionDetails>
            {g.kind==='kv' && <KVTable data={g.data} />}
            {g.kind==='array' && <ArrayPreview data={g.data} />}
            {g.kind==='json' && <PreJSON data={g.data} />}
          </AccordionDetails>
        </Accordion>
      ))}
    </Box>
  )
}

function groupElf(o){
  const groups = []
  groups.push({ name:'core', title:'Core Header', kind:'kv', data: pick(o,['class','endianness','type','machine','entry','osabi','abi_version']), defaultExpanded:true })
  groups.push({ name:'interp', title:'Interpreter / Paths', kind:'kv', data: pick(o,['interp','needed','rpath','runpath','build_id']) })
  groups.push({ name:'sections', title:`Sections (${o.sections})`, kind:'array', data: o.sections_detail })
  groups.push({ name:'top_sections', title:'Top Sections', kind:'array', data: o.top_sections })
  groups.push({ name:'program_headers', title:`Program Headers (${o.program_headers})`, kind:'array', data: o.program_headers_detail })
  groups.push({ name:'symbols', title:'Symbols Summary', kind:'kv', data: o.symbols })
  groups.push({ name:'relocations', title:'Relocations', kind:'kv', data: o.relocations })
  groups.push({ name:'sizes', title:'Section Sizes', kind:'kv', data: o.section_sizes })
  groups.push({ name:'debug', title:'Debug Info', kind:'kv', data: o.debug_info })
  groups.push({ name:'raw', title:'Raw JSON', kind:'json', data: o })
  return groups.filter(g=>g.data)
}
function pick(o, keys){ if(!o) return {}; const r={}; keys.forEach(k=>{ if(o[k]!==undefined) r[k]=o[k] }); return r }

function KVTable({ data }) {
  if (!data || Array.isArray(data)) return null
  return (
    <Box component='table' sx={{ width:'100%', borderCollapse:'collapse', '& td':{ borderBottom:'1px solid', borderColor:'divider', py:0.5, verticalAlign:'top' }, '& td:first-of-type':{ fontWeight:500, pr:2, width:160 } }}>
      <tbody>
        {Object.entries(data).map(([k,v])=> (
          <tr key={k}><td>{k}</td><td>{formatVal(v)}</td></tr>
        ))}
      </tbody>
    </Box>
  )
}
function ArrayPreview({ data }){
  if(!Array.isArray(data)) return null
  return (
    <Box sx={{ maxHeight:260, overflow:'auto', border:'1px solid', borderColor:'divider', borderRadius:1 }}>
      <Box component='table' sx={{ width:'100%', borderCollapse:'collapse', fontSize:12, '& td':{ borderBottom:'1px solid', borderColor:'divider', py:0.25, px:0.5 } }}>
        <tbody>
          {data.map((row,i)=> (
            <tr key={i}>
              <td><code>{truncate(JSON.stringify(row))}</code></td>
            </tr>
          ))}
        </tbody>
      </Box>
    </Box>
  )
}
function truncate(s){ return s.length>160? s.slice(0,157)+'…': s }

function PreJSON({ data }){
  return (
    <Box component='pre' sx={{ m:0, p:1.5, bgcolor:'grey.900', color:'grey.100', fontSize:12, lineHeight:1.4, borderRadius:1, overflow:'auto' }}>
      {syntaxHighlight(JSON.stringify(data,null,2))}
    </Box>
  )
}
function syntaxHighlight(jsonStr){
  // simple regex coloring
  return jsonStr.split(/(\".*?\"\s*:|\".*?\"|\b\d+\b|true|false|null)/g).map((part,i)=>{
    if(/^".*"\s*:$/.test(part)) return <span key={i} style={{ color:'#9cdcfe' }}>{part}</span>
    if(/^".*"$/.test(part)) return <span key={i} style={{ color:'#ce9178' }}>{part}</span>
    if(/\b\d+\b/.test(part)) return <span key={i} style={{ color:'#b5cea8' }}>{part}</span>
    if(/true|false/.test(part)) return <span key={i} style={{ color:'#569cd6' }}>{part}</span>
    if(/null/.test(part)) return <span key={i} style={{ color:'#808080' }}>{part}</span>
    return part
  })
}
function formatVal(v){ if(Array.isArray(v)) return v.join(', '); if(typeof v==='object') return JSON.stringify(v); return String(v) }
