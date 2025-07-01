package util

func RtpEnc(data []byte, marker int, curpts uint64, istcp bool, ssrc, cseq int) []byte {

	pack := make([]byte, RTPHeaderLength)
	bits := BitsInit(RTPHeaderLength, pack)
	BitsWrite(bits, 2, 2)
	BitsWrite(bits, 1, 0)
	BitsWrite(bits, 1, 0)
	BitsWrite(bits, 4, 0)
	BitsWrite(bits, 1, uint64(marker))
	BitsWrite(bits, 7, 96)
	BitsWrite(bits, 16, uint64(cseq))
	BitsWrite(bits, 32, curpts)
	BitsWrite(bits, 32, uint64(ssrc))
	if istcp {
		var rtpOvertcp []byte
		lens := len(data) + 12
		rtpOvertcp = append(rtpOvertcp, byte(uint16(lens)>>8), byte(uint16(lens)))
		rtpOvertcp = append(rtpOvertcp, bits.pData...)
		return append(rtpOvertcp, data...)
	}
	return append(bits.pData, data...)

}
