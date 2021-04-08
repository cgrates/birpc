// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpcc

import "log"

// DebugLog controls the printing of internal and I/O errors.
var DebugLog = false

func debugln(v ...interface{}) {
	if DebugLog {
		log.Println(v...)
	}
}
