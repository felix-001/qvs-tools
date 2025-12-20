package qos

import "log"

type ChartGenerator interface {
	Generate(req QOSRequest) any
}

type ChartGenerator2 interface {
	Generate(req QOSRequest) any
	ID() string
}

var ChartGenerators = map[string]ChartGenerator{}

//var ChartGenerators2 = map[string]ChartGenerator2{}

func RegisterChartGenerator(name string, generator ChartGenerator) {
	ChartGenerators[name] = generator
}

func RegisterChartGenerator2(generator ChartGenerator2) {
	log.Println("RegisterChartGenerator2", generator.ID())
	ChartGenerators[generator.ID()] = generator
}
