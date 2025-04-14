package echonet_lite

import "reflect"

type PropertyRegistryEntry struct {
	ClassCode     EOJClassCode
	PropertyTable PropertyTable
}

type PropertyRegistry struct{}

type PropertyTableMap map[EOJClassCode]PropertyTable

func BuildPropertyTableMap() PropertyTableMap {
	// reflect を使って、 PropertyRegistry のメソッドのうち、戻り型が PropertyRegistryEntry のものを探す
	// そのメソッドを呼び出して、PropertyTableMap を作成する

	result := PropertyTableMap{}

	var registry interface{} = &PropertyRegistry{}
	t := reflect.TypeOf(registry)
	v := reflect.ValueOf(registry)
	for i := 0; i < t.NumMethod(); i++ {
		method := t.Method(i)
		if method.Type.Out(0) == reflect.TypeOf(PropertyRegistryEntry{}) {
			entry := v.Method(i).Call(nil)[0].Interface().(PropertyRegistryEntry)
			result[entry.ClassCode] = entry.PropertyTable
		}
	}
	return result
}
