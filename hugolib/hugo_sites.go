// Copyright 2016-present The Hugo Authors. All rights reserved.
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

package hugolib

// HugoSites represents the sites to build. Each site represents a language.
type HugoSites []*Site

// Reset resets the sites, making it ready for a full rebuild.
// TODO(bep) multilingo
func (h HugoSites) Reset() {
	for i, s := range h {
		h[i] = s.Reset()
	}
}
