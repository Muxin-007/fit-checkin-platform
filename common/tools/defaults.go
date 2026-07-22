package tools

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	errInvalidType = errors.New("not a struct pointer")
)

const (
	fieldName = "default"
)

// SetDefault Set initializes members in a struct referenced by a pointer.
// Maps and slices are initialized by `make` and other primitive types are set with default values.
// `ptr` should be a struct pointer
func SetDefault(ptr any) error {
	if reflect.TypeOf(ptr).Kind() != reflect.Ptr {
		return errInvalidType
	}

	v := reflect.ValueOf(ptr).Elem()
	t := v.Type()

	if t.Kind() != reflect.Struct {
		return errInvalidType
	}

	for i := 0; i < t.NumField(); i++ {
		if defaultVal := t.Field(i).Tag.Get(fieldName); defaultVal != "-" {
			if err := setField(v.Field(i), defaultVal); err != nil {
				return err
			}
		}
	}
	callSetter(ptr)
	return nil
}

func setField(field reflect.Value, defaultVal string) error {
	if !field.CanSet() {
		return nil
	}

	if !shouldInitializeField(field, defaultVal) {
		return nil
	}

	isInitial := isInitialValue(field)
	if isInitial {
		switch field.Kind() {
		case reflect.Bool:
			if val, err := strconv.ParseBool(defaultVal); err == nil {
				field.Set(reflect.ValueOf(val).Convert(field.Type()))
			}
		case reflect.Int:
			if val, err := strconv.ParseInt(defaultVal, 0, strconv.IntSize); err == nil {
				field.Set(reflect.ValueOf(int(val)).Convert(field.Type()))
			}
		case reflect.Int8:
			if val, err := strconv.ParseInt(defaultVal, 0, 8); err == nil {
				field.Set(reflect.ValueOf(int8(val)).Convert(field.Type()))
			}
		case reflect.Int16:
			if val, err := strconv.ParseInt(defaultVal, 0, 16); err == nil {
				field.Set(reflect.ValueOf(int16(val)).Convert(field.Type()))
			}
		case reflect.Int32:
			if val, err := strconv.ParseInt(defaultVal, 0, 32); err == nil {
				field.Set(reflect.ValueOf(int32(val)).Convert(field.Type()))
			}
		case reflect.Int64:
			if val, err := time.ParseDuration(defaultVal); err == nil {
				field.Set(reflect.ValueOf(val).Convert(field.Type()))
			} else if val, err := strconv.ParseInt(defaultVal, 0, 64); err == nil {
				field.Set(reflect.ValueOf(val).Convert(field.Type()))
			}
		case reflect.Uint:
			if val, err := strconv.ParseUint(defaultVal, 0, strconv.IntSize); err == nil {
				field.Set(reflect.ValueOf(uint(val)).Convert(field.Type()))
			}
		case reflect.Uint8:
			if val, err := strconv.ParseUint(defaultVal, 0, 8); err == nil {
				field.Set(reflect.ValueOf(uint8(val)).Convert(field.Type()))
			}
		case reflect.Uint16:
			if val, err := strconv.ParseUint(defaultVal, 0, 16); err == nil {
				field.Set(reflect.ValueOf(uint16(val)).Convert(field.Type()))
			}
		case reflect.Uint32:
			if val, err := strconv.ParseUint(defaultVal, 0, 32); err == nil {
				field.Set(reflect.ValueOf(uint32(val)).Convert(field.Type()))
			}
		case reflect.Uint64:
			if val, err := strconv.ParseUint(defaultVal, 0, 64); err == nil {
				field.Set(reflect.ValueOf(val).Convert(field.Type()))
			}
		case reflect.Uintptr:
			if val, err := strconv.ParseUint(defaultVal, 0, strconv.IntSize); err == nil {
				field.Set(reflect.ValueOf(uintptr(val)).Convert(field.Type()))
			}
		case reflect.Float32:
			if val, err := strconv.ParseFloat(defaultVal, 32); err == nil {
				field.Set(reflect.ValueOf(float32(val)).Convert(field.Type()))
			}
		case reflect.Float64:
			if val, err := strconv.ParseFloat(defaultVal, 64); err == nil {
				field.Set(reflect.ValueOf(val).Convert(field.Type()))
			}
		case reflect.String:
			field.Set(reflect.ValueOf(defaultVal).Convert(field.Type()))

		case reflect.Slice:
			ref := reflect.New(field.Type())
			ref.Elem().Set(reflect.MakeSlice(field.Type(), 0, 0))
			if defaultVal != "" && defaultVal != "[]" {
				if err := json.Unmarshal([]byte(defaultVal), ref.Interface()); err != nil {
					return err
				}
			}
			field.Set(ref.Elem().Convert(field.Type()))
		case reflect.Map:
			ref := reflect.New(field.Type())
			ref.Elem().Set(reflect.MakeMap(field.Type()))
			if defaultVal != "" && defaultVal != "{}" {
				if err := json.Unmarshal([]byte(defaultVal), ref.Interface()); err != nil {
					return err
				}
			}
			field.Set(ref.Elem().Convert(field.Type()))
		case reflect.Struct:
			if defaultVal != "" && defaultVal != "{}" {
				if err := json.Unmarshal([]byte(defaultVal), field.Addr().Interface()); err != nil {
					return err
				}
			}
		case reflect.Ptr:
			field.Set(reflect.New(field.Type().Elem()))
		default:
		}
	}

	switch field.Kind() {
	case reflect.Ptr:
		if isInitial || field.Elem().Kind() == reflect.Struct {
			var _ = setField(field.Elem(), defaultVal)
			callSetter(field.Interface())
		}
	case reflect.Struct:
		if err := SetDefault(field.Addr().Interface()); err != nil {
			return err
		}
	case reflect.Slice:
		for j := 0; j < field.Len(); j++ {
			if err := setField(field.Index(j), defaultVal); err != nil {
				return err
			}
		}
	default:
	}

	return nil
}

func isInitialValue(field reflect.Value) bool {
	return reflect.DeepEqual(reflect.Zero(field.Type()).Interface(), field.Interface())
}

func shouldInitializeField(field reflect.Value, tag string) bool {
	switch field.Kind() {
	case reflect.Struct:
		return true
	case reflect.Ptr:
		if !field.IsNil() && field.Elem().Kind() == reflect.Struct {
			return true
		}
	case reflect.Slice:
		return field.Len() > 0 || tag != ""
	default:
	}

	return tag != ""
}

// Setter is an interface for setting default values
type Setter interface {
	SetDefaults()
}

func callSetter(v interface{}) {
	if ds, ok := v.(Setter); ok {
		ds.SetDefaults()
	}
}

// **递归构建 YAML 结构，添加注释**
func buildYAMLNode(val reflect.Value, typ reflect.Type, parentPath string) *yaml.Node {
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return &yaml.Node{Kind: yaml.ScalarNode, Value: "null"}
		}
		val = val.Elem()
		typ = typ.Elem()
	}

	// 如果不是结构体，直接返回标量节点
	if val.Kind() != reflect.Struct {
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: fmt.Sprintf("%v", val.Interface()),
		}
	}

	root := &yaml.Node{Kind: yaml.MappingNode}

	// 使用 for range 遍历结构体字段
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		yamlTag := field.Tag.Get("yaml")
		comment := field.Tag.Get("comment")
		fieldVal := val.Field(i)

		// 跳过未定义 YAML tag 或 comment 的字段
		if yamlTag == "" || yamlTag == "-" || comment == "" {
			continue
		}

		// 构建当前路径
		currentPath := yamlTag
		if parentPath != "" {
			currentPath = parentPath + "." + yamlTag
		}

		// 添加 Key 并设置注释
		keyNode := &yaml.Node{
			Kind:        yaml.ScalarNode,
			Value:       yamlTag,
			LineComment: comment,
		}

		// 处理不同类型的值
		var valueNode *yaml.Node
		switch fieldVal.Kind() {
		case reflect.Struct:
			valueNode = buildYAMLNode(fieldVal, field.Type, currentPath)
		case reflect.Ptr:
			if fieldVal.IsNil() {
				valueNode = &yaml.Node{Kind: yaml.ScalarNode, Value: "null"}
			} else {
				valueNode = buildYAMLNode(fieldVal.Elem(), field.Type.Elem(), currentPath)
			}
		case reflect.Slice, reflect.Array:
			listNode := &yaml.Node{Kind: yaml.SequenceNode}
			// 使用 for range 遍历切片/数组
			for j := 0; j < fieldVal.Len(); j++ {
				item := fieldVal.Index(j)
				if item.Kind() == reflect.Ptr && !item.IsNil() {
					item = item.Elem()
				}
				elemType := field.Type.Elem()
				if elemType.Kind() == reflect.Ptr {
					elemType = elemType.Elem()
				}
				itemNode := buildYAMLNode(item, elemType, fmt.Sprintf("%s[%d]", currentPath, j))
				listNode.Content = append(listNode.Content, itemNode)
			}
			valueNode = listNode
		case reflect.Map:
			mapNode := &yaml.Node{Kind: yaml.MappingNode}
			// 使用 for range 遍历映射
			for _, key := range fieldVal.MapKeys() {
				keyStr := fmt.Sprintf("%v", key.Interface())
				keyNode := &yaml.Node{
					Kind:  yaml.ScalarNode,
					Value: keyStr,
				}
				value := fieldVal.MapIndex(key)
				if value.Kind() == reflect.Ptr && !value.IsNil() {
					value = value.Elem()
				}
				valueNode := buildYAMLNode(value, field.Type.Elem(), fmt.Sprintf("%s.%s", currentPath, keyStr))
				mapNode.Content = append(mapNode.Content, keyNode, valueNode)
			}
			valueNode = mapNode
		default:
			valueNode = &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: fmt.Sprintf("%v", fieldVal.Interface()),
			}
		}

		// 追加 Key-Value
		root.Content = append(root.Content, keyNode, valueNode)
	}

	return root
}

// **生成 YAML 并格式化**
func GenerateYAMLWithComments(config any) ([]byte, error) {
	val := reflect.ValueOf(config)
	typ := reflect.TypeOf(config)

	// 确保传入的是结构体或指针
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
		typ = typ.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected a struct, but got %v", val.Kind())
	}

	// 处理 YAML
	root := buildYAMLNode(val, typ, "")

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2) // 设置缩进，提高可读性
	defer encoder.Close()

	if err := encoder.Encode(root); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
