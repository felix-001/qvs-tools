package main

func (app *NetprobeSrv) routers() []Router {
	routers := []Router{
		{
			"/fillbw",
			[]string{"node", "type"},
			app.FillBw,
		},
		{
			"/runtimeState",
			[]string{"node", "state"},
			app.SetNodeRuntimeState,
		},
		{
			"/nodeinfo",
			[]string{"node"},
			app.NodeInfo,
		},
		{
			"/dumpAreaIsp",
			[]string{""},
			app.DumpAreaIsp,
		},
		{
			"/genneOfflineData",
			[]string{"area"},
			app.GeneOfflineData,
		},
		{
			"/streamreport",
			[]string{"node", "bucket", "stream"},
			app.StreamReport,
		},
		{
			"/lowThresholdTime",
			[]string{"node", "time"},
			app.SetLowThresholdTime,
		},
	}
	return routers
}
