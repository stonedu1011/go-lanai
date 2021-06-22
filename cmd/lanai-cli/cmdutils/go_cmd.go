package cmdutils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
)

var (
	targetTmpGoModFile string
	targetModule       *GoModule
	targetModuleOnce   = sync.Once{}

	packageImportPathCache     map[string]*GoPackage
	packageImportPathCacheOnce = sync.Once{}
)

type GoCmdOptions func(goCmd *string)

func GoCmdModFile(modFile string) GoCmdOptions {
	return func(goCmd *string) {
		if modFile == "" {
			return
		}
		*goCmd = fmt.Sprintf("%s -modfile %s", *goCmd, modFile)
	}
}

func ResolveTargetModule(ctx context.Context) *GoModule {
	targetModuleOnce.Do(func() {
		// first, prepare a mod file to read
		modFile, e := prepareTargetGoModFile(ctx)
		if e != nil {
			logger.WithContext(ctx).Errorf("unable to prepare temporary go.mod file to resolve target module: %v", e)
		}
		targetTmpGoModFile = modFile

		// find module
		mods, e := FindModule(ctx, []GoCmdOptions{GoCmdModFile(modFile)})
		if e == nil && len(mods) == 1 {
			targetModule = mods[0]
		} else if e != nil {
			logger.WithContext(ctx).Errorf("unable to resolve target module name: %v", e)
		} else {
			logger.WithContext(ctx).Errorf("resolved multiple modules in working directory")
		}
	})
	return targetModule
}

func PackageImportPathCache(ctx context.Context) map[string]*GoPackage {
	packageImportPathCacheOnce.Do(func() {
		module := ResolveTargetModule(ctx)
		if module == nil {
			return
		}
		var err error
		packageImportPathCache, err = FindPackages(ctx, []GoCmdOptions{GoCmdModFile(targetTmpGoModFile)}, module.Path)
		if err != nil {
			logger.WithContext(ctx).Errorf("unable to resolve local packages in module %s", module.Path)
		}
	})
	return packageImportPathCache
}

func FindModule(ctx context.Context, opts []GoCmdOptions, modules ...string) ([]*GoModule, error) {
	cmd := "go list -m -json"
	for _, f := range opts {
		f(&cmd)
	}
	cmd = fmt.Sprintf("%s %s", cmd, strings.Join(modules, " "))

	result, e := GoCommandDecodeJson(ctx, &GoModule{},
		ShellShowCmd(true),
		ShellUseWorkingDir(),
		ShellCmd(cmd),
	)
	if e != nil {
		return nil, e
	}

	var ret []*GoModule
	for _, v := range result {
		m := v.(*GoModule)
		ret = append(ret, m)
	}
	return ret, nil
}

func FindPackages(ctx context.Context, opts []GoCmdOptions, modules ...string) (map[string]*GoPackage, error) {
	cmd := "go list -json"
	for _, f := range opts {
		f(&cmd)
	}
	cmd = fmt.Sprintf("%s %s/...", cmd, strings.Join(modules, " "))

	result, e := GoCommandDecodeJson(ctx, &GoPackage{},
		ShellShowCmd(true),
		ShellUseWorkingDir(),
		ShellCmd(cmd),
	)
	if e != nil {
		return nil, e
	}

	pkgs := map[string]*GoPackage{}
	for _, v := range result {
		pkg := v.(*GoPackage)
		pkgs[pkg.ImportPath] = pkg
	}
	return pkgs, nil
}

// DropInvalidReplace go through the go.mod file and find replace directives that point to a non-existing local directory
func DropInvalidReplace(ctx context.Context, opts ...GoCmdOptions) (ret []*Replace, err error) {
	mod, e := GetGoMod(ctx, opts...)
	if e != nil {
		return nil, e
	}

	cmdOpts := []ShCmdOptions{
		ShellShowCmd(true),
		ShellUseWorkingDir(),
		ShellStdOut(os.Stdout),
	}
	for _, v := range mod.Replace {
		if isInvalidReplace(&v) {
			ret = append(ret, &v)
			cmdOpts = append(cmdOpts, dropReplaceCmd(v.Old.Path, v.Old.Version, opts))
		}
	}
	if len(ret) == 0 {
		return
	}

	if _, e := RunShellCommands(ctx, cmdOpts...); e != nil {
		return nil, e
	}
	return
}

// RestoreInvalidReplace works together with DropInvalidReplace
func RestoreInvalidReplace(ctx context.Context, replaces []*Replace, opts ...GoCmdOptions) error {
	if len(replaces) == 0 {
		return nil
	}

	cmdOpts := []ShCmdOptions{
		ShellShowCmd(true),
		ShellUseWorkingDir(),
		ShellStdOut(os.Stdout),
	}
	for _, v := range replaces {
		cmdOpts = append(cmdOpts, setReplaceCmd(v, opts))
	}

	_, err := RunShellCommands(ctx, cmdOpts...)

	return err
}

func DropReplace(ctx context.Context, module string, version string, opts ...GoCmdOptions) error {
	logger.Infof("dropping replace directive %s, %s", module, version)
	_, err := RunShellCommands(ctx,
		ShellShowCmd(true),
		ShellUseWorkingDir(),
		dropReplaceCmd(module, version, opts),
		ShellStdOut(os.Stdout))

	return err
}

func DropRequire(ctx context.Context, module string, opts ...GoCmdOptions) error {
	cmd := "go mod edit"
	for _, f := range opts {
		f(&cmd)
	}
	cmd = fmt.Sprintf("%s -droprequire %s", cmd, module)

	logger.Infof("dropping require directive %s", module)
	_, err := RunShellCommands(ctx,
		ShellShowCmd(true),
		ShellUseWorkingDir(),
		ShellCmd(cmd),
		ShellStdOut(os.Stdout))

	return err
}

func GoGet(ctx context.Context, module string, versionQuery string, opts ...GoCmdOptions) error {
	cmd := "go get"
	for _, f := range opts {
		f(&cmd)
	}
	cmd = fmt.Sprintf("%s %s@%s", cmd, module, versionQuery)

	_, e := RunShellCommands(ctx,
		ShellShowCmd(true),
		ShellUseWorkingDir(),
		ShellStdOut(os.Stdout),
		ShellCmd(cmd),
	)
	return e
}

func GoModTidy(ctx context.Context, opts ...GoCmdOptions) error {
	cmd := "go mod tidy"
	for _, f := range opts {
		f(&cmd)
	}

	_, e := RunShellCommands(ctx,
		ShellShowCmd(true),
		ShellUseWorkingDir(),
		ShellCmd(cmd),
		ShellStdOut(os.Stdout))
	return e
}

func GetGoMod(ctx context.Context, opts ...GoCmdOptions) (*GoMod, error){
	cmd := fmt.Sprintf("go mod edit -json")
	for _, f := range opts {
		f(&cmd)
	}
	result, e := GoCommandDecodeJson(ctx, &GoMod{},
		ShellShowCmd(true),
		ShellUseWorkingDir(),
		ShellCmd(cmd),
	)
	if e != nil {
		return nil, e
	}

	m := result[0].(*GoMod)
	return m, nil
}

/***********************
	Exported Helpers
 ***********************/

func IsLocalPackageExists(ctx context.Context, pkgPath string) (bool, error) {
	cache := PackageImportPathCache(ctx)
	if cache == nil {
		return false, fmt.Errorf("package import path cache is not available")
	}
	_, ok := cache[pkgPath]
	return ok, nil
}

func GoCommandDecodeJson(ctx context.Context, model interface{}, opts ...ShCmdOptions) (ret []interface{}, err error) {
	mt := reflect.TypeOf(model)
	if mt.Kind() == reflect.Ptr {
		mt = mt.Elem()
	}

	pr, pw := io.Pipe()
	opts = append(opts, ShellStdOut(pw))
	ech := make(chan error, 1)
	go func() {
		defer pw.Close()
		defer close(ech)
		_, e := RunShellCommands(ctx, opts...)
		if e != nil {
			ech <- e
		}
	}()

	dec := json.NewDecoder(pr)
	for {
		m := reflect.New(mt).Interface()
		if e := dec.Decode(&m); e != nil {
			if e != io.EOF {
				err = e
			}
			break
		}
		ret = append(ret, m)
	}

	if e := <-ech; e != nil {
		err = e
		return
	}
	return
}

/***********************
	Helper Functions
 ***********************/

func withVersionQuery(module string, version string) string {
	if version == "" {
		return module
	}

	return fmt.Sprintf("%s@%s", module, version)
}

func dropReplaceCmd(module string, version string, opts []GoCmdOptions) ShCmdOptions {
	cmd := "go mod edit"
	for _, f := range opts {
		f(&cmd)
	}
	cmd = fmt.Sprintf("%s -dropreplace %s", cmd, withVersionQuery(module, version))

	return ShellCmd(cmd)
}

func setReplaceCmd(replace *Replace, opts []GoCmdOptions) ShCmdOptions {
	cmd := "go mod edit"
	for _, f := range opts {
		f(&cmd)
	}
	from := withVersionQuery(replace.Old.Path, replace.Old.Version)
	to := withVersionQuery(replace.New.Path, replace.New.Version)
	cmd = fmt.Sprintf("%s -replace %s=%s", cmd, from, to)

	return ShellCmd(cmd)
}

func tmpGoModFile() string {
	return GlobalArgs.AbsPath(GlobalArgs.TmpDir, "go.tmp.mod")
}

func isInvalidReplace(replace *Replace) bool {
	replaced := replace.New.Path
	// we only care if the replaced path start with "/" or ".",
	// i.e. we will ignore url path such as "cto-github.cisco.com/NFV-BU/go-lanai"
	if replaced == "" || !filepath.IsAbs(replaced) && !strings.HasPrefix(replaced, ".") {
		return false
	}

	if !filepath.IsAbs(replaced) {
		replaced = filepath.Clean(GlobalArgs.WorkingDir + "/" + replaced)
	}
	return !isFileExists(replaced)
}

func prepareTargetGoModFile(ctx context.Context) (string, error) {
	tmpModFile := tmpGoModFile()
	// make a copy of go.mod and go.sum in tmp folder
	files := map[string]string{
		"go.mod": tmpModFile,
		"go.sum": GlobalArgs.AbsPath(GlobalArgs.TmpDir, "go.tmp.sum"),
	}
	if e := copyFiles(ctx, files); e != nil {
		return "", fmt.Errorf("error when copying go.mod: %v", e)
	}

	// drop invalid replace
	replaces, e := DropInvalidReplace(ctx, GoCmdModFile(tmpModFile))
	if e != nil {
		return "", fmt.Errorf("error when drop invalid replaces: %v", e)
	}

	// drop require as well (we don't need to to resolve local packages
	for _, v := range replaces {
		if e := DropRequire(ctx, v.Old.Path, GoCmdModFile(tmpModFile)); e != nil {
			return "", fmt.Errorf("error when dropping require %s: %v", v.Old.Path, e)
		}
	}
	return tmpModFile, nil
}