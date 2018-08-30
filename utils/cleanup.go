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

	"github.com/pkg/errors"
)

// CleanUp temporary files
func CleanUp(dir string, err error) error {
	if dir == "" {
		return err
	}
	if err2 := os.RemoveAll(dir); err2 != nil {
		err2 = errors.Wrapf(err, "could not clean up temp files in: %s", dir)
		if err == nil {
			return err2
		}
		return Errors{err, err2}
	}
	return err
}
