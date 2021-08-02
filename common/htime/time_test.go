// Copyright 2021 The Hugo Authors. All rights reserved.
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

package htime

import (
	"testing"
	"time"

	translators "github.com/gohugoio/localescompressed"
	qt "github.com/frankban/quicktest"
)

func TestTimeFormatter(t *testing.T) {
	c := qt.New(t)

	june06, _ := time.Parse("2006-Jan-02", "2018-Jun-06")
	june06 = june06.Add(7777 * time.Second)

	c.Run("Norsk nynorsk", func(c *qt.C) {
		f := NewTimeFormatter(translators.GetTranslator("nn"))

		c.Assert(f.Format(june06, "Monday Jan 2 2006"), qt.Equals, "onsdag juni 6 2018")
		c.Assert(f.Format(june06, "Mon January 2 2006"), qt.Equals, "on. juni 6 2018")
		c.Assert(f.Format(june06, "Mon Mon"), qt.Equals, "on. on.")
	})

	c.Run("Custom layouts Norsk nynorsk", func(c *qt.C) {
		f := NewTimeFormatter(translators.GetTranslator("nn"))

		c.Assert(f.Format(june06, ":date_full"), qt.Equals, "onsdag 6. juni 2018")
		c.Assert(f.Format(june06, ":date_long"), qt.Equals, "6. juni 2018")
		c.Assert(f.Format(june06, ":date_medium"), qt.Equals, "6. juni 2018")
		c.Assert(f.Format(june06, ":date_short"), qt.Equals, "06.06.2018")

		c.Assert(f.Format(june06, ":time_full"), qt.Equals, "kl. 02:09:37 UTC")
		c.Assert(f.Format(june06, ":time_long"), qt.Equals, "02:09:37 UTC")
		c.Assert(f.Format(june06, ":time_medium"), qt.Equals, "02:09:37")
		c.Assert(f.Format(june06, ":time_short"), qt.Equals, "02:09")

	})

	c.Run("Custom layouts English", func(c *qt.C) {
		f := NewTimeFormatter(translators.GetTranslator("en"))

		c.Assert(f.Format(june06, ":date_full"), qt.Equals, "Wednesday, June 6, 2018")
		c.Assert(f.Format(june06, ":date_long"), qt.Equals, "June 6, 2018")
		c.Assert(f.Format(june06, ":date_medium"), qt.Equals, "Jun 6, 2018")
		c.Assert(f.Format(june06, ":date_short"), qt.Equals, "6/6/18")

		c.Assert(f.Format(june06, ":time_full"), qt.Equals, "2:09:37 am UTC")
		c.Assert(f.Format(june06, ":time_long"), qt.Equals, "2:09:37 am UTC")
		c.Assert(f.Format(june06, ":time_medium"), qt.Equals, "2:09:37 am")
		c.Assert(f.Format(june06, ":time_short"), qt.Equals, "2:09 am")

	})

	c.Run("English", func(c *qt.C) {
		f := NewTimeFormatter(translators.GetTranslator("en"))

		c.Assert(f.Format(june06, "Monday Jan 2 2006"), qt.Equals, "Wednesday Jun 6 2018")
		c.Assert(f.Format(june06, "Mon January 2 2006"), qt.Equals, "Wed June 6 2018")
		c.Assert(f.Format(june06, "Mon Mon"), qt.Equals, "Wed Wed")
	})

}

func BenchmarkTimeFormatter(b *testing.B) {
	june06, _ := time.Parse("2006-Jan-02", "2018-Jun-06")

	b.Run("Native", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			got := june06.Format("Monday Jan 2 2006")
			if got != "Wednesday Jun 6 2018" {
				b.Fatalf("invalid format, got %q", got)
			}
		}
	})

	b.Run("Localized", func(b *testing.B) {
		f := NewTimeFormatter(translators.GetTranslator("nn"))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			got := f.Format(june06, "Monday Jan 2 2006")
			if got != "onsdag juni 6 2018" {
				b.Fatalf("invalid format, got %q", got)
			}
		}
	})

	b.Run("Localized Custom", func(b *testing.B) {
		f := NewTimeFormatter(translators.GetTranslator("nn"))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			got := f.Format(june06, ":date_medium")
			if got != "6. juni 2018" {
				b.Fatalf("invalid format, got %q", got)
			}
		}
	})
}
