package yaml

import "reflect"

// this is a direct copy of the v2 encoder
// https://github.com/go-yaml/yaml/blob/7649d4548cb53a614db133b2a8ac1f31859dda8c/encode.go

var (
	mapItemType = reflect.TypeOf(MapItem{})
)

type MapSlice []MapItem

type MapItem struct {
	Key   any
	Value any
}

func (e *encoder) itemsv(tag string, in reflect.Value) {
	e.mappingv(tag, func() {
		slice := in.Convert(reflect.TypeOf([]MapItem{})).Interface().([]MapItem)
		for _, item := range slice {
			e.marshal("", reflect.ValueOf(item.Key))
			e.marshal("", reflect.ValueOf(item.Value))
		}
	})
}
