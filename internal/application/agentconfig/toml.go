package agentconfig

import (
	"bytes"
	"github.com/pelletier/go-toml/v2"
	"github.com/pelletier/go-toml/v2/unstable"
	"sort"
	"strings"
)

type expression struct {
	start, end   int
	key, section string
	table        bool
}
type edit struct {
	start, end int
	body       []byte
}

// patchTOML 以官方语法解析位置，只替换受管字段；保留其他字段、注释和表顺序。
func patchTOML(body []byte, provider string, updates map[string]string) ([]byte, error) {
	for _, r := range provider {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
			return nil, failure("unsupported_config")
		}
	}
	parser := unstable.Parser{}
	parser.Reset(body)
	expressions := []expression{}
	section := ""
	for parser.NextExpression() {
		node := parser.Expression()
		if node.Kind != unstable.KeyValue && node.Kind != unstable.Table && node.Kind != unstable.ArrayTable {
			continue
		}
		key := []string{}
		iterator := node.Key()
		offset := -1
		for iterator.Next() {
			part := iterator.Node()
			key = append(key, string(part.Data))
			if offset < 0 {
				offset = int(part.Raw.Offset)
			}
		}
		start := bytes.LastIndexByte(body[:offset], '\n') + 1
		table := node.Kind != unstable.KeyValue
		if table {
			section = strings.Join(key, ".")
		}
		name := strings.Join(key, ".")
		if !table && section != "" {
			name = section + "." + name
		}
		if len(expressions) > 0 {
			expressions[len(expressions)-1].end = start
		}
		expressions = append(expressions, expression{start, len(body), name, section, table})
	}
	if parser.Error() != nil {
		return nil, failure("invalid_config")
	}
	edits := []edit{}
	present := map[string]bool{}
	sectionEnds := map[string]int{}
	lastSection := ""
	for _, e := range expressions {
		if e.table {
			if lastSection != "" {
				sectionEnds[lastSection] = e.start
			}
			lastSection = e.section
			sectionEnds[e.section] = len(body)
			continue
		}
		if replacement, ok := updates[e.key]; ok {
			present[e.key] = true
			line := ""
			if replacement != "" {
				key := e.key
				if e.section != "" {
					key = strings.TrimPrefix(key, e.section+".")
				}
				line = key + " = " + replacement + "\n"
			}
			// 保留字段后方的空行／注释；复杂多行值通过下一表达式边界整体替换。
			original := body[e.start:e.end]
			end := len(original)
			for end > 0 {
				idx := bytes.LastIndexByte(original[:end-1], '\n')
				start := idx + 1
				part := strings.TrimSpace(string(original[start:end]))
				if part == "" || strings.HasPrefix(part, "#") {
					end = start
				} else {
					break
				}
			}
			if end == 0 {
				end = len(original)
			}
			edits = append(edits, edit{e.start, e.start + end, []byte(line)})
		}
	}
	root := []string{}
	groups := map[string][]string{}
	keys := []string{}
	for key := range updates {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := updates[key]
		if present[key] || value == "" {
			continue
		}
		if dot := strings.LastIndex(key, "."); dot >= 0 {
			group := key[:dot]
			groups[group] = append(groups[group], key[dot+1:]+" = "+value)
		} else {
			root = append(root, key+" = "+value)
		}
	}
	groupNames := []string{}
	for name := range groups {
		groupNames = append(groupNames, name)
	}
	sort.Strings(groupNames)
	for _, group := range groupNames {
		addition := strings.Join(groups[group], "\n") + "\n"
		end, exists := sectionEnds[group]
		if !exists {
			end = len(body)
			addition = "\n[" + group + "]\n" + addition
		}
		edits = append(edits, edit{end, end, []byte("\n" + addition)})
	}
	if len(root) > 0 {
		edits = append(edits, edit{0, 0, []byte(strings.Join(root, "\n") + "\n")})
	}
	sort.SliceStable(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
	result := append([]byte(nil), body...)
	for _, e := range edits {
		result = append(result[:e.start], append(e.body, result[e.end:]...)...)
	}
	var check map[string]any
	if toml.Unmarshal(result, &check) != nil {
		return nil, failure("unsupported_config")
	}
	return result, nil
}
