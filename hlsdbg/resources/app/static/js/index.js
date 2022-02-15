let index = {
    about: function(html) {
        let c = document.createElement("div");
        c.innerHTML = html;
        asticode.modaler.setContent(c);
        asticode.modaler.show();
    },
    createLineChart: function() {
        let ctx = document.createElement("canvas");
        document.getElementById("files_panel").append(ctx);
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

        let ctx = document.createElement("canvas");
        document.getElementById("files_panel").append(ctx);
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
        c0 = this.createLineChart("c0")
        ids = ["c0", "c1", "c2", "c3"]
        for (id in ids) {
            c = 
        }

        let ctx1 = document.createElement("canvas");
        document.getElementById("c2").append(ctx1);
        var c = new Chart(ctx1, {
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

        let ctx2 = document.createElement("canvas");
        document.getElementById("c3").append(ctx2);
        var c = new Chart(ctx2, {
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

        let ctx4 = document.createElement("canvas");
        document.getElementById("c4").append(ctx4);
        var c = new Chart(ctx4, {
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