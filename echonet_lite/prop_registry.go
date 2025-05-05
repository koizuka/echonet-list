package echonet_lite

import "reflect"

type PropertyRegistry struct{}

type PropertyTableMap map[EOJClassCode]PropertyTable

func BuildPropertyTableMap() PropertyTableMap {
	// reflect を使って、 PropertyRegistry のメソッドのうち、戻り型が PropertyRegistryEntry のものを探す
	// そのメソッドを呼び出して、PropertyTableMap を作成する

	_ = ManufacturerCodeEDTs // これを使うことで、PropertyTableMap の初期化時に ManufacturerCodeEDTs を参照できるようにする

	result := PropertyTableMap{}

	var registry any = &PropertyRegistry{}
	t := reflect.TypeOf(registry)
	v := reflect.ValueOf(registry)
	for i := range t.NumMethod() {
		method := t.Method(i)
		if method.Type.Out(0) == reflect.TypeOf(PropertyTable{}) {
			entry := v.Method(i).Call(nil)[0].Interface().(PropertyTable)
			result[entry.ClassCode] = entry
		}
	}
	return result
}
