package main

func RtpEnc(data []byte, marker int, curpts uint64, istcp bool, ssrc, cseq int) []byte {

	pack := make([]byte, RTPHeaderLength)
	bits := bitsInit(RTPHeaderLength, pack)
	bitsWrite(bits, 2, 2)
	bitsWrite(bits, 1, 0)
	bitsWrite(bits, 1, 0)
	bitsWrite(bits, 4, 0)
	bitsWrite(bits, 1, uint64(marker))
	bitsWrite(bits, 7, 96)
	bitsWrite(bits, 16, uint64(cseq))
	bitsWrite(bits, 32, curpts)
	bitsWrite(bits, 32, uint64(ssrc))
	if istcp {
		var rtpOvertcp []byte
		lens := len(data) + 12
		rtpOvertcp = append(rtpOvertcp, byte(uint16(lens)>>8), byte(uint16(lens)))
		rtpOvertcp = append(rtpOvertcp, bits.pData...)
		return append(rtpOvertcp, data...)
	}
	return append(bits.pData, data...)

}
