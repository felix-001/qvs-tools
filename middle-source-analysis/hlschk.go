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

	lastBwsMap := make(map[BwKey][]uint64)

	for range ticker.C {
		var total uint64
		s.buildAllNodesMap()
		s.buildNodeStreamsMap()
		for _, nodeStreams := range s.nodeStremasMap {
			if time.Now().Unix()-nodeStreams.LastUpdateTime > 300 {
				continue
			}
			for _, streamInfoRT := range nodeStreams.Streams {
				var nodeTotal uint64
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
							nodeTotal += ipInfo.HlsBytes
						}
					}
				}

				if nodeTotal <= 0 {
					continue
				}
				key := BwKey{
					Bucket: streamInfoRT.Bucket,
					Stream: streamInfoRT.Key,
					Domain: streamInfoRT.Domain,
				}
				if _, ok := lastBwsMap[key]; !ok {
					lastBwsMap[key] = make([]uint64, 0, 20)
				}
				s.logger.Info().Uint64("nodeTotal", nodeTotal).Msg("")
				if len(lastBwsMap[key]) == 20 {
					lastBwsMap[key] = append(lastBwsMap[key][1:], nodeTotal)
				} else {
					lastBwsMap[key] = append(lastBwsMap[key], nodeTotal)

				}

			}
		}

		var totalAvgBw float64
		for key, bws := range lastBwsMap {
			var total uint64
			for _, bw := range bws {
				total += bw
			}
			avgBw := (float64(total) / float64(len(bws))) * 8 / 1e6
			s.logger.Info().
				Int("bwsLen", len(bws)).
				Float64("avgBw", avgBw).
				Str("bucket", key.Bucket).
				Str("domain", key.Domain).
				Str("stream", key.Stream).
				Uint64("total", total).
				Msg("")
			totalAvgBw += avgBw
		}

		s.logger.Info().
			Float64("total", float64(total*8/1e6)).
			Float64("totalAvgBw", totalAvgBw).
			Msg("")
	}
}
