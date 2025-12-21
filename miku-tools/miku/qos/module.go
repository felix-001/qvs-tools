package qos

type ChartGenerator interface {
	Generate(req QOSRequest) any
	ID() string
	ChartInfo() ChartInfo
}

var ChartGenerators = map[string]ChartGenerator{}

func RegisterChartGenerator(generator ChartGenerator) {
	ChartGenerators[generator.ID()] = generator
}
