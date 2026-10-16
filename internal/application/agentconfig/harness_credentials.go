package agentconfig

import (
	"gopkg.in/yaml.v3"
	"regexp"
)

var harnessReferencePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func harnessCredentialDocument(body []byte) (*yaml.Node, *yaml.Node, error) {
	root, err := yamlDocument(body, yaml.MappingNode)
	if err != nil {
		return nil, nil, err
	}
	version := yamlField(root, "version")
	// rc.7 使用扁平引用；新版会在启动时迁移此格式。保留已有布局，兼容新旧客户端。
	if version == nil {
		for i := 0; i < len(root.Content); i += 2 {
			if !harnessReferencePattern.MatchString(root.Content[i].Value) || root.Content[i+1].Tag != "!!str" || root.Content[i+1].Value == "" {
				return nil, nil, failure("invalid_config")
			}
		}
		return root, root, nil
	}
	if version.Tag != "!!int" || version.Value != "1" {
		return nil, nil, failure("invalid_config")
	}
	refs := yamlField(root, "refs")
	if refs != nil && refs.Kind != yaml.MappingNode {
		return nil, nil, failure("invalid_config")
	}
	if records := yamlField(root, "records"); records != nil && records.Kind != yaml.MappingNode {
		return nil, nil, failure("invalid_config")
	}
	if refs != nil {
		for i := 1; i < len(refs.Content); i += 2 {
			if !harnessReferencePattern.MatchString(refs.Content[i-1].Value) || refs.Content[i].Tag != "!!str" || refs.Content[i].Value == "" {
				return nil, nil, failure("invalid_config")
			}
		}
	}
	return root, refs, nil
}

func harnessCredential(body []byte, ref string) (*string, error) {
	_, refs, err := harnessCredentialDocument(body)
	if err != nil {
		return nil, err
	}
	value := yamlField(refs, ref)
	if value == nil {
		return nil, nil
	}
	result := value.Value
	return &result, nil
}

func patchHarnessCredential(body []byte, ref string, key *string) ([]byte, error) {
	root, refs, err := harnessCredentialDocument(body)
	if err != nil {
		return nil, err
	}
	if refs == nil {
		refs = &yaml.Node{Kind: yaml.MappingNode}
		yamlSet(root, "refs", refs)
	}
	var value *yaml.Node
	if key != nil {
		value = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: *key}
	}
	yamlSet(refs, ref, value)
	if refs != root {
		yamlSet(root, "version", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: "1"})
	}
	return encodeYAML(root)
}
