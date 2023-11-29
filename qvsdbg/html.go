package main

var html = `
<!DOCTYPE html>
<html>
<head>
  <title>QVS日志搜索</title>
  <script src="https://code.jquery.com/jquery-3.6.0.min.js"></script>
</head>
<body>
  <div>
  <input type="text" id="inputData" size="100" placeholder="输入搜索关键字,以逗号分隔,例如: 202090356770,15010400401320001441">
  <button onclick="sendData()">搜索SIP原始日志</button>
  </div>

  <div>
  <input type="text" id="inputDataService" size="100" placeholder="输入搜索关键字,以逗号分隔,例如: 202090356770,15010400401320001441">
  <button onclick="searchServiceLog()">搜索服务日志</button>
  </div>

  <div id="response"></div>

  <script>
    function sendData() {
      var data = document.getElementById("inputData").value;
      $.ajax({
        url: "http://127.0.0.1:8001/sip", // 后端服务器的地址
        type: "POST",
        data: { data: data },
        success: function(response) {
          document.getElementById("response").innerText = response;
	  downloadFile(response, 'sip_logs.txt');
        }
      });
    }

    function searchServiceLog() {
      var data = document.getElementById("inputDataService").value;
      $.ajax({
        url: "http://127.0.0.1:8001/service", // 后端服务器的地址
        type: "POST",
        data: { data: data },
        success: function(response) {
          document.getElementById("response").innerText = response;
	  downloadFile(response, 'sip_logs.txt');
        }
      });
    }

   function downloadFile(content, filename) {
    var element = document.createElement('a');
    element.setAttribute('href', 'data:text/plain;charset=utf-8,' + encodeURIComponent(content));
    element.setAttribute('download', filename);

    element.style.display = 'none';
    document.body.appendChild(element);

    element.click();

    document.body.removeChild(element);
  }

  </script>
</body>
</html>

`
