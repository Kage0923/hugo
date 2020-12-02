// Copyright 2019 The Hugo Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:generate go run generate.go

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	templateFolder := filepath.Join("..", "templates")

	temlatePath := filepath.Join(".", templateFolder)

	file, err := os.Create("../templates.autogen.go")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var nameValues []string

	err = filepath.Walk(temlatePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		templateName := filepath.ToSlash(strings.TrimPrefix(path, templateFolder+string(os.PathSeparator)))

		templateContent, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		nameValues = append(nameValues, nameValue(templateName, string(templateContent)))

		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Fprint(file, `// Copyright 2019 The Hugo Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This file is autogenerated.

// Package embedded defines the internal templates that Hugo provides.
package embedded

// EmbeddedTemplates represents all embedded templates.
var EmbeddedTemplates = [][2]string{
`)

	for _, v := range nameValues {
		fmt.Fprint(file, "	", v, ",\n")
	}
	fmt.Fprint(file, "}\n")
}

func nameValue(name, value string) string {
	return fmt.Sprintf("{`%s`, `%s`}", name, value)
}
