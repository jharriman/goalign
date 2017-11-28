package main

import (
	"bytes"
	"flag"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

var write = flag.Bool("w", false, "Overwrite the existing files.")

func main() {
	if err := alignMain(); err != nil {
		os.Stderr.WriteString(errors.Wrap(err, "unable to align file").Error())
		os.Exit(1)
	}
	os.Exit(0)
}

func alignMain() error {
	flag.Parse()
	filename := flag.Arg(0)
	return walk(filename, *write)
}

func walk(filename string, write bool) error {
	directory, filename := path.Split(filename)
	var root string
	if directory != "" && filename == "..." {
		root = directory
	} else if directory == "" && filename == "." {
		root = "."
	} else {
		root = filename
	}
	return filepath.Walk(root, getWalker(nil))
}

func getWalker(dst io.Writer) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		// Ignore errors, directories and non-Go files.
		if err != nil || info.IsDir() || !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}
		file, err := os.OpenFile(path, os.O_RDWR, info.Mode())
		src, err := ioutil.ReadAll(file)
		if err != nil {
			return err
		}
		fileSet := token.NewFileSet()
		astFile, err := parser.ParseFile(fileSet, path, src, parser.ParseComments)
		for _, decl := range astFile.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}
				alignFieldTags(structType.Fields.List)
			}
		}
		// Print the modified file.
		if dst == nil {
			// Seek to the beginning so that we can write it again later.
			if _, err := file.Seek(0, 0); err != nil {
				return err
			}
			dst = file
		}
		return format.Node(dst, fileSet, astFile)
	}
}

func alignFieldTags(fields []*ast.Field) error {
	encounteredTags := make([]string, 0)
	maxTagLengths := make(map[string]int)
	fieldTagNamesToValues := make([]map[string]string, len(fields))
	fieldNumTags := make([]int, len(fields))

	// Iterate once through to find all the max values.
	for i, field := range fields {
		tags := strings.Fields(strings.Trim(field.Tag.Value, "`"))
		namesToValues := make(map[string]string)
		for _, tag := range tags {
			tagName := strings.SplitN(tag, ":", 2)[0]
			namesToValues[tagName] = tag

			if _, ok := maxTagLengths[tagName]; !ok {
				encounteredTags = append(encounteredTags, tagName)
			}
			prevMax, ok := maxTagLengths[tagName]
			if !ok {
				maxTagLengths[tagName] = len(tag)
			} else if prevMax < len(tag) {
				maxTagLengths[tagName] = len(tag)
			}
		}
		fieldTagNamesToValues[i] = namesToValues
		fieldNumTags[i] = len(tags)
	}

	// Pad all the strings according to the max values.
	for i, field := range fields {
		// Keep track of the visited number of tags in order to bail early.
		visitedTags := 0
		tagBuffer := bytes.NewBufferString("`")
		for _, key := range encounteredTags {
			val, ok := fieldTagNamesToValues[i][key]
			if ok {
				visitedTags += 1
			}
			numSpaces := maxTagLengths[key] - len(val)
			if _, err := tagBuffer.WriteString(val); err != nil {
				return err
			}
			// Bail early since we don't need to print anymore whitespace.
			if visitedTags >= fieldNumTags[i] {
				break
			}
			// Pad the tag.
			for i := 0; i < numSpaces; i++ {
				if _, err := tagBuffer.WriteString(" "); err != nil {
					return err
				}
			}
			// Add space between each tag.
			if _, err := tagBuffer.WriteString(" "); err != nil {
				return err
			}
		}
		tagBuffer.WriteString("`")
		field.Tag.Value = tagBuffer.String()
	}
	return nil
}
