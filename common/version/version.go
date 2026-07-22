package version

import (
	"fmt"
)

type VersionInfo struct {
	Name        string
	Version     string
	Branch      string
	CommitHash  string
	Compiler    string
	CompileTime string
}

func (v *VersionInfo) String() string {
	return fmt.Sprintf(
		"Module Name:   %s\nVersion:       %s\nBranch:        %s\nCommit Hash:   %s\nCompiler:      %s\nCompile Time:  %s",
		v.Name,
		v.Version,
		v.Branch,
		v.CommitHash,
		v.Compiler,
		v.CompileTime,
	)
}

func NewVersionInfo(name, version, branch, commitHash, compiler, compileTime string) *VersionInfo {
	return &VersionInfo{
		Name:        name,
		Version:     version,
		Branch:      branch,
		CommitHash:  commitHash,
		Compiler:    compiler,
		CompileTime: compileTime,
	}
}
