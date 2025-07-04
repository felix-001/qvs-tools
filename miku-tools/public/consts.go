package public

var SepcialNodeList = []string{
	"9598268883e6a78ec2629ff1744f4a39",
	"12d36ff36fa07728ade4eee479149f4f",
	"caf569903bc0215098f86d8fc095c469",
	"f50582f62156c54d0eda2802a98858fd",
	"ff5b501909ecd88512dddc29cc61224f",
	"b4683a0f4ddf7797668ed49ec322192e",
	"9e5d7e2a58bf9a7470e88c9ed6f06297",
	"331b36824e2ffacd7112e9c24ff8d70f",
	"83ec106a407b3b755c7f91edb1438161",
	"c414604c8a7516db150c3d8d15aa8503",
	"d60266d39fde01d1127d9f5f75b0c416",
	"e5e68dc66d6101c13febbf3d641cd88c",
	"6c5c16c2db39a3c35791a3a7f4212d78",
	"da8cafc176443d28ff174d9e8e76c31d",
	"93af467fac9ee99d0adcd7ef7ea928ce",
	"7cbddce47b5307f6ca50f53f45ab0b3b",
	"a8bb3af1f150f4d468eb7a6eb855c5da",
	"05665e6a7161a9f62df6a3349dd9aced",
	"f490ec946a530fd07e5cc627e0d1ff51",
	"0edfb4be7a148ff879127aab3fcd6c37",
	"d6521f84210193e45b5bd811264a02d3",
	"74abc3ed9add54f0548a4d27c0c68a08",
	"402241bcfbc26f749287071dcc7bab56",
	"3698b5833be29d535b3d4b63f4f87281",
	"5d48e81d0122644530667ce46b741bf2",
	"f2b8d69b1a2e68962b1ea5def349a4ec",
	"26ac66b12327721e9f72160736ec48b2",
	"6a6301c51ae5ab911d00f625f45b761f",
	"1d134795dcc22aad950851b5d0b7cfb0"}

var CustomerIdMap map[uint32]string = map[uint32]string{
	1380460970: "七牛CDN",
	1380317970: "快手点播",
	10000005:   "快手直播",
	10000034:   "优酷专线",
	10000014:   "百度XCDN专线",
}

var Provinces = []string{"湖南", "内蒙古", "贵州", "山西", "河南", "天津", "江苏", "四川", "西藏", "湖北", "上海", "江西", "广东", "陕西", "辽宁", "河北", "山东", "福建", "云南", "新疆", "黑龙江", "宁夏", "安徽", "重庆", "浙江", "吉林", "海南", "甘肃", "青海", "北京", "广西"}

var Isps = []string{"移动", "电信", "联通"}

var Areas = []string{"东北", "华北", "华中", "华东", "华南", "西北", "西南"}

var AreaMap = map[string]string{
	"东北": "db",
	"华北": "hb",
	"华中": "hz",
	"华东": "hd",
	"华南": "hn",
	"西北": "xb",
	"西南": "xn",
}

var IspMap = map[string]string{
	"联通": "lt",
	"移动": "yd",
	"电信": "dx",
}
