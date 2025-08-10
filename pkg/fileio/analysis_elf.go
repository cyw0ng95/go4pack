package fileio

import (
	"encoding/json"

	elfutil "go4pack/pkg/common/elf"
	"go4pack/pkg/common/logger"
	"go4pack/pkg/common/worker"
)

// scheduleELFAnalysis submits an async job to analyze ELF and update DB record.
func scheduleELFAnalysis(recID uint, data []byte) {
	_ = worker.Submit(func() {
		logger.GetLogger().Debug().Uint("record_id", recID).Msg("starting async ELF analysis")
		db, err := ensureDB()
		if err != nil {
			return
		}
		analysis, aerr := elfutil.AnalyzeBytes(data)
		if aerr != nil {
			msg := aerr.Error()
			db.Model(&FileRecord{}).
				Where("id = ?", recID).
				Updates(map[string]any{"analysis_status": "error", "analysis_error": msg})
			logger.GetLogger().Error().Uint("record_id", recID).Err(aerr).Msg("elf analysis failed")
			return
		}
		b, _ := json.Marshal(analysis)
		js := string(b)
		cache := &ElfAnalyzeCached{FileID: recID, Data: js}
		_ = db.Where("file_id = ?", recID).
			Assign(map[string]any{"data": js}).
			FirstOrCreate(cache).Error
		db.Model(&FileRecord{}).Where("id = ?", recID).Update("analysis_status", "done")
		logger.GetLogger().Info().Uint("record_id", recID).Int("size", len(data)).Msg("elf analysis completed")
	})
}
