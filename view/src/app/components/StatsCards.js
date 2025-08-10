'use client'
import { Card, CardContent, Grid, Typography, Stack, Chip, Divider } from '@mui/material'

export function StatsCards({ stats, statsLoading, formatFileSize }) {
  return (
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
                <div>Logical Compressed: {formatFileSize(stats.total_compressed_size||0)}</div>
                <div>Physical Compressed: {formatFileSize(stats.physical_objects_size||0)}</div>
                <div>Dedup Saved (Compressed): {formatFileSize(stats.dedup_saved_compressed||0)} ({stats.dedup_saved_compr_pct? stats.dedup_saved_compr_pct.toFixed(2)+'%':'0%'})</div>
                <div>Dedup Saved (Original Basis): {formatFileSize(stats.dedup_saved_original||0)} ({stats.dedup_saved_original_pct? stats.dedup_saved_original_pct.toFixed(2)+'%':'0%'})</div>
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
    </Grid>
  )
}
