// Copyright © 2015 Steve Francia <spf@spf13.com>.
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

package helpers

import (
	"fmt"
)

// this should be the only one
const hugoVersionMain = 0.14
const hugoVersionSuffix = "-DEV" // blank this when doing a release

// HugoVersion returns the current Hugo version. It will include
// a suffix, typically '-DEV', if it's development version.
func HugoVersion() string {
	return hugoVersion(hugoVersionMain, hugoVersionSuffix)
}

// HugoReleaseVersion is same as HugoVersion, but no suffix.
func HugoReleaseVersion() string {
	return hugoVersionNoSuffix(hugoVersionMain)
}

// NextHugoReleaseVersion returns the next Hugo release version.
func NextHugoReleaseVersion() string {
	return hugoVersionNoSuffix(hugoVersionMain + 0.01)
}

func hugoVersion(version float32, suffix string) string {
	return fmt.Sprintf("%.2g%s", version, suffix)
}

func hugoVersionNoSuffix(version float32) string {
	return fmt.Sprintf("%.2g", version)
}
