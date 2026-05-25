// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

//go:build !go1.10
// +build !go1.10

package vagrant

import "archive/tar"

func setHeaderFormat(header *tar.Header) {
	// no-op
}
