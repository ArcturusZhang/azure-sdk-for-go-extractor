package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ArcturusZhang/azure-sdk-for-go-extractor/pkgs"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "extractor <go sdk service directory> <result path>",
	Short: "",
	Long:  "",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return theCommand(args)
	},
}

// Execute runs the command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func theCommand(args []string) error {
	servicePath, err := filepath.Abs(args[0])
	if err != nil {
		return fmt.Errorf("failed to get absolute path of services path: %+v", err)
	}
	resultPath, err := filepath.Abs(args[1])
	if err != nil {
		return fmt.Errorf("failed to get absolute path of result path: %+v", err)
	}
	packages, err := pkgs.GetPackages(servicePath)
	if err != nil {
		return fmt.Errorf("failed to get packages in '%s': %+v", servicePath, err)
	}
	m := make(map[string][]ServiceInfo)
	for _, pkg := range packages {
		fmt.Printf("Analysing package: %s\n", pkg.Name())
		path := filepath.Join(servicePath, pkg.Dest)
		if _, ok := m[pkg.Name()]; !ok {
			m[pkg.Name()] = make([]ServiceInfo, 0)
		}
		services := m[pkg.Name()]
		e, err := ParseGoPackage(path)
		if err != nil {
			return fmt.Errorf("failed to parse package '%s': %+v", path, err)
		}
		info := ServiceInfo{
			Dest:  pkg.Dest,
			Enums: e,
		}
		m[pkg.Name()] = append(services, info)
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal result to json: %+v", err)
	}
	return ioutil.WriteFile(resultPath, b, 0755)
}

// ParseGoPackage parses the package and convert its exports into a Enums struct
func ParseGoPackage(dir string) (map[string][]pkgs.EnumEntry, error) {
	p, err := pkgs.LoadPackage(dir)
	if err != nil {
		return nil, err
	}
	c := p.GetEnumerations()
	// remove those keys are "string"
	delete(c, "string")
	return c, nil
}

type ServiceInfo struct {
	Dest string `json:"dest"`
	Enums map[string][]pkgs.EnumEntry `json:"enums"`
}
