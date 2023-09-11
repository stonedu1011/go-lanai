package generator

import (
	"bytes"
	"cto-github.cisco.com/NFV-BU/go-lanai/cmd/lanai-cli/cmdutils"
	"embed"
	"fmt"
	"github.com/bmatcuk/doublestar/v4"
	"go/format"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

const defaultSrcRootDir = "template/src"

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}
func GenerateFileFromTemplate(gc GenerationContext, template *template.Template) error {
	if gc.templatePath == "" {
		return fmt.Errorf("no templatePath name defined")
	}
	if !path.IsAbs(gc.filename) {
		return fmt.Errorf("templatePath output path should be absolute, but got [%s]", gc.filename)
	}

	outputFolder := path.Dir(gc.filename)
	if err := mkdirIfNotExists(outputFolder); err != nil {
		return fmt.Errorf("unable to create directory of templatePath output [%s]", outputFolder)
	}
	wc, err := applyRegenRule(&gc)
	if err != nil {
		return err
	}
	defer func() { _ = wc.Close() }()

	if err := template.ExecuteTemplate(wc, gc.templatePath, gc.model); err != nil {
		return fmt.Errorf("templatePath could not be executed: %v", err)
	}

	if isGoExt(gc.filename) {
		err := formatGoCode(gc.filename)
		if err != nil {
			return fmt.Errorf("error formatting go code for file %v: %v", gc.filename, err)
		}
	}
	return nil
}

func mkdirIfNotExists(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("%s is not absolute path", path)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if e := os.MkdirAll(path, 0744); e != nil {
			return e
		}
	}
	return nil
}

type emptyWriteCloser struct{}

func (e *emptyWriteCloser) Write(p []byte) (n int, err error) {
	return 0, nil
}

func (e *emptyWriteCloser) Close() (err error) {
	return nil
}

func applyRegenRule(gc *GenerationContext) (f io.WriteCloser, err error) {
	if fileExists(gc.filename) {
		switch gc.regenMode {
		case RegenModeIgnore:
			logger.Infof("ignore rule defined for existing file %v, ignoring", gc.filename)
			//	make an empty applyRegenRule to allow the template to be executed (and keep any runtime logic consistent)
			return &emptyWriteCloser{}, nil
		case RegenModeReference:
			gc.filename += "ref"
			fallthrough
		case RegenModeOverwrite:
			break
		}
	}
	f, err = os.OpenFile(gc.filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func isGoExt(filename string) bool {
	return path.Ext(filename) == ".go" || path.Ext(filename) == ".goref"
}

func formatGoCode(outputFilePath string) error {
	r, err := os.ReadFile(outputFilePath)
	if err != nil {
		return err
	}
	formatted, err := format.Source(r)
	if err != nil {
		return err
	}
	if err := os.WriteFile(outputFilePath, formatted, 0644); err != nil {
		return err
	}
	return nil
}

// ConvertSrcRootToTargetDir will take a path containing a SrcRoot directory, and return
// an equivalent path to the target directory, with any special folders resolved with modifiers
// e.g template/srcRoot/cmd/@NAME@/main.go -> output/dir/cmd/myservice/main.go
func ConvertSrcRootToTargetDir(srcPath string, modifiers map[string]interface{}, filesystem fs.FS) (string, error) {
	relativeDir := relativePathFromSrcRoot(srcPath, filesystem)
	unresolvedTargetDir := combineWithOutputDir(relativeDir)

	if modifiers == nil {
		return unresolvedTargetDir, nil
	}

	return resolvePath(modifiers, unresolvedTargetDir)
}

func relativePathFromSrcRoot(path string, templates fs.FS) string {
	_, isEmbedFS := templates.(embed.FS)
	if isEmbedFS {
		parts := strings.SplitAfterN(path, defaultSrcRootDir+"/", 2)
		if len(parts) != 2 {
			return ""
		} else {
			return parts[1]
		}
	} else {
		//	For os.FS, the path is already considered relative
		return path
	}
}

func combineWithOutputDir(relativeDir string) string {
	return path.Join(cmdutils.GlobalArgs.OutputDir, relativeDir)
}

func resolvePath(modifiers map[string]interface{}, unresolvedTargetDir string) (string, error) {
	matches := regexp.MustCompile("@(.+?)@").FindAllStringSubmatch(unresolvedTargetDir, -1)
	if len(matches) == 0 {
		return unresolvedTargetDir, nil
	}

	for _, match := range matches {
		if len(match) != 2 {
			continue
		}
		// replace @s to template compatible format
		unresolvedTargetDir = strings.Replace(unresolvedTargetDir, match[0], fmt.Sprintf("{{ with index . \"%v\"}}{{.}}{{ end }}", match[1]), 1)
	}

	tmpl := template.Must(template.New("filepath").Parse(unresolvedTargetDir))
	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, modifiers); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func copyOf(data map[string]interface{}) map[string]interface{} {
	dataCopy := make(map[string]interface{})
	for k, v := range data {
		dataCopy[k] = v
	}
	return dataCopy
}

func getApplicableRegenRules(outputFile string, rules RegenRules, defaultMode RegenMode) (RegenMode, error) {
	pathAfterOutputDir := strings.TrimPrefix(outputFile, cmdutils.GlobalArgs.OutputDir+"/")
	mode := defaultMode
	for _, r := range rules {
		match, err := doublestar.Match(r.Pattern, pathAfterOutputDir)
		if err != nil {
			return "", err
		}
		if match {
			mode = r.Mode
		}
	}
	return mode, nil
}
