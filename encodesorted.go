package yaml

import (
	"encoding/json"
	"reflect"
	"sort"
)

// this is a direct copy of the v2 encoder
// https://github.com/go-yaml/yaml/blob/7649d4548cb53a614db133b2a8ac1f31859dda8c/encode.go

var (
	mapItemType = reflect.TypeOf(MapItem{})

	_ json.Marshaler   = MapSlice{}
	_ json.Unmarshaler = &MapSlice{}
)

type MapSlice []MapItem

type MapItem struct {
	Key   any
	Value any
}

func (e *encoder) itemsv(tag string, slice MapSlice) {
	e.mappingv(tag, func() {
		for _, item := range slice {
			e.marshal("", reflect.ValueOf(item.Key))
			e.marshal("", reflect.ValueOf(item.Value))
		}
	})
}

// mappingSlice decodes a YAML node into a MapSlice-like structure (which is ordered).
func (d *decoder) mappingSlice(n *Node, out reflect.Value) (good bool) {
	outt := out.Type()
	if outt.Elem() != mapItemType {
		d.terror(n, yaml_MAP_TAG, out)
		return false
	}

	mapType := d.stringMapType
	d.stringMapType = outt
	d.generalMapType = outt

	// Prepare to collect items as MapSlice
	var slice []MapItem

	// Loop through the content of the node (which holds both keys and values)
	l := len(n.Content)
	for i := 0; i < l; i += 2 {
		item := MapItem{}

		// Decode the key into item.Key
		k := reflect.ValueOf(&item.Key).Elem()
		if d.unmarshal(n.Content[i], k) {
			// Decode the value into item.Value
			v := reflect.ValueOf(&item.Value).Elem()
			if d.unmarshal(n.Content[i+1], v) {
				// Append the key-value pair as a MapItem
				slice = append(slice, item)
			}
		}
	}

	// Set the slice in the output value
	out.Set(reflect.ValueOf(slice))
	d.stringMapType = mapType
	d.generalMapType = mapType
	return true
}

func (m MapSlice) MarshalJSON() ([]byte, error) {
	mapper, err := NewOrderedMapFromKVPairs(m)
	if err != nil {
		return nil, err
	}
	return json.Marshal(mapper)
}

func (m *MapSlice) UnmarshalJSON(data []byte) error {
	kvp := NewOrderedMap()
	err := kvp.UnmarshalJSON(data)
	if err == nil {
		*m = kvp.ToMapSlice()
	}
	return err
}

// Helper function to sort specific sections of the YAML document
func (me *MapSlice) SortKeys(keys ...string) {
	for i := range *me {
		mapItem := &(*me)[i]
		// Check if the current key matches any of the provided keys
		for _, key := range keys {
			if mapItem.Key == key {
				if valueNode, ok := mapItem.Value.(MapSlice); ok {
					// Sort the map slice by keys
					sorted := sortMapSlice(valueNode)
					mapItem.Value = sorted
				}
			}
		}
	}
}

// Helper function to sort a yaml.MapSlice by its keys
func sortMapSlice(m MapSlice) MapSlice {
	// Create a slice to hold the keys
	keys := make([]string, len(m))
	for i, item := range m {
		keys[i] = item.Key.(string)
	}

	// Sort the keys alphabetically
	sort.Strings(keys)

	// Create a new yaml.MapSlice to hold the sorted items
	var sortedMap MapSlice
	for _, key := range keys {
		for _, item := range m {
			if item.Key == key {
				sortedMap = append(sortedMap, item)
				break
			}
		}
	}
	return sortedMap
}
