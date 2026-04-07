package openapi

import (
	"path"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
)

type OperationSpec struct {
	Method        string
	Path          string
	Query         any
	RequestBody   any
	ResponseBody  any
	SuccessStatus int
}

type schemaBuilder struct {
	schemas        map[string]any
	componentNames map[reflect.Type]string
	typesByName    map[string]reflect.Type
}

func newSchemaBuilder(schemas map[string]any) *schemaBuilder {
	return &schemaBuilder{
		schemas:        schemas,
		componentNames: map[reflect.Type]string{},
		typesByName:    map[string]reflect.Type{},
	}
}

func buildOperationSpecIndex(specs []OperationSpec) map[string]OperationSpec {
	if len(specs) == 0 {
		specs = defaultOperationSpecs()
	}

	index := make(map[string]OperationSpec, len(specs))
	for _, spec := range specs {
		index[operationSpecKey(spec.Method, spec.Path)] = spec
	}
	return index
}

func operationSpecKey(method, path string) string {
	return strings.ToUpper(strings.TrimSpace(method)) + " " + strings.TrimSpace(path)
}

func applyOperationSpec(op map[string]any, spec OperationSpec, builder *schemaBuilder) {
	if spec.RequestBody != nil {
		op["requestBody"] = map[string]any{
			"required": true,
			"content": map[string]any{
				"application/json": map[string]any{
					"schema": builder.schemaForValue(spec.RequestBody),
				},
			},
		}
	}

	if spec.Query != nil {
		parameters, _ := op["parameters"].([]any)
		parameters = append(parameters, queryParametersForValue(spec.Query)...)
		op["parameters"] = parameters
	}

	responses := op["responses"].(map[string]any)
	successStatus := spec.SuccessStatus
	if successStatus == 0 {
		successStatus = 200
	}
	delete(responses, "200")

	successResponse := map[string]any{
		"description": responseDescription(successStatus),
	}
	if spec.ResponseBody != nil && successStatus != 204 {
		successResponse["content"] = map[string]any{
			"application/json": map[string]any{
				"schema": builder.responseEnvelopeSchemaForValue(spec.ResponseBody),
			},
		}
	}

	responses[statusCodeKey(successStatus)] = successResponse
}

func statusCodeKey(status int) string {
	return strconv.Itoa(status)
}

func responseDescription(status int) string {
	switch status {
	case 201:
		return "Created"
	case 204:
		return "No Content"
	default:
		return "OK"
	}
}

func queryParametersForValue(value any) []any {
	typeOfValue := indirectType(reflect.TypeOf(value))
	if typeOfValue == nil || typeOfValue.Kind() != reflect.Struct {
		return nil
	}

	parameters := make([]any, 0, typeOfValue.NumField())
	for i := 0; i < typeOfValue.NumField(); i++ {
		field := typeOfValue.Field(i)
		if !field.IsExported() {
			continue
		}

		name, ok := field.Tag.Lookup("form")
		if !ok || name == "" || name == "-" {
			continue
		}

		parameters = append(parameters, map[string]any{
			"name":     name,
			"in":       "query",
			"required": false,
			"schema":   inlineSchemaForType(indirectType(field.Type)),
		})
	}

	return parameters
}

func (b *schemaBuilder) responseEnvelopeSchemaForValue(value any) map[string]any {
	typeOfValue := indirectType(reflect.TypeOf(value))
	name := "ResponseOf" + b.schemaDescriptorName(typeOfValue)
	if _, ok := b.schemas[name]; !ok {
		b.schemas[name] = map[string]any{
			"type": "object",
			"properties": map[string]any{
				"data": b.schemaForType(typeOfValue),
			},
			"required": []string{"data"},
		}
	}
	return map[string]any{"$ref": "#/components/schemas/" + name}
}

func (b *schemaBuilder) schemaForValue(value any) map[string]any {
	return b.schemaForType(indirectType(reflect.TypeOf(value)))
}

func (b *schemaBuilder) schemaForType(typeOfValue reflect.Type) map[string]any {
	if typeOfValue == nil {
		return map[string]any{}
	}

	if shouldUseComponentSchema(typeOfValue) {
		name := b.componentName(typeOfValue)
		if _, ok := b.schemas[name]; !ok {
			b.schemas[name] = inlineSchemaForStruct(typeOfValue, b)
		}
		return map[string]any{"$ref": "#/components/schemas/" + name}
	}

	return inlineSchemaForTypeWithBuilder(typeOfValue, b)
}

func (b *schemaBuilder) componentName(typeOfValue reflect.Type) string {
	if existing, ok := b.componentNames[typeOfValue]; ok {
		return existing
	}

	baseName := exportedName(typeOfValue.Name())
	if baseName == "" {
		baseName = "AnonymousObject"
	}

	componentName := baseName
	if existingType, ok := b.typesByName[componentName]; ok && existingType != typeOfValue {
		componentName = exportedName(path.Base(typeOfValue.PkgPath())) + baseName
	}

	b.componentNames[typeOfValue] = componentName
	b.typesByName[componentName] = typeOfValue
	return componentName
}

func (b *schemaBuilder) schemaDescriptorName(typeOfValue reflect.Type) string {
	typeOfValue = indirectType(typeOfValue)
	if typeOfValue == nil {
		return "Empty"
	}

	switch typeOfValue.Kind() {
	case reflect.Slice, reflect.Array:
		return b.schemaDescriptorName(typeOfValue.Elem()) + "List"
	case reflect.Map:
		return "MapOf" + b.schemaDescriptorName(typeOfValue.Elem())
	case reflect.Struct:
		if shouldUseComponentSchema(typeOfValue) {
			return b.componentName(typeOfValue)
		}
		return "Object"
	case reflect.String:
		return "String"
	case reflect.Bool:
		return "Boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "Integer"
	case reflect.Float32, reflect.Float64:
		return "Number"
	default:
		return "Value"
	}
}

func inlineSchemaForType(typeOfValue reflect.Type) map[string]any {
	return inlineSchemaForTypeWithBuilder(typeOfValue, nil)
}

func inlineSchemaForTypeWithBuilder(typeOfValue reflect.Type, builder *schemaBuilder) map[string]any {
	typeOfValue = indirectType(typeOfValue)
	if typeOfValue == nil {
		return map[string]any{}
	}

	if typeOfValue == reflect.TypeOf(time.Time{}) {
		return map[string]any{
			"type":   "string",
			"format": "date-time",
		}
	}

	switch typeOfValue.Kind() {
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Slice, reflect.Array:
		itemType := indirectType(typeOfValue.Elem())
		itemSchema := inlineSchemaForTypeWithBuilder(itemType, builder)
		if builder != nil && shouldUseComponentSchema(itemType) {
			itemSchema = builder.schemaForType(itemType)
		}
		return map[string]any{
			"type":  "array",
			"items": itemSchema,
		}
	case reflect.Map:
		schema := map[string]any{"type": "object"}
		elemType := indirectType(typeOfValue.Elem())
		if elemType != nil {
			additionalProperties := inlineSchemaForTypeWithBuilder(elemType, builder)
			if builder != nil && shouldUseComponentSchema(elemType) {
				additionalProperties = builder.schemaForType(elemType)
			}
			schema["additionalProperties"] = additionalProperties
		}
		return schema
	case reflect.Struct:
		if builder != nil && shouldUseComponentSchema(typeOfValue) {
			return builder.schemaForType(typeOfValue)
		}
		return inlineSchemaForStruct(typeOfValue, builder)
	case reflect.Interface:
		return map[string]any{}
	default:
		return map[string]any{}
	}
}

func inlineSchemaForStruct(typeOfValue reflect.Type, builder *schemaBuilder) map[string]any {
	properties := map[string]any{}
	required := make([]string, 0)

	for i := 0; i < typeOfValue.NumField(); i++ {
		field := typeOfValue.Field(i)
		if !field.IsExported() {
			continue
		}

		name, omitEmpty, ok := jsonFieldName(field)
		if !ok {
			continue
		}

		fieldType := indirectType(field.Type)
		fieldSchema := inlineSchemaForTypeWithBuilder(fieldType, builder)
		if builder != nil && shouldUseComponentSchema(fieldType) {
			fieldSchema = builder.schemaForType(fieldType)
		}
		properties[name] = fieldSchema

		if isRequiredField(field, omitEmpty) {
			required = append(required, name)
		}
	}

	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		slices.Sort(required)
		schema["required"] = required
	}
	return schema
}

func shouldUseComponentSchema(typeOfValue reflect.Type) bool {
	typeOfValue = indirectType(typeOfValue)
	return typeOfValue != nil && typeOfValue.Kind() == reflect.Struct && typeOfValue.Name() != "" && typeOfValue != reflect.TypeOf(time.Time{})
}

func indirectType(typeOfValue reflect.Type) reflect.Type {
	for typeOfValue != nil && typeOfValue.Kind() == reflect.Pointer {
		typeOfValue = typeOfValue.Elem()
	}
	return typeOfValue
}

func jsonFieldName(field reflect.StructField) (string, bool, bool) {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return "", false, false
	}
	if tag == "" {
		return field.Name, false, true
	}

	parts := strings.Split(tag, ",")
	name := parts[0]
	if name == "" {
		name = field.Name
	}

	return name, slices.Contains(parts[1:], "omitempty"), true
}

func isRequiredField(field reflect.StructField, omitEmpty bool) bool {
	return strings.Contains(field.Tag.Get("binding"), "required") && !omitEmpty
}

func exportedName(name string) string {
	if name == "" {
		return ""
	}
	return strings.ToUpper(name[:1]) + name[1:]
}
