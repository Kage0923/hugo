// Copyright © 2013 Steve Francia <spf@spf13.com>.
//
// Licensed under the Simple Public License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://opensource.org/licenses/Simple-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hugolib

import (
	"html/template"
	"time"
)

type Node struct {
	Url         string
	Permalink   template.HTML
	RSSlink     template.HTML
	Site        SiteInfo
	layout      string
	Data        map[string]interface{}
	Section     string
	Slug        string
	Title       string
	Description string
	Keywords    []string
	Date        time.Time
}

func (n *Node) GetSection() string {
	s := ""
	if n.Section != "" {
		s = n.Section
	}

	return s
}
