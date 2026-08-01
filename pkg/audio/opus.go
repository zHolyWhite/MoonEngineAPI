package audio

import (
	"bytes"
	"encoding/binary"
)

func CombinePCM(chunks [][]byte) []byte {
	total := 0
	for _, c := range chunks {
		total += len(c)
	}
	out := make([]byte, 0, total)
	for _, c := range chunks {
		out = append(out, c...)
	}
	return out
}

func PCMToWAV(pcm []byte, sampleRate, channels, bitsPerSample int) []byte {
	dataSize := uint32(len(pcm))
	buf := &bytes.Buffer{}

	buf.WriteString("RIFF")
	binary.Write(buf, binary.LittleEndian, uint32(36+dataSize))
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	binary.Write(buf, binary.LittleEndian, uint32(16))
	binary.Write(buf, binary.LittleEndian, uint16(1))
	binary.Write(buf, binary.LittleEndian, uint16(channels))
	binary.Write(buf, binary.LittleEndian, uint32(sampleRate))
	byteRate := uint32(sampleRate * channels * bitsPerSample / 8)
	binary.Write(buf, binary.LittleEndian, byteRate)
	blockAlign := uint16(channels * bitsPerSample / 8)
	binary.Write(buf, binary.LittleEndian, blockAlign)
	binary.Write(buf, binary.LittleEndian, uint16(bitsPerSample))
	buf.WriteString("data")
	binary.Write(buf, binary.LittleEndian, dataSize)
	buf.Write(pcm)

	return buf.Bytes()
}
