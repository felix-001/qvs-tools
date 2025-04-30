
import argparse

class LogParser(object):
    def __init__(self, time, srv, string):
        self.time = time
        self.string = string

	if srv == 'server':
		self.type = 'APP_QVS-SERVER'
	elif srv == 'flowd':
		self.type = 'APP_PILI-FLOWD'



    def parse(self):
	path = "/"
        pass



if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('-t', '--time', type=str, required=True,
                        help="时间,格式:2024-03-30 15-00")
    parser.add_argument('-srv', type=str, required=True,
                        help="服务,server/sip/rtp/flowd/seg/themisd")
    parser.add_argument('-s', '--string', type=str, required=True,
                        help="需要搜索的字符串,类似于grep")
    args = parser.parse_args()
    