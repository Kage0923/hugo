// Copyright 2018 The Hugo Authors. All rights reserved.
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

package tplimpl

import (
	"fmt"

	"github.com/gohugoio/hugo/common/herrors"
	"github.com/spf13/afero"
)

type templateInfo struct {
	name     string
	template string
	isText   bool // HTML or plain text template.

	// Used to create some error context in error situations
	fs afero.Fs

	// The filename relative to the fs above.
	filename string

	// The real filename (if possible). Used for logging.
	realFilename string
}

func (t templateInfo) Name() string {
	return t.name
}

func (t templateInfo) Filename() string {
	return t.realFilename
}

func (t templateInfo) IsZero() bool {
	return t.name == ""
}

func (t templateInfo) resolveType() templateType {
	return resolveTemplateType(t.name)
}

func (info templateInfo) errWithFileContext(what string, err error) error {
	err = fmt.Errorf(what+": %w", err)
	fe := herrors.NewFileErrorFromName(err, info.realFilename)
	f, err := info.fs.Open(info.filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return fe.UpdateContent(f, nil)

}
