package main

import "time"

func (s *Parser) HlsChk() {
	var total uint64
	for _, nodeStreams := range s.nodeStremasMap {
		if time.Now().Unix()-nodeStreams.LastUpdateTime > 300 {
			continue
		}
		for _, streamInfoRT := range nodeStreams.Streams {
			for _, player := range streamInfoRT.Players {
				for _, ipInfo := range player.Ips {
					if player.Protocol == "hls" {
						s.logger.Info().
							Uint64("hlsBytes", ipInfo.HlsBytes).
							Str("ip", ipInfo.Ip).
							Str("stream", streamInfoRT.Key).
							Str("bkt", streamInfoRT.Bucket).
							Str("node", nodeStreams.NodeId).
							Str("domain", streamInfoRT.Domain).
							Msg("")
						total += ipInfo.HlsBytes
					}
				}
			}
		}
	}

	s.logger.Info().Float64("total", float64(total*8/1e6)).Msg("")
}
