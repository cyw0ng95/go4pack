'use client'
import { Card, CardContent, Grid, Typography, Stack, Chip, Divider } from '@mui/material'

export function StatsCards({ stats, statsLoading, formatFileSize, poolStats, poolLoading }) {
  return (
    <Grid container spacing={3}>
      {/* Files count */}
      <Grid item xs={12} md={3}>
        <Card><CardContent>
          <Typography variant='overline'>Files</Typography>
            <Typography variant='h4' sx={{ mt:1 }}>{statsLoading? '…' : stats?.file_count ?? 0}</Typography>
            <Typography variant='caption' color='text.secondary'>Unique {statsLoading? '…' : stats?.unique_hash_count ?? 0}</Typography>
        </CardContent></Card>
      </Grid>
      {/* Original vs Compressed */}
      <Grid item xs={12} md={3}>
        <Card><CardContent>
          <Typography variant='overline'>Original vs Compressed</Typography>
          <Typography variant='body2' sx={{ mt:1 }}>{statsLoading||!stats? '…' : `${formatFileSize(stats.total_original_size)} → ${formatFileSize(stats.total_compressed_size)}`}</Typography>
          <Typography variant='caption' color='text.secondary'>Saved {statsLoading||!stats? '…' : formatFileSize(stats.space_saved)}</Typography>
        </CardContent></Card>
      </Grid>
      {/* Physical Usage */}
      <Grid item xs={12} md={3}>
        <Card><CardContent>
          <Typography variant='overline'>Physical Usage</Typography>
          <Typography variant='body2' sx={{ mt:1 }}>{statsLoading||!stats? '…' : formatFileSize(stats.physical_objects_size||0)}</Typography>
          <Typography variant='caption' color='text.secondary'>Blobs {statsLoading||!stats? '…' : stats.physical_objects_count}</Typography>
        </CardContent></Card>
      </Grid>
      {/* Compression Ratio */}
      <Grid item xs={12} md={3}>
        <Card><CardContent>
          <Typography variant='overline'>Compression Ratio</Typography>
          <Typography variant='h5' sx={{ mt:1 }}>{statsLoading||!stats? '…' : (stats.compression_ratio? stats.compression_ratio.toFixed(2):'1.00')}</Typography>
          <Typography variant='caption' color='text.secondary'>Saved % {statsLoading||!stats? '…' : (stats.space_saved_percentage? stats.space_saved_percentage.toFixed(2)+'%':'0%')}</Typography>
        </CardContent></Card>
      </Grid>
      {/* Pool Stats */}
      <Grid item xs={12} md={4}>
        <Card><CardContent>
          <Typography variant='overline'>Worker Pool</Typography>
          <Stack spacing={0.5} sx={{ mt:1, fontSize:12 }}>
            <div>Running: {poolLoading||!poolStats? '…' : poolStats.running} / {poolLoading||!poolStats? '…' : poolStats.capacity}</div>
            <div>Free: {poolLoading||!poolStats? '…' : poolStats.free} | Queued: {poolLoading||!poolStats? '…' : poolStats.queued_est}</div>
            <div>Submitted: {poolLoading||!poolStats? '…' : poolStats.submitted} • Completed: {poolLoading||!poolStats? '…' : poolStats.completed}</div>
            <div>Last Duration: {poolLoading||!poolStats? '…' : (poolStats.last_duration_ms? poolStats.last_duration_ms+' ms':'-')}</div>
            <div>Last Finish: {poolLoading||!poolStats? '…' : (poolStats.last_finished_at? new Date(poolStats.last_finished_at).toLocaleTimeString(): '-')}</div>
            {poolStats?.last_error && <div style={{ color:'#d32f2f' }}>Last Error: {poolStats.last_error}</div>}
          </Stack>
        </CardContent></Card>
      </Grid>
      {/* Dedup savings */}
      {stats && (
        <Grid item xs={12} md={4}>
          <Card>
            <CardContent>
              <Typography variant='subtitle2' gutterBottom>Dedup Savings</Typography>
              <Stack spacing={1} sx={{ fontSize:12 }}>
                <div>Logical Compressed: {formatFileSize(stats.total_compressed_size||0)}</div>
                <div>Physical Compressed: {formatFileSize(stats.physical_objects_size||0)}</div>
                <div>Dedup Saved (Compressed): {formatFileSize(stats.dedup_saved_compressed||0)} ({stats.dedup_saved_compr_pct? stats.dedup_saved_compr_pct.toFixed(2)+'%':'0%'})</div>
                <div>Dedup Saved (Original Basis): {formatFileSize(stats.dedup_saved_original||0)} ({stats.dedup_saved_original_pct? stats.dedup_saved_original_pct.toFixed(2)+'%':'0%'})</div>
              </Stack>
            </CardContent>
          </Card>
        </Grid>
      )}
      {/* Compression & MIME */}
      {stats && (
        <Grid item xs={12} md={4}>
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
    </Grid>
  )
}
