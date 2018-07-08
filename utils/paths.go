// Copyright © 2018 SENETAS SECURITY PTY LTD
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"os"
	"path"
	"path/filepath"
)

// PathTrailingJoin is like path.Join but ensures there is a trailing seprator
func PathTrailingJoin(s ...string) string {
	return path.Join(s...) + "/"
}

// FilePathTrailingJoin is like filepath.Join but ensures there is a trailing seprator
func FilePathTrailingJoin(s ...string) string {
	return filepath.Join(s...) + string(os.PathSeparator)
}
