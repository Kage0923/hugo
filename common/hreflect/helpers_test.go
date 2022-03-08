// Copyright 2019 The Hugo Authors. All rights reserved.
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

package hreflect

import (
	"reflect"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
)

func TestIsTruthful(t *testing.T) {
	c := qt.New(t)

	c.Assert(IsTruthful(true), qt.Equals, true)
	c.Assert(IsTruthful(false), qt.Equals, false)
	c.Assert(IsTruthful(time.Now()), qt.Equals, true)
	c.Assert(IsTruthful(time.Time{}), qt.Equals, false)
}

func TestGetMethodByName(t *testing.T) {
	c := qt.New(t)
	v := reflect.ValueOf(&testStruct{})
	tp := v.Type()

	c.Assert(GetMethodIndexByName(tp, "Method1"), qt.Equals, 0)
	c.Assert(GetMethodIndexByName(tp, "Method3"), qt.Equals, 2)
	c.Assert(GetMethodIndexByName(tp, "Foo"), qt.Equals, -1)
}

func BenchmarkIsTruthFul(b *testing.B) {
	v := reflect.ValueOf("Hugo")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !IsTruthfulValue(v) {
			b.Fatal("not truthful")
		}
	}
}

type testStruct struct{}

func (t *testStruct) Method1() string {
	return "Hugo"
}

func (t *testStruct) Method2() string {
	return "Hugo"
}

func (t *testStruct) Method3() string {
	return "Hugo"
}

func (t *testStruct) Method4() string {
	return "Hugo"
}

func (t *testStruct) Method5() string {
	return "Hugo"
}

func BenchmarkGetMethodByName(b *testing.B) {
	v := reflect.ValueOf(&testStruct{})
	methods := []string{"Method1", "Method2", "Method3", "Method4", "Method5"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, method := range methods {
			_ = GetMethodByName(v, method)
		}
	}
}
