
function createLineChart(xAxis, yAxis) {
	var dom = document.createElement('div');
	var myChart = echarts.init(dom, null, {
		renderer: 'canvas',
		useDirtyRect: false
	});
	var option = {
		xAxis: {
			type: 'category',
			data: xAxis
		},
		yAxis: {
			type: 'value'
		},
		series: [{
			data: yAxis,
			type: 'line'
		}]
	};

	myChart.setOption(option);
	return dom;
}