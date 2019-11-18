package pkgs

import (
	"go/ast"
	"strings"
)

type constMap map[string]Const

func newEnums() constMap {
	return make(map[string]Const)
}

type Const struct {
	Type string `json:"type"`
	Value string `json:"value"`
}

func (c *constMap) addEnum(pkg Package, g *ast.GenDecl) {
	for _, s := range g.Specs {
		co := Const{}
		vs := s.(*ast.ValueSpec)
		v := ""
		// Type is nil for untyped consts
		if vs.Type != nil {
			co.Type = vs.Type.(*ast.Ident).Name
			v = vs.Values[0].(*ast.BasicLit).Value
		} else {
			// get the type from the token type
			if bl, ok := vs.Values[0].(*ast.BasicLit); ok {
				co.Type = strings.ToLower(bl.Kind.String())
				v = bl.Value
			} else if ce, ok := vs.Values[0].(*ast.CallExpr); ok {
				// const FooConst = FooType("value")
				co.Type = pkg.getText(ce.Fun.Pos(), ce.Fun.End())
				v = pkg.getText(ce.Args[0].Pos(), ce.Args[0].End())
			} else {
				panic("unhandled cases for adding contanst")
			}
		}
		// remove any surrounding quotes
		if v[0] == '"' {
			v = v[1 : len(v) - 1]
		}
		co.Value = v
		(*c)[vs.Names[0].Name] = co
	}
}

type EnumEntry struct {
	Name string `json:"name"`
	Value string `json:"value"`
}

func (c *constMap) ToMap() map[string][]EnumEntry {
	result := make(map[string][]EnumEntry)
	for vName, constant := range *c {
		if _, ok := result[constant.Type]; !ok {
			result[constant.Type] = make([]EnumEntry, 0)
		}
		array := result[constant.Type]
		result[constant.Type] = append(array, EnumEntry{
			Name:  vName,
			Value: constant.Value,
		})
	}
	return result
}
