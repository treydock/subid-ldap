// Copyright 2021 Trey Dockendorf
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"reflect"
	"testing"
)

func TestSliceContains(t *testing.T) {
	value := SliceContains([]string{"foo", "bar"}, "bar")
	if value != true {
		t.Errorf("Expected slice to contain value")
	}
	value = SliceContains([]string{"foo"}, "bar")
	if value != false {
		t.Errorf("Expected slice not to contain value")
	}
}

func TestSortSliceStringInts(t *testing.T) {
	input := []string{"3", "1", "2"}
	SortSliceStringInts(&input)
	if !reflect.DeepEqual(input, []string{"1", "2", "3"}) {
		t.Errorf("Unexpected result, got: %+v", input)
	}
}
