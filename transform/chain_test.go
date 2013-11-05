package transform

import (
	"bytes"
	"testing"
)

const H5_JS_CONTENT_ABS_URL_WITH_NAV = "<!DOCTYPE html><html><head><script src=\"/foobar.js\"></script></head><body><nav><ul><li hugo-nav=\"section_0\"></li><li hugo-nav=\"section_1\"></li></ul></nav><article>content <a href=\"/foobar\">foobar</a>. Follow up</article></body></html>"

const CORRECT_OUTPUT_SRC_HREF_WITH_NAV = "<!DOCTYPE html><html><head><script src=\"http://two/foobar.js\"></script></head><body><nav><ul><li hugo-nav=\"section_0\"></li><li hugo-nav=\"section_1\"></li></ul></nav><article>content <a href=\"http://two/foobar\">foobar</a>. Follow up</article></body></html>"

var two_chain_tests = []test{
	{H5_JS_CONTENT_ABS_URL_WITH_NAV, CORRECT_OUTPUT_SRC_HREF_WITH_NAV},
}

func TestChainZeroTransformers(t *testing.T) {
	tr := NewChain()
	in := new(bytes.Buffer)
	out := new(bytes.Buffer)
	if err := tr.Apply(in, out); err != nil {
		t.Errorf("A zero transformer chain returned an error.")
	}
}

func BenchmarkChain(b *testing.B) {
	absURL, _ := AbsURL("http://two")
	tr := NewChain(absURL...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		apply(b.Errorf, tr, two_chain_tests)
	}
}
