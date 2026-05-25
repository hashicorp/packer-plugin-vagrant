// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package vagrant

import (
	"testing"
)

func TestVMwareProvider_impl(t *testing.T) {
	var _ Provider = new(VMwareProvider)
}
