package pkgs

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type servicePackage struct {
	// the directory where the package resides relative to the root dir
	Dest string

	// the AST of the package
	p *ast.Package
}

// returns true if the package directory corresponds to an ARM package
func (p servicePackage) IsARMPackage() bool {
	return strings.Index(p.Dest, "/mgmt/") > -1
}

// returns true if the package directory corresponds to a preview package
func (p servicePackage) IsPreviewPackage() bool {
	return strings.Index(p.Dest, "preview") > -1
}

func (p servicePackage) Name() string {
	return p.p.Name
}

// GetPackages walks the directory hierarchy from the specified root returning a slice of all the packages found
func GetPackages(rootDir string) ([]servicePackage, error) {
	pkgs := make([]servicePackage, 0)
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// check if leaf dir
			fi, err := ioutil.ReadDir(path)
			if err != nil {
				return err
			}
			hasSubDirs := false
			interfacesDir := false
			for _, f := range fi {
				if f.IsDir() {
					hasSubDirs = true
					break
				}
				if f.Name() == "interfaces.go" {
					interfacesDir = true
				}
			}
			if !hasSubDirs {
				fs := token.NewFileSet()
				// with interfaces codegen the majority of leaf directories are now the
				// *api packages. when this is the case parse from the parent directory.
				if interfacesDir {
					path = filepath.Dir(path)
				}
				packages, err := parser.ParseDir(fs, path, func(fi os.FileInfo) bool {
					return fi.Name() != "interfaces.go"
				}, parser.PackageClauseOnly)
				if err != nil {
					return err
				}
				if len(packages) < 1 {
					return errors.New("didn't find any packages which is unexpected")
				}
				if len(packages) > 1 {
					return errors.New("found more than one package which is unexpected")
				}
				var p *ast.Package
				for _, pkgs := range packages {
					p = pkgs
				}
				// normalize directory separator to '/' character
				pkgs = append(pkgs, servicePackage{
					Dest: strings.Replace(path[len(rootDir):], "\\", "/", -1),
					p:    p,
				})
			}
		}
		return nil
	})
	return pkgs, err
}

type Package struct {
	f     *token.FileSet
	p     *ast.Package
	files map[string][]byte
}

func (pkg Package) getText(start token.Pos, end token.Pos) string {
	// convert to absolute position within the containing file
	p := pkg.f.Position(start)
	// check if the file has been loaded, if not then load it
	if _, ok := pkg.files[p.Filename]; !ok {
		b, err := ioutil.ReadFile(p.Filename)
		if err != nil {
			panic(err)
		}
		pkg.files[p.Filename] = b
	}
	content := pkg.files[p.Filename]
	bytes := content[p.Offset : p.Offset + int(end - start)]
	return string(bytes)
}

func (pkg Package) GetEnumerations() map[string][]EnumEntry {
	c := newEnums()
	ast.Inspect(pkg.p, func(node ast.Node) bool {
		switch x := node.(type) {
		case *ast.GenDecl:
			// indicate this type definition is an type alias
			if x.Tok == token.CONST {
				c.addEnum(pkg, x)
			}
		}
		return true
	})
	return c.ToMap()
}

func LoadPackage(dir string) (*Package, error) {
	pkg := Package{
		f:     token.NewFileSet(),
		files: map[string][]byte{},
	}
	packages, err := parser.ParseDir(pkg.f, dir, nil, 0)
	if err != nil {
		return nil, err
	}
	if len(packages) < 1 {
		return nil, fmt.Errorf("did not find any packages in '%s'", dir)
	}
	if len(packages) > 1 {
		return nil, fmt.Errorf("found more than one package in '%s'", dir)
	}
	for name, p := range packages {
		// trim non-exports
		if exp := ast.PackageExports(p); !exp {
			return nil, fmt.Errorf("package '%s' does not contain any exports", name)
		}
		pkg.p = p
		return &pkg, nil
	}
	// we should never reach here
	return nil, nil
}
