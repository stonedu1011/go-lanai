package generator

import (
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"text/template"
)

const (
	projectMatcherRegexTemplate = "^(?:%s)(.+)(?:.tmpl)"
	projectDefaultPrefix        = "project."
)

// FileGenerator is a basic generator that generates 1 file based on the templatePath being used
type FileGenerator struct {
	data             map[string]interface{}
	template         *template.Template
	nameRegex        *regexp.Regexp
	templateFS       fs.FS
	outputFS         fs.FS
	priorityOrder    int
	defaultRegenRule RegenMode
	rules            RegenRules
}

type FileGenOption struct {
	Option
	Prefix        string
	PriorityOrder int
}

// newFileGenerator returns a new generator for single files
func newFileGenerator(opts ...func(opt *FileGenOption)) *FileGenerator {
	o := &FileGenOption{
		Prefix: projectDefaultPrefix,
		PriorityOrder: defaultProjectPriorityOrder,
	}
	for _, fn := range opts {
		fn(o)
	}

	regex := fmt.Sprintf(projectMatcherRegexTemplate, regexp.QuoteMeta(o.Prefix))

	logger.Infof("DefaultRegenMode: %v", o.DefaultRegenMode)
	return &FileGenerator{
		data:             o.Data,
		template:         o.Template,
		nameRegex:        regexp.MustCompile(regex),
		templateFS:       o.TemplateFS,
		outputFS:         o.OutputFS,
		priorityOrder:    o.PriorityOrder,
		defaultRegenRule: o.DefaultRegenMode,
		rules:            o.RegenRules,
	}
}

func (o *FileGenerator) determineFilename(template string) string {
	var result string
	matches := o.nameRegex.FindStringSubmatch(path.Base(template))
	if len(matches) < 2 {
		result = ""
	}

	result = matches[1]
	return result
}

func (o *FileGenerator) Generate(tmplPath string, dirEntry fs.DirEntry) error {
	if dirEntry.IsDir() || !o.nameRegex.MatchString(path.Base(tmplPath)) {
		// Skip over it
		return nil
	}

	targetDir, err := ConvertSrcRootToTargetDir(path.Dir(tmplPath), o.data, o.templateFS)
	if err != nil {
		return err
	}
	baseFilename := o.determineFilename(tmplPath)

	outputFile := path.Join(targetDir, baseFilename)

	regenRule, err := getApplicableRegenRules(outputFile, o.rules, o.defaultRegenRule)
	if err != nil {
		return err
	}
	file := *NewGenerationContext(
		tmplPath,
		outputFile,
		regenRule,
		o.data,
	)
	logger.Infof("project generator generating %v\n", outputFile)
	return GenerateFileFromTemplate(file, o.template)
}

func (o *FileGenerator) PriorityOrder() int {
	return o.priorityOrder
}