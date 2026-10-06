// tools/gen — Weft Protocol schema → multi-language type generator.
//
// Reads protocol/schema/weftlink.yaml (single source of truth) and emits
// typed models for Go, Dart and ArkTS into gen/. Stdlib only.
//
// Usage:
//
//	go run ./tools/gen            # generate all
//	go run ./tools/gen -check     # verify gen/ matches schema (CI)
package main

import "fmt"
import "go/format"
import "os"
import "path/filepath"
import "sort"
import "strings"

// -- Model -------------------------------------------------------------------

type Field struct {
	Name     string
	Type     string
	Required bool
	Enum     []string
	Desc     string
}

type Entity struct {
	Name        string
	Description string
	Fields      []Field
}

type Schema struct {
	Version  string
	Types    []Entity
	Messages []Entity
}

// -- YAML subset parser ------------------------------------------------------
//
// Supports exactly the shape used by protocol/schema/weftlink.yaml:
//   key: scalar
//   key:
//     child:
//       field: { type: T, required: bool, enum: [a, b] }
// Comments (#) and document markers (---) are ignored.

func stripComment(line string) string {
	var inQuote = false
	var out []rune
	for _, r := range line {
		if r == '"' {
			inQuote = !inQuote
		}
		if r == '#' && !inQuote {
			break
		}
		out = append(out, r)
	}
	return strings.TrimRight(string(out), " \t\r")
}

func indentOf(line string) int {
	var n = 0
	for _, r := range line {
		if r == ' ' {
			n++
			continue
		}
		if r == '\t' {
			n += 2
			continue
		}
		break
	}
	return n
}

// parseSchema builds a Schema from the YAML subset.
func parseSchema(path string, sc *Schema) error {
	var raw []byte
	var err error
	raw, err = os.ReadFile(path)
	if err != nil {
		return err
	}
	var lines = strings.Split(string(raw), "\n")

	var section = ""
	var cur *Entity = nil
	var inFields = false

	for _, rawLine := range lines {
		var line = stripComment(rawLine)
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.TrimSpace(line) == "---" {
			continue
		}
		var ind = indentOf(line)
		var body = strings.TrimSpace(line)

		if ind == 0 {
			// top-level key
			if strings.HasPrefix(body, "version:") {
				sc.Version = unquote(strings.TrimSpace(strings.TrimPrefix(body, "version:")))
				continue
			}
			if body == "types:" {
				section = "types"
				cur = nil
				continue
			}
			if body == "messages:" {
				section = "messages"
				cur = nil
				continue
			}
			continue
		}

		if ind == 2 {
			// entity name
			if !strings.HasSuffix(body, ":") {
				continue
			}
			var name = strings.TrimSuffix(body, ":")
			var e = Entity{Name: name}
			if section == "types" {
				sc.Types = append(sc.Types, e)
				cur = &sc.Types[len(sc.Types)-1]
			} else if section == "messages" {
				sc.Messages = append(sc.Messages, e)
				cur = &sc.Messages[len(sc.Messages)-1]
			}
			inFields = false
			continue
		}

		if ind == 4 {
			if cur == nil {
				continue
			}
			if strings.HasPrefix(body, "description:") {
				cur.Description = unquote(strings.TrimSpace(strings.TrimPrefix(body, "description:")))
				continue
			}
			if body == "fields:" {
				inFields = true
				continue
			}
			continue
		}

		if ind >= 6 {
			if cur == nil || !inFields {
				continue
			}
			var colon = strings.Index(body, ":")
			if colon < 0 {
				continue
			}
			var fname = strings.TrimSpace(body[:colon])
			var rest = strings.TrimSpace(body[colon+1:])
			var f = Field{Name: fname}
			parseFlowMap(rest, &f)
			cur.Fields = append(cur.Fields, f)
		}
	}
	return nil
}

// parseFlowMap parses `{ type: T, required: bool, enum: [a, b] }`.
func parseFlowMap(s string, f *Field) {
	var inner = strings.TrimSpace(s)
	inner = strings.TrimPrefix(inner, "{")
	inner = strings.TrimSuffix(inner, "}")
	var parts = splitTopLevel(inner, ',')
	for _, p := range parts {
		var p2 = strings.TrimSpace(p)
		if p2 == "" {
			continue
		}
		var ci = strings.Index(p2, ":")
		if ci < 0 {
			continue
		}
		var k = strings.TrimSpace(p2[:ci])
		var v = strings.TrimSpace(p2[ci+1:])
		switch k {
		case "type":
			f.Type = unquote(v)
		case "required":
			f.Required = v == "true"
		case "enum":
			f.Enum = parseList(v)
		case "description":
			f.Desc = unquote(v)
		}
	}
}

// splitTopLevel splits on sep, ignoring separators inside [] or {} or "".
func splitTopLevel(s string, sep rune) []string {
	var depth = 0
	var inQuote = false
	var cur []rune
	var out []string
	for _, r := range s {
		if r == '"' {
			inQuote = !inQuote
		}
		if !inQuote {
			if r == '[' || r == '{' {
				depth++
			} else if r == ']' || r == '}' {
				depth--
			}
		}
		if r == sep && depth == 0 && !inQuote {
			out = append(out, string(cur))
			cur = nil
			continue
		}
		cur = append(cur, r)
	}
	out = append(out, string(cur))
	return out
}

func parseList(s string) []string {
	var inner = strings.TrimSpace(s)
	inner = strings.TrimPrefix(inner, "[")
	inner = strings.TrimSuffix(inner, "]")
	var parts = splitTopLevel(inner, ',')
	var out []string
	for _, p := range parts {
		var v = unquote(strings.TrimSpace(p))
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func unquote(s string) string {
	var t = strings.TrimSpace(s)
	t = strings.TrimPrefix(t, "\"")
	t = strings.TrimSuffix(t, "\"")
	t = strings.TrimPrefix(t, "'")
	t = strings.TrimSuffix(t, "'")
	return t
}

// -- Identifier safety -------------------------------------------------------

var dartKeywords = map[string]bool{
	"abstract": true, "as": true, "assert": true, "async": true,
	"await": true, "break": true, "case": true, "catch": true,
	"class": true, "const": true, "continue": true, "covariant": true,
	"default": true, "deferred": true, "do": true, "dynamic": true,
	"else": true, "enum": true, "export": true, "extends": true,
	"extension": true, "external": true, "factory": true, "false": true,
	"final": true, "finally": true, "for": true, "get": true,
	"hide": true, "if": true, "implements": true, "import": true,
	"in": true, "interface": true, "is": true, "late": true,
	"library": true, "mixin": true, "new": true, "null": true,
	"on": true, "operator": true, "part": true, "required": true,
	"rethrow": true, "return": true, "set": true, "show": true,
	"static": true, "super": true, "switch": true, "sync": true,
	"this": true, "throw": true, "true": true, "try": true,
	"typedef": true, "var": true, "void": true, "while": true,
	"with": true, "yield": true,
}

// safeDart returns a Dart-safe identifier (escapes reserved words).
func safeDart(name string) string {
	if dartKeywords[name] {
		return name + "_"
	}
	return name
}

// -- Type mapping ------------------------------------------------------------

func goType(t string) string {
	if strings.HasPrefix(t, "list<") && strings.HasSuffix(t, ">") {
		var inner = t[5 : len(t)-1]
		return "[]" + goType(inner)
	}
	switch t {
	case "string":
		return "string"
	case "bool":
		return "bool"
	case "uint16":
		return "uint16"
	case "uint32":
		return "uint32"
	case "uint64":
		return "uint64"
	case "bytes":
		return "[]byte"
	default:
		return t // entity reference
	}
}

func dartType(t string) string {
	if strings.HasPrefix(t, "list<") && strings.HasSuffix(t, ">") {
		var inner = t[5 : len(t)-1]
		return "List<" + dartType(inner) + ">"
	}
	switch t {
	case "string":
		return "String"
	case "bool":
		return "bool"
	case "uint16", "uint32", "uint64":
		return "int"
	case "bytes":
		return "List<int>"
	default:
		return t
	}
}

func arktsType(t string) string {
	if strings.HasPrefix(t, "list<") && strings.HasSuffix(t, ">") {
		var inner = t[5 : len(t)-1]
		return arktsType(inner) + "[]"
	}
	switch t {
	case "string":
		return "string"
	case "bool":
		return "boolean"
	case "uint16", "uint32", "uint64":
		return "number"
	case "bytes":
		return "ArrayBuffer"
	default:
		return t
	}
}

func goFieldName(s string) string {
	var parts = strings.Split(s, "_")
	var out = ""
	for _, p := range parts {
		if p == "" {
			continue
		}
		out += strings.ToUpper(p[:1]) + p[1:]
	}
	return out
}

// -- Emitters ----------------------------------------------------------------

func emitGo(sc *Schema) string {
	var b strings.Builder
	b.WriteString("// Code generated by tools/gen from protocol/schema/weftlink.yaml. DO NOT EDIT.\n")
	b.WriteString("// Schema version: " + sc.Version + "\n")
	b.WriteString("package gen\n\n")

	b.WriteString("// -- Types -------------------------------------------------------------------\n\n")
	for _, e := range sc.Types {
		if e.Description != "" {
			b.WriteString("// " + e.Name + " — " + e.Description + "\n")
		}
		b.WriteString("type " + e.Name + " struct {\n")
		for _, f := range e.Fields {
			var line = "\t" + goFieldName(f.Name) + " " + goType(f.Type)
			line += " `json:\"" + f.Name + "\"`"
			if f.Desc != "" {
				line += " // " + f.Desc
			}
			b.WriteString(line + "\n")
		}
		b.WriteString("}\n\n")
	}

	b.WriteString("// -- Messages ----------------------------------------------------------------\n\n")
	for _, e := range sc.Messages {
		if e.Description != "" {
			b.WriteString("// " + e.Name + " — " + e.Description + "\n")
		}
		b.WriteString("type " + e.Name + " struct {\n")
		for _, f := range e.Fields {
			var line = "\t" + goFieldName(f.Name) + " " + goType(f.Type)
			line += " `json:\"" + f.Name + "\"`"
			if f.Desc != "" {
				line += " // " + f.Desc
			}
			b.WriteString(line + "\n")
		}
		b.WriteString("}\n\n")
	}
	// Normalise with gofmt so committed output is format-clean.
	var src = []byte(b.String())
	var formatted []byte
	var fErr error
	formatted, fErr = format.Source(src)
	if fErr != nil {
		return b.String()
	}
	return string(formatted)
}

func emitDart(sc *Schema) string {
	var b strings.Builder
	b.WriteString("// Code generated by tools/gen from protocol/schema/weftlink.yaml. DO NOT EDIT.\n")
	b.WriteString("// Schema version: " + sc.Version + "\n")
	b.WriteString("// ignore_for_file: non_constant_identifier_names\n\n")

	var all = append([]Entity{}, sc.Types...)
	all = append(all, sc.Messages...)
	for _, e := range all {
		emitDartEntity(&b, e)
	}
	return b.String()
}

func emitDartEntity(b *strings.Builder, e Entity) {
	if e.Description != "" {
		b.WriteString("/// " + e.Description + "\n")
	}
	b.WriteString("class " + e.Name + " {\n")
	for _, f := range e.Fields {
		b.WriteString("  final " + dartType(f.Type) + " " + safeDart(f.Name) + ";\n")
	}
	b.WriteString("\n  " + e.Name + "({\n")
	for _, f := range e.Fields {
		b.WriteString("    required this." + safeDart(f.Name) + ",\n")
	}
	b.WriteString("  });\n\n")
	b.WriteString("  factory " + e.Name + ".fromJson(Map<String, dynamic> j) => " + e.Name + "(\n")
	for _, f := range e.Fields {
		b.WriteString("    " + safeDart(f.Name) + ": j['" + f.Name + "'],\n")
	}
	b.WriteString("  );\n\n")
	b.WriteString("  Map<String, dynamic> toJson() => {\n")
	for _, f := range e.Fields {
		b.WriteString("    '" + f.Name + "': " + safeDart(f.Name) + ",\n")
	}
	b.WriteString("  };\n")
	b.WriteString("}\n\n")
}

func emitArkTS(sc *Schema) string {
	var b strings.Builder
	b.WriteString("// Code generated by tools/gen from protocol/schema/weftlink.yaml. DO NOT EDIT.\n")
	b.WriteString("// Schema version: " + sc.Version + "\n\n")

	var all = append([]Entity{}, sc.Types...)
	all = append(all, sc.Messages...)
	for _, e := range all {
		if e.Description != "" {
			b.WriteString("/** " + e.Description + " */\n")
		}
		b.WriteString("export interface " + e.Name + " {\n")
		for _, f := range e.Fields {
			b.WriteString("  " + f.Name + ": " + arktsType(f.Type) + ";\n")
		}
		b.WriteString("}\n\n")
	}
	return b.String()
}

// -- Main --------------------------------------------------------------------

func main() {
	var check = false
	for _, a := range os.Args[1:] {
		if a == "-check" {
			check = true
		}
	}

	var schemaPath = "protocol/schema/weftlink.yaml"
	var sc Schema
	var err error
	err = parseSchema(schemaPath, &sc)
	if err != nil {
		fmt.Println("parse error:", err)
		os.Exit(1)
	}
	if len(sc.Types) == 0 && len(sc.Messages) == 0 {
		fmt.Println("no types/messages parsed from", schemaPath)
		os.Exit(1)
	}

	var outputs = map[string]string{
		"gen/go/types.go":     emitGo(&sc),
		"gen/dart/types.dart": emitDart(&sc),
		"gen/arkts/types.ets": emitArkTS(&sc),
	}

	var names []string
	for k := range outputs {
		names = append(names, k)
	}
	sort.Strings(names)

	var failed = false
	for _, path := range names {
		var content = outputs[path]
		if check {
			var existing []byte
			existing, err = os.ReadFile(path)
			if err != nil || string(existing) != content {
				fmt.Println("STALE:", path, "(run: go run ./tools/gen)")
				failed = true
			} else {
				fmt.Println("ok:", path)
			}
			continue
		}
		os.MkdirAll(filepath.Dir(path), 0755)
		var wErr error
		wErr = os.WriteFile(path, []byte(content), 0644)
		if wErr != nil {
			fmt.Println("write error:", wErr)
			os.Exit(1)
		}
		fmt.Println("wrote", path)
	}

	if check && failed {
		os.Exit(1)
	}
	fmt.Printf("schema v%s: %d types, %d messages\n", sc.Version, len(sc.Types), len(sc.Messages))
}
