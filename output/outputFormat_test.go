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

package output

import (
	"sort"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/gohugoio/hugo/media"
	"github.com/google/go-cmp/cmp"
)

var eq = qt.CmpEquals(
	cmp.Comparer(func(m1, m2 media.Type) bool {
		return m1.Type() == m2.Type()
	}),
	cmp.Comparer(func(o1, o2 Format) bool {
		return o1.Name == o2.Name
	}),
)

func TestDefaultTypes(t *testing.T) {
	c := qt.New(t)
	c.Assert(CalendarFormat.Name, qt.Equals, "Calendar")
	c.Assert(CalendarFormat.MediaType, eq, media.CalendarType)
	c.Assert(CalendarFormat.Protocol, qt.Equals, "webcal://")
	c.Assert(CalendarFormat.Path, qt.HasLen, 0)
	c.Assert(CalendarFormat.IsPlainText, qt.Equals, true)
	c.Assert(CalendarFormat.IsHTML, qt.Equals, false)

	c.Assert(CSSFormat.Name, qt.Equals, "CSS")
	c.Assert(CSSFormat.MediaType, eq, media.CSSType)
	c.Assert(CSSFormat.Path, qt.HasLen, 0)
	c.Assert(CSSFormat.Protocol, qt.HasLen, 0) // Will inherit the BaseURL protocol.
	c.Assert(CSSFormat.IsPlainText, qt.Equals, true)
	c.Assert(CSSFormat.IsHTML, qt.Equals, false)

	c.Assert(CSVFormat.Name, qt.Equals, "CSV")
	c.Assert(CSVFormat.MediaType, eq, media.CSVType)
	c.Assert(CSVFormat.Path, qt.HasLen, 0)
	c.Assert(CSVFormat.Protocol, qt.HasLen, 0)
	c.Assert(CSVFormat.IsPlainText, qt.Equals, true)
	c.Assert(CSVFormat.IsHTML, qt.Equals, false)
	c.Assert(CSVFormat.Permalinkable, qt.Equals, false)

	c.Assert(HTMLFormat.Name, qt.Equals, "HTML")
	c.Assert(HTMLFormat.MediaType, eq, media.HTMLType)
	c.Assert(HTMLFormat.Path, qt.HasLen, 0)
	c.Assert(HTMLFormat.Protocol, qt.HasLen, 0)
	c.Assert(HTMLFormat.IsPlainText, qt.Equals, false)
	c.Assert(HTMLFormat.IsHTML, qt.Equals, true)
	c.Assert(AMPFormat.Permalinkable, qt.Equals, true)

	c.Assert(AMPFormat.Name, qt.Equals, "AMP")
	c.Assert(AMPFormat.MediaType, eq, media.HTMLType)
	c.Assert(AMPFormat.Path, qt.Equals, "amp")
	c.Assert(AMPFormat.Protocol, qt.HasLen, 0)
	c.Assert(AMPFormat.IsPlainText, qt.Equals, false)
	c.Assert(AMPFormat.IsHTML, qt.Equals, true)
	c.Assert(AMPFormat.Permalinkable, qt.Equals, true)

	c.Assert(RSSFormat.Name, qt.Equals, "RSS")
	c.Assert(RSSFormat.MediaType, eq, media.RSSType)
	c.Assert(RSSFormat.Path, qt.HasLen, 0)
	c.Assert(RSSFormat.IsPlainText, qt.Equals, false)
	c.Assert(RSSFormat.NoUgly, qt.Equals, true)
	c.Assert(CalendarFormat.IsHTML, qt.Equals, false)

}

func TestGetFormatByName(t *testing.T) {
	c := qt.New(t)
	formats := Formats{AMPFormat, CalendarFormat}
	tp, _ := formats.GetByName("AMp")
	c.Assert(tp, eq, AMPFormat)
	_, found := formats.GetByName("HTML")
	c.Assert(found, qt.Equals, false)
	_, found = formats.GetByName("FOO")
	c.Assert(found, qt.Equals, false)
}

func TestGetFormatByExt(t *testing.T) {
	c := qt.New(t)
	formats1 := Formats{AMPFormat, CalendarFormat}
	formats2 := Formats{AMPFormat, HTMLFormat, CalendarFormat}
	tp, _ := formats1.GetBySuffix("html")
	c.Assert(tp, eq, AMPFormat)
	tp, _ = formats1.GetBySuffix("ics")
	c.Assert(tp, eq, CalendarFormat)
	_, found := formats1.GetBySuffix("not")
	c.Assert(found, qt.Equals, false)

	// ambiguous
	_, found = formats2.GetBySuffix("html")
	c.Assert(found, qt.Equals, false)
}

func TestGetFormatByFilename(t *testing.T) {
	c := qt.New(t)
	noExtNoDelimMediaType := media.TextType
	noExtNoDelimMediaType.Delimiter = ""

	noExtMediaType := media.TextType

	var (
		noExtDelimFormat = Format{
			Name:      "NEM",
			MediaType: noExtNoDelimMediaType,
			BaseName:  "_redirects",
		}
		noExt = Format{
			Name:      "NEX",
			MediaType: noExtMediaType,
			BaseName:  "next",
		}
	)

	formats := Formats{AMPFormat, HTMLFormat, noExtDelimFormat, noExt, CalendarFormat}
	f, found := formats.FromFilename("my.amp.html")
	c.Assert(found, qt.Equals, true)
	c.Assert(f, eq, AMPFormat)
	_, found = formats.FromFilename("my.ics")
	c.Assert(found, qt.Equals, true)
	f, found = formats.FromFilename("my.html")
	c.Assert(found, qt.Equals, true)
	c.Assert(f, eq, HTMLFormat)
	f, found = formats.FromFilename("my.nem")
	c.Assert(found, qt.Equals, true)
	c.Assert(f, eq, noExtDelimFormat)
	f, found = formats.FromFilename("my.nex")
	c.Assert(found, qt.Equals, true)
	c.Assert(f, eq, noExt)
	_, found = formats.FromFilename("my.css")
	c.Assert(found, qt.Equals, false)

}

func TestDecodeFormats(t *testing.T) {
	c := qt.New(t)

	mediaTypes := media.Types{media.JSONType, media.XMLType}

	var tests = []struct {
		name        string
		maps        []map[string]interface{}
		shouldError bool
		assert      func(t *testing.T, name string, f Formats)
	}{
		{
			"Redefine JSON",
			[]map[string]interface{}{
				{
					"JsON": map[string]interface{}{
						"baseName":    "myindex",
						"isPlainText": "false"}}},
			false,
			func(t *testing.T, name string, f Formats) {
				msg := qt.Commentf(name)
				c.Assert(len(f), qt.Equals, len(DefaultFormats), msg)
				json, _ := f.GetByName("JSON")
				c.Assert(json.BaseName, qt.Equals, "myindex")
				c.Assert(json.MediaType, eq, media.JSONType)
				c.Assert(json.IsPlainText, qt.Equals, false)

			}},
		{
			"Add XML format with string as mediatype",
			[]map[string]interface{}{
				{
					"MYXMLFORMAT": map[string]interface{}{
						"baseName":  "myxml",
						"mediaType": "application/xml",
					}}},
			false,
			func(t *testing.T, name string, f Formats) {
				c.Assert(len(f), qt.Equals, len(DefaultFormats)+1)
				xml, found := f.GetByName("MYXMLFORMAT")
				c.Assert(found, qt.Equals, true)
				c.Assert(xml.BaseName, qt.Equals, "myxml")
				c.Assert(xml.MediaType, eq, media.XMLType)

				// Verify that we haven't changed the DefaultFormats slice.
				json, _ := f.GetByName("JSON")
				c.Assert(json.BaseName, qt.Equals, "index")

			}},
		{
			"Add format unknown mediatype",
			[]map[string]interface{}{
				{
					"MYINVALID": map[string]interface{}{
						"baseName":  "mymy",
						"mediaType": "application/hugo",
					}}},
			true,
			func(t *testing.T, name string, f Formats) {

			}},
		{
			"Add and redefine XML format",
			[]map[string]interface{}{
				{
					"MYOTHERXMLFORMAT": map[string]interface{}{
						"baseName":  "myotherxml",
						"mediaType": media.XMLType,
					}},
				{
					"MYOTHERXMLFORMAT": map[string]interface{}{
						"baseName": "myredefined",
					}},
			},
			false,
			func(t *testing.T, name string, f Formats) {
				c.Assert(len(f), qt.Equals, len(DefaultFormats)+1)
				xml, found := f.GetByName("MYOTHERXMLFORMAT")
				c.Assert(found, qt.Equals, true)
				c.Assert(xml.BaseName, qt.Equals, "myredefined")
				c.Assert(xml.MediaType, eq, media.XMLType)
			}},
	}

	for _, test := range tests {
		result, err := DecodeFormats(mediaTypes, test.maps...)
		msg := qt.Commentf(test.name)

		if test.shouldError {
			c.Assert(err, qt.Not(qt.IsNil), msg)
		} else {
			c.Assert(err, qt.IsNil, msg)
			test.assert(t, test.name, result)
		}
	}
}

func TestSort(t *testing.T) {
	c := qt.New(t)
	c.Assert(DefaultFormats[0].Name, qt.Equals, "HTML")
	c.Assert(DefaultFormats[1].Name, qt.Equals, "AMP")

	json := JSONFormat
	json.Weight = 1

	formats := Formats{
		AMPFormat,
		HTMLFormat,
		json,
	}

	sort.Sort(formats)

	c.Assert(formats[0].Name, qt.Equals, "JSON")
	c.Assert(formats[1].Name, qt.Equals, "HTML")
	c.Assert(formats[2].Name, qt.Equals, "AMP")

}
