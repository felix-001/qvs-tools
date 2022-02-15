let index = {
    about: function(html) {
        let c = document.createElement("div");
        c.innerHTML = html;
        asticode.modaler.setContent(c);
        asticode.modaler.show();
    },
    createLineChart: function(id) {
        let ctx = document.createElement("canvas");
        document.getElementById(id).append(ctx);
        var c = new Chart(ctx, {
            // 要创建的图表类型
            type: 'line',
            // 数据集
            data: {
                labels: ["January", "February", "March", "April", "May", "June", "July"],
                datasets: [{
                    label: "My First dataset",
                    backgroundColor: 'rgb(255, 99, 132)',
                    borderColor: 'rgb(255, 99, 132)',
                    data: [0, 10, 5, 2, 20, 30, 45],
                }]
            },
            // 配置选项
            options: {}
        });
        return c
    },
    init: function() {
        // Init
        asticode.loader.init();
        asticode.modaler.init();
        asticode.notifier.init();

        c01 = this.createLineChart("c00")
        c01 = this.createLineChart("c01")
        c02 = this.createLineChart("c02")
        c11 = this.createLineChart("c10")
        c11 = this.createLineChart("c11")
        c12 = this.createLineChart("c12")
        c21 = this.createLineChart("c20")
        c21 = this.createLineChart("c21")
        c22 = this.createLineChart("c22")

        // Wait for astilectron to be ready
        document.addEventListener('astilectron-ready', function() {
            index.listen();
            index.show();
        })
    },
    show: function(path) {
        // Create message
        let message = {"name": "disp"};
        if (typeof path !== "undefined") {
            message.payload = path
        }

        // Send message
        asticode.loader.show();
        astilectron.sendMessage(message, function(message) {
            // Init
            asticode.loader.hide();

            // Check error
            if (message.name === "error") {
                asticode.notifier.error(message.payload);
                return
            }

            // Process files
            document.getElementById("files").innerHTML = "";
            if (typeof message.payload.chart !== "undefined") {
                document.getElementById("files_panel").style.display = "block";
                let canvas = document.createElement("canvas");
                document.getElementById("files").append(canvas);
                chart = new Chart(canvas, message.payload.chart);
            } else {
                document.getElementById("files_panel").style.display = "none";
            }
        })
    },
    listen: function() {
        astilectron.onMessage(function(message) {
            switch (message.name) {
                case "about":
                    index.about(message.payload);
                    return {payload: "payload"};
                case "update":
                    chart.data.datasets[0].data = message.payload
                    chart.update()
                    break;
            }
        });
    }
};