package main

var html = `
<!DOCTYPE html>
<html>
<head>
  <title>前端发送数据到后端示例</title>
  <script src="https://code.jquery.com/jquery-3.6.0.min.js"></script>
</head>
<body>
  <input type="text" id="inputData" size="100" placeholder="输入数据">
  <button onclick="sendData()">发送</button>
  <div id="response"></div>

  <script>
    function sendData() {
      var data = document.getElementById("inputData").value;
      $.ajax({
        url: "http://localhost:8080/data", // 后端服务器的地址
        type: "POST",
        data: { data: data },
        success: function(response) {
          document.getElementById("response").innerText = response;
        }
      });
    }
  </script>
</body>
</html>
`
