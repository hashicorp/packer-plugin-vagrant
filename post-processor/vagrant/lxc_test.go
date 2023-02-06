// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package vagrant

import (
	"testing"
)

func TestLXCProvider_impl(t *testing.T) {
	var _ Provider = new(LXCProvider)
}
