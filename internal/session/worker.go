package session

import "time"

func (s *PlayerSession) runWorker() {
	var buffer[][]byte
	silenceTimer := time.NewTimer(0

	const silencieTimeout = 600 * time.Millisecond

	for {
		select {
		case chunk := <-s.AudioCh:
			buffer = append(buffer, chunk)

			if !silenceTimer.Stop() {
				select { case <-silenceTimer.C: default: }
			}
			silenceTimer.Reset(silencieTimeout)

		case <-silenceTimer.C:
			if len(buffer) > 0 {
				combined := combineChunks(buffer)
				buffer = buffer[:0]
				go s.processPipeLine(combined)
			}

		case <-s.AudioCh:
			return
		}
	}
}