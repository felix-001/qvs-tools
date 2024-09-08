package main

import "time"

type BwKey struct {
	Bucket string
	Stream string
	Domain string
}

func (s *Parser) HlsChk() {
	ticker := time.NewTicker(time.Duration(10) * time.Second)
	defer ticker.Stop()

	historyBws := make([]uint64, 0, 20)

	for range ticker.C {
		var total uint64
		s.buildAllNodesMap()
		s.buildNodeStreamsMap()
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

		if len(historyBws) == 20 {
			historyBws = append(historyBws[1:], total)
		} else {
			historyBws = append(historyBws, total)
		}

		var historyTotal uint64
		for _, bw := range historyBws {
			historyTotal += bw
		}
		totalAvgBw := float64(historyTotal) / float64(len(historyBws)) * 8 / 1e6

		s.logger.Info().
			Float64("total", float64(total*8/1e6)).
			Float64("totalAvgBw", totalAvgBw).
			Int("historyBwsLen", len(historyBws)).
			Msg("")
	}
}
