package qos

type ChartGenerator interface {
	Generate(req QOSRequest) string
}

var ChartGenerators = map[string]ChartGenerator{}

func RegisterChartGenerator(name string, generator ChartGenerator) {
	ChartGenerators[name] = generator
}
