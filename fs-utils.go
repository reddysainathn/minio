/*
 * Minio Cloud Storage, (C) 2016 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"regexp"
	"runtime"
	"strings"
	"unicode/utf8"
)

// validVolname regexp.
var validVolname = regexp.MustCompile(`^.{3,63}$`)

// isValidVolname verifies a volname name in accordance with object
// layer requirements.
func isValidVolname(volname string) bool {
	if !validVolname.MatchString(volname) {
		return false
	}
	switch runtime.GOOS {
	case "windows":
		// Volname shouldn't have reserved characters on windows in it.
		return !strings.ContainsAny(volname, "/\\:*?\"<>|")
	default:
		// Volname shouldn't have '/' in it.
		return !strings.ContainsAny(volname, "/")
	}
}

// Keeping this as lower bound value supporting Linux, Darwin and Windows operating systems.
const pathMax = 4096

// isValidPath verifies if a path name is in accordance with FS limitations.
func isValidPath(path string) bool {
	// TODO: Make this FSType or Operating system specific.
	if len(path) > pathMax || len(path) == 0 {
		return false
	}
	if !utf8.ValidString(path) {
		return false
	}
	return true
}

// isValidPrefix verifies where the prefix is a valid path.
func isValidPrefix(prefix string) bool {
	// Prefix can be empty.
	if prefix == "" {
		return true
	}
	// Verify if prefix is a valid path.
	return isValidPath(prefix)
}

// List of reserved words for files, includes old and new ones.
var reservedKeywords = []string{
	"$multiparts",
	"$tmpobject",
	"$tmpfile",
	// Add new reserved words if any used in future.
}

// hasReservedPrefix - returns true if name has a reserved keyword suffixed.
func hasReservedSuffix(name string) (isReserved bool) {
	for _, reservedKey := range reservedKeywords {
		if strings.HasSuffix(name, reservedKey) {
			isReserved = true
			break
		}
		isReserved = false
	}
	return isReserved
}

// hasReservedPrefix - has reserved prefix.
func hasReservedPrefix(name string) (isReserved bool) {
	for _, reservedKey := range reservedKeywords {
		if strings.HasPrefix(name, reservedKey) {
			isReserved = true
			break
		}
		isReserved = false
	}
	return isReserved
}
