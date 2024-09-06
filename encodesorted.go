package yaml

import (
	"encoding/json"
	"fmt"
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

// Helper function to sort specific sections of the YAML document by keys, including nested keys.
func (me *MapSlice) SortKeys(keys ...string) {
	for i := range *me {
		mapItem := &(*me)[i]
		// Check if the current key matches any of the provided keys
		for _, key := range keys {
			if mapItem.Key == key {
				// Check if the value is already a MapSlice
				if valueNode, ok := mapItem.Value.(MapSlice); ok {
					// Sort the map slice by keys
					sorted := sortMapSlice(valueNode)
					mapItem.Value = sorted

					// Recursively sort nested maps by keys if needed
					sorted.SortKeys(keys...)
				} else if node, ok := mapItem.Value.(*Node); ok {
					// If it's a Node, try to convert to MapSlice
					converted, err := nodeToMapSlice(node)
					if err == nil {
						// Sort the newly converted MapSlice by keys
						sorted := sortMapSlice(converted)
						mapItem.Value = sorted

						// Recursively sort nested maps by keys if needed
						sorted.SortKeys(keys...)
					}
				}
			}
		}
	}
}

// Helper function to sort a MapSlice by its keys.
func sortMapSlice(m MapSlice) MapSlice {
	// Sort the MapSlice by key
	sorted := make(MapSlice, len(m))
	copy(sorted, m)

	// Use standard Go sorting to order by key
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Key.(string) < sorted[j].Key.(string)
	})

	return sorted
}

// nodeToMapSlice converts a *Node to a MapSlice, if the node is a mapping
func nodeToMapSlice(node *Node) (MapSlice, error) {
	if node.Kind != MappingNode {
		return nil, fmt.Errorf("expected mapping node, got %v", node.Kind)
	}

	var mapSlice MapSlice
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		if keyNode.Kind != ScalarNode {
			return nil, fmt.Errorf("expected scalar key, got %v", keyNode.Kind)
		}

		// Recursively convert the value node
		var value interface{}
		switch valueNode.Kind {
		case MappingNode:
			converted, err := nodeToMapSlice(valueNode)
			if err != nil {
				return nil, err
			}
			value = converted
		case ScalarNode:
			value = valueNode.Value
		case SequenceNode:
			var seq []interface{}
			for _, item := range valueNode.Content {
				seq = append(seq, item.Value)
			}
			value = seq
		default:
			value = valueNode.Value
		}

		mapSlice = append(mapSlice, MapItem{
			Key:   keyNode.Value,
			Value: value,
		})
	}
	return mapSlice, nil
}
