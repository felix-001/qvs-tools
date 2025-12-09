package qos

import "log"

type ChartGenerator interface {
	Generate(req QOSRequest) string
}

var ChartGenerators = map[string]ChartGenerator{}

func RegisterChartGenerator(name string, generator ChartGenerator) {
	log.Println("RegisterChartGenerator", name)
	ChartGenerators[name] = generator
}
