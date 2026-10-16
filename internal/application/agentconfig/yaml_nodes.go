package agentconfig

import (
	"bytes"
	"gopkg.in/yaml.v3"
	"io"
)

func yamlDocument(body []byte, kind yaml.Kind) (*yaml.Node, error) {
	if len(bytes.TrimSpace(body)) == 0 {
		return &yaml.Node{Kind: kind}, nil
	}
	decoder := yaml.NewDecoder(bytes.NewReader(body))
	var doc, extra yaml.Node
	if decoder.Decode(&doc) != nil || decoder.Decode(&extra) != io.EOF {
		return nil, failure("invalid_config")
	}
	if len(doc.Content) == 0 || len(doc.Content) == 1 && doc.Content[0].Tag == "!!null" {
		return &yaml.Node{Kind: kind, HeadComment: doc.HeadComment}, nil
	}
	if len(doc.Content) != 1 || doc.Content[0].Kind != kind {
		return nil, failure("invalid_config")
	}
	root := doc.Content[0]
	var validate func(*yaml.Node, int) bool
	count := 0
	validate = func(node *yaml.Node, depth int) bool {
		count++
		if depth > 48 || count > 20000 || node.Kind == yaml.AliasNode || node.Anchor != "" {
			return false
		}
		if node.Kind == yaml.MappingNode {
			seen := map[string]bool{}
			for i := 0; i < len(node.Content); i += 2 {
				k := node.Content[i]
				if k.Kind != yaml.ScalarNode || k.Tag != "!!str" || seen[k.Value] || k.Value == "<<" {
					return false
				}
				seen[k.Value] = true
			}
		}
		for _, child := range node.Content {
			if !validate(child, depth+1) {
				return false
			}
		}
		return true
	}
	if !validate(root, 0) {
		return nil, failure("invalid_config")
	}
	return root, nil
}

func yamlField(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func yamlSet(node *yaml.Node, key string, value *yaml.Node) {
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			if value == nil {
				node.Content = append(node.Content[:i], node.Content[i+2:]...)
			} else {
				node.Content[i+1] = value
			}
			return
		}
	}
	if value != nil {
		node.Content = append(node.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, value)
	}
}

func yamlValue(value any) (*yaml.Node, error) {
	var node yaml.Node
	if err := node.Encode(value); err != nil {
		return nil, failure("invalid_config")
	}
	return &node, nil
}

func encodeYAML(node *yaml.Node) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)
	if encoder.Encode(node) != nil || encoder.Close() != nil {
		return nil, failure("invalid_config")
	}
	return buffer.Bytes(), nil
}
