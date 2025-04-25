package echonet_lite

const (
	HomeAirConditioner_ClassCode     EOJClassCode = 0x0130 // 家庭用エアコン
	ElectricWaterHeater_ClassCode    EOJClassCode = 0x026b // 電気式給湯器(エコキュート含む) (TODO)
	FloorHeating_ClassCode           EOJClassCode = 0x027b // 床暖房
	SingleFunctionLighting_ClassCode EOJClassCode = 0x0291 // 単機能照明
	LightingSystem_ClassCode         EOJClassCode = 0x02a3 // 照明システム
	Refrigerator_ClassCode           EOJClassCode = 0x03b7 // 冷凍冷蔵庫
	Controller_ClassCode             EOJClassCode = 0x05ff // コントローラ
	NodeProfile_ClassCode            EOJClassCode = 0x0ef0 // ノードプロファイル
)
