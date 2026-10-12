package agentprotocol

import "strings"

func toolName(item Object) string {
	name := text(item["name"])
	namespace := text(item["namespace"])
	if namespace != "" {
		return namespace + "__" + name
	}
	return name
}

func chatTools(source []any, protocol string) ([]any, map[string]string, error) {
	tools := []any{}
	kinds := map[string]string{}
	add := func(t Object, namespace string) error {
		kind, name := text(t["type"]), text(t["name"])
		parameters := t["parameters"]
		if protocol == "messages" {
			parameters = t["input_schema"]
			if kind == "" {
				kind = "function"
			}
		}
		if kind == "custom" {
			parameters = Object{"type": "object", "properties": Object{"input": Object{"type": "string"}}, "required": []string{"input"}}
		} else if kind != "function" {
			return ErrUnsupported
		}
		flat := name
		if namespace != "" {
			flat = namespace + "__" + name
		}
		if len(flat) < 1 || len(flat) > 128 || kinds[flat] != "" || strings.Contains(name, "|") {
			return ErrUnsupported
		}
		metadata := kind
		if namespace != "" {
			metadata += "|" + namespace + "|" + name
		}
		kinds[flat] = metadata
		tools = append(tools, Object{"type": "function", "function": Object{"name": flat, "description": t["description"], "parameters": parameters}})
		return nil
	}
	for _, v := range source {
		t := object(v)
		if text(t["type"]) == "namespace" {
			namespace := text(t["name"])
			if namespace == "" || strings.ContainsAny(namespace, "|.") {
				return nil, nil, ErrUnsupported
			}
			for _, entry := range array(t["tools"]) {
				if err := add(object(entry), namespace); err != nil {
					return nil, nil, err
				}
			}
		} else {
			if err := add(t, ""); err != nil {
				return nil, nil, err
			}
		}
	}
	return tools, kinds, nil
}
